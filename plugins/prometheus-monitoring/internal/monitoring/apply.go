package monitoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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
	if plan.ManagedPrometheus.Enabled {
		if err := applyManagedPrometheus(ctx, client, plan); err != nil {
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
			Name:        plan.ServiceName,
			Namespace:   plan.Namespace,
			Labels:      serviceLabels(plan.Deployment),
			Annotations: serviceAnnotations(plan),
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
	if next.Annotations == nil {
		next.Annotations = map[string]string{}
	}
	for key, value := range serviceLabels(plan.Deployment) {
		next.Labels[key] = value
	}
	for key, value := range serviceAnnotations(plan) {
		next.Annotations[key] = value
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

func serviceAnnotations(plan Plan) map[string]string {
	return map[string]string{
		PrometheusScrapeAnnotation: "true",
		PrometheusPortAnnotation:   strconv.Itoa(plan.MetricsPort),
		PrometheusPathAnnotation:   plan.MetricsPath,
	}
}

func serviceLabelsAny(deploymentName string) map[string]any {
	result := make(map[string]any, len(serviceLabels(deploymentName)))
	for key, value := range serviceLabels(deploymentName) {
		result[key] = value
	}
	return result
}

func applyManagedPrometheus(ctx context.Context, client *ClusterClient, plan Plan) error {
	prom := plan.ManagedPrometheus
	if err := ensureNamespace(ctx, client, prom); err != nil {
		return err
	}
	if err := ensurePrometheusServiceAccount(ctx, client, prom); err != nil {
		return err
	}
	if err := ensurePrometheusClusterRole(ctx, client, prom); err != nil {
		return err
	}
	if err := ensurePrometheusClusterRoleBinding(ctx, client, prom); err != nil {
		return err
	}
	if err := ensurePrometheusConfigMap(ctx, client, plan); err != nil {
		return err
	}
	if err := ensurePrometheusDeployment(ctx, client, plan); err != nil {
		return err
	}
	if err := ensurePrometheusService(ctx, client, prom); err != nil {
		return err
	}
	return nil
}

func ensureNamespace(ctx context.Context, client *ClusterClient, prom ManagedPrometheus) error {
	namespace := prom.Namespace
	_, err := client.Kubernetes.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get namespace %s: %w", namespace, err)
	}
	_, err = client.Kubernetes.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: commonPrometheusLabels(prom.Name),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create namespace %s: %w", namespace, err)
	}
	return nil
}

func ensurePrometheusServiceAccount(ctx context.Context, client *ClusterClient, prom ManagedPrometheus) error {
	desired := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prom.Name,
			Namespace: prom.Namespace,
			Labels:    commonPrometheusLabels(prom.Name),
		},
	}
	current, err := client.Kubernetes.CoreV1().ServiceAccounts(prom.Namespace).Get(ctx, prom.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.Kubernetes.CoreV1().ServiceAccounts(prom.Namespace).Create(ctx, desired, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create Prometheus ServiceAccount: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get Prometheus ServiceAccount: %w", err)
	}
	desired.ResourceVersion = current.ResourceVersion
	if _, err := client.Kubernetes.CoreV1().ServiceAccounts(prom.Namespace).Update(ctx, desired, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update Prometheus ServiceAccount: %w", err)
	}
	return nil
}

func ensurePrometheusClusterRole(ctx context.Context, client *ClusterClient, prom ManagedPrometheus) error {
	desired := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   prom.Name,
			Labels: commonPrometheusLabels(prom.Name),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes", "nodes/metrics", "services", "endpoints", "pods", "namespaces"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	current, err := client.Kubernetes.RbacV1().ClusterRoles().Get(ctx, prom.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.Kubernetes.RbacV1().ClusterRoles().Create(ctx, desired, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create Prometheus ClusterRole: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get Prometheus ClusterRole: %w", err)
	}
	desired.ResourceVersion = current.ResourceVersion
	if _, err := client.Kubernetes.RbacV1().ClusterRoles().Update(ctx, desired, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update Prometheus ClusterRole: %w", err)
	}
	return nil
}

func ensurePrometheusClusterRoleBinding(ctx context.Context, client *ClusterClient, prom ManagedPrometheus) error {
	desired := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   prom.Name,
			Labels: commonPrometheusLabels(prom.Name),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     prom.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      prom.Name,
				Namespace: prom.Namespace,
			},
		},
	}
	current, err := client.Kubernetes.RbacV1().ClusterRoleBindings().Get(ctx, prom.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.Kubernetes.RbacV1().ClusterRoleBindings().Create(ctx, desired, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create Prometheus ClusterRoleBinding: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get Prometheus ClusterRoleBinding: %w", err)
	}
	desired.ResourceVersion = current.ResourceVersion
	if _, err := client.Kubernetes.RbacV1().ClusterRoleBindings().Update(ctx, desired, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update Prometheus ClusterRoleBinding: %w", err)
	}
	return nil
}

