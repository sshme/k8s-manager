package tracing

import (
	"context"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Apply(ctx context.Context, client *ClusterClient, plan Plan) error {
	deployment, err := GetDeployment(ctx, client, plan.Namespace, plan.Deployment)
	if err != nil {
		return err
	}
	if err := applyDeploymentTracingEnv(ctx, client, deployment, plan); err != nil {
		return err
	}
	if plan.ManagedJaeger.Enabled {
		if err := applyManagedJaeger(ctx, client, plan.ManagedJaeger); err != nil {
			return err
		}
	}
	return nil
}

func applyDeploymentTracingEnv(ctx context.Context, client *ClusterClient, deployment *appsv1.Deployment, plan Plan) error {
	next := deployment.DeepCopy()
	targets := make(map[string]struct{}, len(plan.TargetContainers))
	for _, name := range plan.TargetContainers {
		targets[name] = struct{}{}
	}

	for i := range next.Spec.Template.Spec.Containers {
		container := &next.Spec.Template.Spec.Containers[i]
		if _, ok := targets[container.Name]; ok {
			container.Env = mergeEnv(container.Env, plan.Env)
		}
	}

	if _, err := client.Kubernetes.AppsV1().Deployments(plan.Namespace).Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update deployment tracing env: %w", err)
	}
	return nil
}

func mergeEnv(current []corev1.EnvVar, values map[string]string) []corev1.EnvVar {
	result := make([]corev1.EnvVar, 0, len(current)+len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range current {
		if value, ok := values[item.Name]; ok {
			result = append(result, corev1.EnvVar{Name: item.Name, Value: value})
			seen[item.Name] = struct{}{}
			continue
		}
		result = append(result, item)
	}

	keys := make([]string, 0, len(values)-len(seen))
	for key := range values {
		if _, ok := seen[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		result = append(result, corev1.EnvVar{Name: key, Value: values[key]})
	}
	return result
}

func applyManagedJaeger(ctx context.Context, client *ClusterClient, jaeger ManagedJaeger) error {
	if err := ensureNamespace(ctx, client, jaeger); err != nil {
		return err
	}
	if err := ensureJaegerDeployment(ctx, client, jaeger); err != nil {
		return err
	}
	if err := ensureJaegerService(ctx, client, jaeger); err != nil {
		return err
	}
	return nil
}

func ensureNamespace(ctx context.Context, client *ClusterClient, jaeger ManagedJaeger) error {
	_, err := client.Kubernetes.CoreV1().Namespaces().Get(ctx, jaeger.Namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get namespace %s: %w", jaeger.Namespace, err)
	}
	_, err = client.Kubernetes.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   jaeger.Namespace,
			Labels: commonJaegerLabels(jaeger.Name),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create namespace %s: %w", jaeger.Namespace, err)
	}
	return nil
}

func ensureJaegerDeployment(ctx context.Context, client *ClusterClient, jaeger ManagedJaeger) error {
	labels := commonJaegerLabels(jaeger.Name)
	replicas := int32(1)
	desired := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": jaeger.Name}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "jaeger",
							Image: jaeger.Image,
							Env: []corev1.EnvVar{
								{Name: "COLLECTOR_OTLP_ENABLED", Value: "true"},
							},
							Args: []string{"--memory.max-traces=50000"},
							Ports: []corev1.ContainerPort{
								{Name: "ui", ContainerPort: 16686},
								{Name: "otlp-grpc", ContainerPort: 4317},
								{Name: "otlp-http", ContainerPort: 4318},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	current, err := client.Kubernetes.AppsV1().Deployments(jaeger.Namespace).Get(ctx, jaeger.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.Kubernetes.AppsV1().Deployments(jaeger.Namespace).Create(ctx, desired, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create Jaeger Deployment: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get Jaeger Deployment: %w", err)
	}
	desired.ResourceVersion = current.ResourceVersion
	if _, err := client.Kubernetes.AppsV1().Deployments(jaeger.Namespace).Update(ctx, desired, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update Jaeger Deployment: %w", err)
	}
	return nil
}

func ensureJaegerService(ctx context.Context, client *ClusterClient, jaeger ManagedJaeger) error {
	current, err := client.Kubernetes.CoreV1().Services(jaeger.Namespace).Get(ctx, jaeger.ServiceName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.Kubernetes.CoreV1().Services(jaeger.Namespace).Create(ctx, jaegerService(jaeger), metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create Jaeger Service: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get Jaeger Service: %w", err)
	}

	next := current.DeepCopy()
	next.Labels = commonJaegerLabels(jaeger.Name)
	next.Spec.Type = corev1.ServiceTypeClusterIP
	next.Spec.Selector = map[string]string{"app.kubernetes.io/name": jaeger.Name}
	next.Spec.Ports = jaegerServicePorts()
	if _, err := client.Kubernetes.CoreV1().Services(jaeger.Namespace).Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update Jaeger Service: %w", err)
	}
	return nil
}

func jaegerService(jaeger ManagedJaeger) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaeger.ServiceName,
			Namespace: jaeger.Namespace,
			Labels:    commonJaegerLabels(jaeger.Name),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app.kubernetes.io/name": jaeger.Name},
			Ports:    jaegerServicePorts(),
		},
	}
}

func jaegerServicePorts() []corev1.ServicePort {
	return []corev1.ServicePort{
		{Name: "ui", Port: 16686, TargetPort: intstr.FromString("ui")},
		{Name: "otlp-grpc", Port: 4317, TargetPort: intstr.FromString("otlp-grpc")},
		{Name: "otlp-http", Port: 4318, TargetPort: intstr.FromString("otlp-http")},
	}
}

func commonJaegerLabels(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      name,
		"app.kubernetes.io/component": "tracing",
		ManagedByLabel:                "k8s-manager",
		PluginIDLabel:                 PluginID,
	}
}
