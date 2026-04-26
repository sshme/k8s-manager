package monitoring

import (
	"context"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Apply(ctx context.Context, client *ClusterClient, plan Plan) error {
	deployment, err := GetDeployment(ctx, client, plan.Namespace, plan.Deployment)
	if err != nil {
		return err
	}
	if err := applyDeploymentAnnotations(ctx, client, deployment, plan); err != nil {
		return err
	}

	service, err := FindServiceForDeployment(ctx, client, plan.Namespace, deployment)
	if err != nil {
		return err
	}
	if service == nil {
		if _, err := createMetricsService(ctx, client, deployment, plan); err != nil {
			return err
		}
	} else if err := updateMetricsService(ctx, client, service, plan); err != nil {
		return err
	}

	if plan.PrometheusOperator {
		if err := applyServiceMonitor(ctx, client, plan); err != nil {
			return err
		}
	}
	return nil
}

func applyDeploymentAnnotations(ctx context.Context, client *ClusterClient, deployment *appsv1.Deployment, plan Plan) error {
	next := deployment.DeepCopy()
	ensureAnnotations(&next.ObjectMeta, plan)
	ensureAnnotations(&next.Spec.Template.ObjectMeta, plan)

	if _, err := client.Kubernetes.AppsV1().Deployments(plan.Namespace).Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update deployment annotations: %w", err)
	}
	return nil
}

func ensureAnnotations(meta *metav1.ObjectMeta, plan Plan) {
	if meta.Annotations == nil {
		meta.Annotations = map[string]string{}
	}
	meta.Annotations[PrometheusScrapeAnnotation] = "true"
	meta.Annotations[PrometheusPortAnnotation] = strconv.Itoa(plan.MetricsPort)
	meta.Annotations[PrometheusPathAnnotation] = plan.MetricsPath
}

func createMetricsService(ctx context.Context, client *ClusterClient, deployment *appsv1.Deployment, plan Plan) (*corev1.Service, error) {
	selector := deployment.Spec.Selector.MatchLabels
	if len(selector) == 0 {
		return nil, fmt.Errorf("deployment %s/%s has no matchLabels selector; cannot create Service", plan.Namespace, plan.Deployment)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      plan.ServiceName,
			Namespace: plan.Namespace,
			Labels:    serviceLabels(plan.Deployment),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: copyStringMap(selector),
			Ports: []corev1.ServicePort{
				metricsServicePort(plan.ServicePortName, plan.MetricsPort),
			},
		},
	}

	created, err := client.Kubernetes.CoreV1().Services(plan.Namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create metrics service: %w", err)
	}
	return created, nil
}

func updateMetricsService(ctx context.Context, client *ClusterClient, service *corev1.Service, plan Plan) error {
	next := service.DeepCopy()
	if next.Labels == nil {
		next.Labels = map[string]string{}
	}
	for key, value := range serviceLabels(plan.Deployment) {
		next.Labels[key] = value
	}
	ensureServicePort(next, plan.ServicePortName, plan.MetricsPort)

	if _, err := client.Kubernetes.CoreV1().Services(plan.Namespace).Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update metrics service: %w", err)
	}
	return nil
}

func ensureServicePort(service *corev1.Service, portName string, metricsPort int) {
	for i := range service.Spec.Ports {
		port := &service.Spec.Ports[i]
		if servicePortTargetsMetrics(*port, metricsPort) {
			if port.Name == "" {
				port.Name = portName
			}
			port.Protocol = corev1.ProtocolTCP
			return
		}
		if port.Name == portName && port.Name == "metrics" {
			port.Port = int32(metricsPort)
			port.TargetPort = intstr.FromInt32(int32(metricsPort))
			port.Protocol = corev1.ProtocolTCP
			return
		}
	}
	service.Spec.Ports = append(service.Spec.Ports, metricsServicePort(portName, metricsPort))
}

func metricsServicePort(name string, metricsPort int) corev1.ServicePort {
	return corev1.ServicePort{
		Name:       name,
		Protocol:   corev1.ProtocolTCP,
		Port:       int32(metricsPort),
		TargetPort: intstr.FromInt32(int32(metricsPort)),
	}
}

func applyServiceMonitor(ctx context.Context, client *ClusterClient, plan Plan) error {
	desired := serviceMonitorObject(plan)
	current, err := client.Dynamic.Resource(client.MonitorGVR).Namespace(plan.Namespace).Get(ctx, plan.ServiceMonitorName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, createErr := client.Dynamic.Resource(client.MonitorGVR).Namespace(plan.Namespace).Create(ctx, desired, metav1.CreateOptions{}); createErr != nil {
			return fmt.Errorf("create ServiceMonitor: %w", createErr)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get ServiceMonitor: %w", err)
	}

	desired.SetResourceVersion(current.GetResourceVersion())
	if _, err := client.Dynamic.Resource(client.MonitorGVR).Namespace(plan.Namespace).Update(ctx, desired, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update ServiceMonitor: %w", err)
	}
	return nil
}

func serviceMonitorObject(plan Plan) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": ServiceMonitorAPIGroup + "/" + ServiceMonitorAPIVersion,
			"kind":       ServiceMonitorKind,
			"metadata": map[string]any{
				"name":      plan.ServiceMonitorName,
				"namespace": plan.Namespace,
				"labels":    serviceLabelsAny(plan.Deployment),
			},
			"spec": map[string]any{
				"selector": map[string]any{
					"matchLabels": map[string]any{
						PrometheusTargetLabel: plan.Deployment,
						PluginIDLabel:         PluginID,
					},
				},
				"namespaceSelector": map[string]any{
					"matchNames": []any{plan.Namespace},
				},
				"endpoints": []any{
					map[string]any{
						"port":     plan.ServicePortName,
						"path":     plan.MetricsPath,
						"interval": plan.ScrapeInterval,
					},
				},
			},
		},
	}
}

func serviceLabels(deploymentName string) map[string]string {
	return map[string]string{
		ManagedByLabel:        "k8s-manager",
		PluginIDLabel:         PluginID,
		PrometheusTargetLabel: deploymentName,
	}
}

func serviceLabelsAny(deploymentName string) map[string]any {
	result := make(map[string]any, len(serviceLabels(deploymentName)))
	for key, value := range serviceLabels(deploymentName) {
		result[key] = value
	}
	return result
}

func copyStringMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