func ensurePrometheusConfigMap(ctx context.Context, client *ClusterClient, plan Plan) error {
	prom := plan.ManagedPrometheus
	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prom.Name,
			Namespace: prom.Namespace,
			Labels:    commonPrometheusLabels(prom.Name),
		},
		Data: map[string]string{
			"prometheus.yml": prometheusConfig(plan.ScrapeInterval),
		},
	}
	current, err := client.Kubernetes.CoreV1().ConfigMaps(prom.Namespace).Get(ctx, prom.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.Kubernetes.CoreV1().ConfigMaps(prom.Namespace).Create(ctx, desired, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create Prometheus ConfigMap: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get Prometheus ConfigMap: %w", err)
	}
	desired.ResourceVersion = current.ResourceVersion
	if _, err := client.Kubernetes.CoreV1().ConfigMaps(prom.Namespace).Update(ctx, desired, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update Prometheus ConfigMap: %w", err)
	}
	return nil
}

func ensurePrometheusDeployment(ctx context.Context, client *ClusterClient, plan Plan) error {
	prom := plan.ManagedPrometheus
	labels := commonPrometheusLabels(prom.Name)
	replicas := int32(1)
	annotations := map[string]string{
		"k8s-manager.io/config-sha256": prometheusConfigHash(plan.ScrapeInterval),
	}
	desired := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prom.Name,
			Namespace: prom.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": prom.Name}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: prom.Name,
					Containers: []corev1.Container{
						{
							Name:  "prometheus",
							Image: prom.Image,
							Args: []string{
								"--config.file=/etc/prometheus/prometheus.yml",
								"--storage.tsdb.path=/prometheus",
								"--web.enable-lifecycle",
							},
							Ports: []corev1.ContainerPort{
								{Name: "web", ContainerPort: 9090},
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
							VolumeMounts: []corev1.VolumeMount{
								{Name: "config", MountPath: "/etc/prometheus"},
								{Name: "data", MountPath: "/prometheus"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: prom.Name},
						}}},
						{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
					},
				},
			},
		},
	}
	current, err := client.Kubernetes.AppsV1().Deployments(prom.Namespace).Get(ctx, prom.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.Kubernetes.AppsV1().Deployments(prom.Namespace).Create(ctx, desired, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create Prometheus Deployment: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get Prometheus Deployment: %w", err)
	}
	desired.ResourceVersion = current.ResourceVersion
	if _, err := client.Kubernetes.AppsV1().Deployments(prom.Namespace).Update(ctx, desired, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update Prometheus Deployment: %w", err)
	}
	return nil
}

func ensurePrometheusService(ctx context.Context, client *ClusterClient, prom ManagedPrometheus) error {
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prom.ServiceName,
			Namespace: prom.Namespace,
			Labels:    commonPrometheusLabels(prom.Name),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app.kubernetes.io/name": prom.Name},
			Ports: []corev1.ServicePort{
				{Name: "web", Port: 9090, TargetPort: intstr.FromString("web")},
			},
		},
	}
	current, err := client.Kubernetes.CoreV1().Services(prom.Namespace).Get(ctx, prom.ServiceName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.Kubernetes.CoreV1().Services(prom.Namespace).Create(ctx, desired, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create Prometheus Service: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get Prometheus Service: %w", err)
	}
	next := current.DeepCopy()
	next.Labels = commonPrometheusLabels(prom.Name)
	next.Spec.Type = corev1.ServiceTypeClusterIP
	next.Spec.Selector = map[string]string{"app.kubernetes.io/name": prom.Name}
	next.Spec.Ports = []corev1.ServicePort{
		{Name: "web", Port: 9090, TargetPort: intstr.FromString("web")},
	}
	if _, err := client.Kubernetes.CoreV1().Services(prom.Namespace).Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update Prometheus Service: %w", err)
	}
	return nil
}

func commonPrometheusLabels(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      name,
		"app.kubernetes.io/component": "prometheus",
		ManagedByLabel:                "k8s-manager",
		PluginIDLabel:                 PluginID,
	}
}

func prometheusConfig(scrapeInterval string) string {
	return fmt.Sprintf(`global:
  scrape_interval: %s
  evaluation_interval: %s

scrape_configs:
  - job_name: k8s-manager-prometheus
    static_configs:
      - targets:
          - localhost:9090

  - job_name: k8s-manager-managed-services
    kubernetes_sd_configs:
      - role: endpoints
    relabel_configs:
      - action: keep
        source_labels:
          - __meta_kubernetes_service_label_k8s_manager_io_plugin
        regex: %s
      - action: keep
        source_labels:
          - __meta_kubernetes_service_annotation_prometheus_io_scrape
        regex: "true"
      - action: replace
        source_labels:
          - __meta_kubernetes_service_annotation_prometheus_io_path
        target_label: __metrics_path__
        regex: (.+)
      - action: replace
        source_labels:
          - __address__
          - __meta_kubernetes_service_annotation_prometheus_io_port
        target_label: __address__
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: ${1}:${2}
      - action: replace
        source_labels:
          - __meta_kubernetes_namespace
        target_label: namespace
      - action: replace
        source_labels:
          - __meta_kubernetes_service_name
        target_label: service
      - action: replace
        source_labels:
          - __meta_kubernetes_pod_name
        target_label: pod
`, scrapeInterval, scrapeInterval, PluginID)
}

func prometheusConfigHash(scrapeInterval string) string {
	sum := sha256.Sum256([]byte(prometheusConfig(scrapeInterval)))
	return hex.EncodeToString(sum[:])
}

func copyStringMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
