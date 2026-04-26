package monitoring

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func BuildPlan(ctx context.Context, client *ClusterClient, cfg Config) (Plan, error) {
	if cfg.Deployment == "" {
		return Plan{}, fmt.Errorf("deployment is required")
	}

	deployment, err := GetDeployment(ctx, client, cfg.Namespace, cfg.Deployment)
	if err != nil {
		return Plan{}, err
	}

	metricsPort, metricsPortSource := resolveMetricsPort(*deployment, cfg.MetricsPort)
	metricsPath := strings.TrimSpace(cfg.MetricsPath)
	if metricsPath == "" {
		metricsPath = DefaultMetricsPath
	}
	scrapeInterval := strings.TrimSpace(cfg.ScrapeInterval)
	if scrapeInterval == "" {
		scrapeInterval = DefaultScrapeInterval
	}

	service, err := FindServiceForDeployment(ctx, client, cfg.Namespace, deployment)
	if err != nil {
		return Plan{}, err
	}

	operatorInstalled, err := HasPrometheusOperator(ctx, client)
	if err != nil {
		return Plan{}, err
	}

	serviceName := metricsServiceName(deployment.Name)
	serviceExists := service != nil
	servicePortName := "metrics"
	if serviceExists {
		serviceName = service.Name
		servicePortName = servicePortNameForMetrics(*service, metricsPort)
	}
	serviceMonitorName := serviceMonitorName(deployment.Name)

	plan := Plan{
		Plugin:              PluginID,
		Namespace:           cfg.Namespace,
		Deployment:          deployment.Name,
		MetricsPath:         metricsPath,
		MetricsPort:         metricsPort,
		MetricsPortSource:   metricsPortSource,
		ScrapeInterval:      scrapeInterval,
		HasScrapeAnnotation: hasScrapeAnnotation(*deployment),
		ServiceExists:       serviceExists,
		ServiceName:         serviceName,
		ServicePortName:     servicePortName,
		PrometheusOperator:  operatorInstalled,
	}
	if operatorInstalled {
		plan.ServiceMonitorName = serviceMonitorName
	}
	if cfg.InstallPrometheus {
		plan.ManagedPrometheus = managedPrometheusPlan(cfg)
	}

	if deploymentNeedsAnnotationUpdate(*deployment, metricsPath, metricsPort) {
		plan.Operations = append(plan.Operations, Operation{
			Action:    "update",
			Kind:      "Deployment",
			Namespace: cfg.Namespace,
			Name:      deployment.Name,
			Reason:    "configure Prometheus scrape annotations on Deployment and pod template",
			Details: map[string]string{
				PrometheusScrapeAnnotation: "true",
				PrometheusPortAnnotation:   strconv.Itoa(metricsPort),
				PrometheusPathAnnotation:   metricsPath,
			},
		})
	} else {
		plan.Operations = append(plan.Operations, Operation{
			Action:    "noop",
			Kind:      "Deployment",
			Namespace: cfg.Namespace,
			Name:      deployment.Name,
			Reason:    "Prometheus scrape annotations are already configured",
		})
	}

	if !serviceExists {
		plan.Operations = append(plan.Operations, Operation{
			Action:    "create",
			Kind:      "Service",
			Namespace: cfg.Namespace,
			Name:      serviceName,
			Reason:    "no Service selects pods from the Deployment",
			Details: map[string]string{
				"port":       strconv.Itoa(metricsPort),
				"targetPort": strconv.Itoa(metricsPort),
				"portName":   servicePortName,
			},
		})
	} else if serviceNeedsUpdate(*service, deployment.Name, metricsPort) {
		plan.Operations = append(plan.Operations, Operation{
			Action:    "update",
			Kind:      "Service",
			Namespace: cfg.Namespace,
			Name:      serviceName,
			Reason:    "ensure Service exposes the metrics endpoint and has a stable selector label for ServiceMonitor",
			Details: map[string]string{
				"port":       strconv.Itoa(metricsPort),
				"targetPort": strconv.Itoa(metricsPort),
				"portName":   servicePortName,
			},
		})
	} else {
		plan.Operations = append(plan.Operations, Operation{
			Action:    "noop",
			Kind:      "Service",
			Namespace: cfg.Namespace,
			Name:      serviceName,
			Reason:    "Service already exposes the metrics endpoint",
		})
	}

	if operatorInstalled {
		plan.Operations = append(plan.Operations, Operation{
			Action:    "apply",
			Kind:      ServiceMonitorKind,
			Namespace: cfg.Namespace,
			Name:      serviceMonitorName,
			Reason:    "Prometheus Operator is installed",
			Details: map[string]string{
				"service":  serviceName,
				"portName": servicePortName,
				"path":     metricsPath,
				"interval": scrapeInterval,
			},
		})
	} else {
		plan.Operations = append(plan.Operations, Operation{
			Action: "skip",
			Kind:   ServiceMonitorKind,
			Name:   serviceMonitorName,
			Reason: "Prometheus Operator CRD servicemonitors.monitoring.coreos.com is not installed",
		})
	}

	if plan.ManagedPrometheus.Enabled {
		plan.Operations = append(plan.Operations, Operation{
			Action:    "apply",
			Kind:      "PrometheusStack",
			Namespace: plan.ManagedPrometheus.Namespace,
			Name:      plan.ManagedPrometheus.Name,
			Reason:    "ensure a Prometheus server with UI is available and scrapes services configured by this plugin",
			Details: map[string]string{
				"image":       plan.ManagedPrometheus.Image,
				"service":     plan.ManagedPrometheus.ServiceName,
				"portForward": plan.ManagedPrometheus.PortForward,
			},
		})
	}

	return plan, nil
}

func resolveMetricsPort(deployment appsv1.Deployment, override int) (int, string) {
	if override > 0 {
		return override, "user"
	}
	if port, ok := findNamedContainerPort(deployment, "metrics"); ok {
		return port, "containerPort named metrics"
	}
	if port, ok := findContainerPort(deployment, 8080); ok {
		return port, "containerPort 8080"
	}
	return DefaultMetricsPort, "default"
}

func findNamedContainerPort(deployment appsv1.Deployment, name string) (int, bool) {
	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, port := range container.Ports {
			if port.Name == name && port.ContainerPort > 0 {
				return int(port.ContainerPort), true
			}
		}
	}
	return 0, false
}

func findContainerPort(deployment appsv1.Deployment, target int) (int, bool) {
	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, port := range container.Ports {
			if int(port.ContainerPort) == target {
				return target, true
			}
		}
	}
	return 0, false
}

func hasScrapeAnnotation(deployment appsv1.Deployment) bool {
	return deployment.Annotations[PrometheusScrapeAnnotation] == "true" ||
		deployment.Spec.Template.Annotations[PrometheusScrapeAnnotation] == "true"
}

func deploymentNeedsAnnotationUpdate(deployment appsv1.Deployment, path string, port int) bool {
	expectedPort := strconv.Itoa(port)
	return deployment.Annotations[PrometheusScrapeAnnotation] != "true" ||
		deployment.Annotations[PrometheusPortAnnotation] != expectedPort ||
		deployment.Annotations[PrometheusPathAnnotation] != path ||
		deployment.Spec.Template.Annotations[PrometheusScrapeAnnotation] != "true" ||
		deployment.Spec.Template.Annotations[PrometheusPortAnnotation] != expectedPort ||
		deployment.Spec.Template.Annotations[PrometheusPathAnnotation] != path
}

func serviceSelectsDeployment(service corev1.Service, deployment appsv1.Deployment) bool {
	if len(service.Spec.Selector) == 0 {
		return false
	}
	podLabels := deployment.Spec.Template.Labels
	for key, value := range service.Spec.Selector {
		if podLabels[key] != value {
			return false
		}
	}
	return true
}

func serviceNeedsUpdate(service corev1.Service, deploymentName string, metricsPort int) bool {
	return service.Labels[PrometheusTargetLabel] != deploymentName ||
		service.Labels[PluginIDLabel] != PluginID ||
		service.Annotations[PrometheusScrapeAnnotation] != "true" ||
		service.Annotations[PrometheusPortAnnotation] != strconv.Itoa(metricsPort) ||
		!serviceHasMetricsPort(service, metricsPort)
}

func serviceHasMetricsPort(service corev1.Service, metricsPort int) bool {
	for _, port := range service.Spec.Ports {
		if servicePortTargetsMetrics(port, metricsPort) {
			return true
		}
	}
	return false
}

func servicePortNameForMetrics(service corev1.Service, metricsPort int) string {
	for _, port := range service.Spec.Ports {
		if port.Name != "" && (port.Name == "metrics" || servicePortTargetsMetrics(port, metricsPort)) {
			return port.Name
		}
	}
	return "metrics"
}

func servicePortTargetsMetrics(port corev1.ServicePort, metricsPort int) bool {
	if port.TargetPort.Type == intstr.String {
		return port.TargetPort.StrVal == "metrics"
	}
	if port.TargetPort.IntVal > 0 {
		return port.TargetPort.IntVal == int32(metricsPort)
	}
	return int(port.Port) == metricsPort
}

func metricsServiceName(deploymentName string) string {
	return deploymentName + "-metrics"
}

func serviceMonitorName(deploymentName string) string {
	return deploymentName + "-prometheus"
}

func managedPrometheusPlan(cfg Config) ManagedPrometheus {
	serviceName := cfg.PrometheusName
	return ManagedPrometheus{
		Enabled:     true,
		Namespace:   cfg.PrometheusNamespace,
		Name:        cfg.PrometheusName,
		Image:       cfg.PrometheusImage,
		ServiceName: serviceName,
		UIURL:       "http://localhost:9090",
		PortForward: fmt.Sprintf("kubectl port-forward -n %s svc/%s 9090:9090", cfg.PrometheusNamespace, serviceName),
	}
}
