package tracing

import (
	"context"
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func BuildPlan(ctx context.Context, client *ClusterClient, cfg Config) (Plan, error) {
	if cfg.Deployment == "" {
		return Plan{}, fmt.Errorf("deployment is required")
	}

	deployment, err := GetDeployment(ctx, client, cfg.Namespace, cfg.Deployment)
	if err != nil {
		return Plan{}, err
	}

	protocol, err := normalizeProtocol(cfg.Protocol)
	if err != nil {
		return Plan{}, err
	}
	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = deployment.Name
	}
	targetContainers, err := resolveTargetContainers(*deployment, cfg.Container)
	if err != nil {
		return Plan{}, err
	}

	collectorEndpoint := strings.TrimSpace(cfg.CollectorEndpoint)
	if collectorEndpoint == "" && cfg.InstallJaeger {
		collectorEndpoint = managedCollectorEndpoint(cfg.JaegerName, cfg.JaegerNamespace, protocol)
	}
	if collectorEndpoint == "" {
		return Plan{}, fmt.Errorf("--collector-endpoint is required when --skip-jaeger-install is used")
	}

	samplingRatio := strings.TrimSpace(cfg.SamplingRatio)
	if samplingRatio == "" {
		samplingRatio = DefaultSamplingRatio
	}
	resourceAttrs := resourceAttributes(cfg.Namespace, deployment.Name, serviceName)
	env := tracingEnv(serviceName, collectorEndpoint, protocol, samplingRatio, resourceAttrs)

	plan := Plan{
		Plugin:            PluginID,
		Namespace:         cfg.Namespace,
		Deployment:        deployment.Name,
		TargetContainers:  targetContainers,
		ServiceName:       serviceName,
		CollectorEndpoint: collectorEndpoint,
		Protocol:          protocol,
		SamplingRatio:     samplingRatio,
		ResourceAttrs:     resourceAttrs,
		DeploymentReady:   deploymentHasEnv(*deployment, targetContainers, env),
		Env:               env,
	}
	if cfg.InstallJaeger {
		plan.ManagedJaeger = managedJaegerPlan(cfg, protocol)
	}

	if plan.DeploymentReady {
		plan.Operations = append(plan.Operations, Operation{
			Action:    "noop",
			Kind:      "Deployment",
			Namespace: cfg.Namespace,
			Name:      deployment.Name,
			Reason:    "OpenTelemetry tracing environment is already configured",
			Details: map[string]string{
				"containers": strings.Join(targetContainers, ","),
			},
		})
	} else {
		plan.Operations = append(plan.Operations, Operation{
			Action:    "update",
			Kind:      "Deployment",
			Namespace: cfg.Namespace,
			Name:      deployment.Name,
			Reason:    "inject OpenTelemetry OTLP environment variables into target containers",
			Details: map[string]string{
				"containers":        strings.Join(targetContainers, ","),
				"collectorEndpoint": collectorEndpoint,
				"protocol":          protocol,
				"serviceName":       serviceName,
			},
		})
	}

	if plan.ManagedJaeger.Enabled {
		plan.Operations = append(plan.Operations, Operation{
			Action:    "apply",
			Kind:      "JaegerStack",
			Namespace: plan.ManagedJaeger.Namespace,
			Name:      plan.ManagedJaeger.Name,
			Reason:    "ensure a managed Jaeger collector and UI are available for OTLP traces",
			Details: map[string]string{
				"image":       plan.ManagedJaeger.Image,
				"service":     plan.ManagedJaeger.ServiceName,
				"portForward": plan.ManagedJaeger.PortForward,
			},
		})
	}

	return plan, nil
}

func normalizeProtocol(protocol string) (string, error) {
	value := strings.TrimSpace(strings.ToLower(protocol))
	if value == "" {
		return DefaultProtocol, nil
	}
	switch value {
	case "grpc", "http/protobuf":
		return value, nil
	case "http", "http-protobuf":
		return "http/protobuf", nil
	default:
		return "", fmt.Errorf("unsupported protocol %q; use grpc or http/protobuf", protocol)
	}
}

func resolveTargetContainers(deployment appsv1.Deployment, container string) ([]string, error) {
	requested := strings.TrimSpace(container)
	if requested != "" {
		for _, item := range deployment.Spec.Template.Spec.Containers {
			if item.Name == requested {
				return []string{requested}, nil
			}
		}
		return nil, fmt.Errorf("container %q was not found in deployment %s", requested, deployment.Name)
	}

	names := make([]string, 0, len(deployment.Spec.Template.Spec.Containers))
	for _, item := range deployment.Spec.Template.Spec.Containers {
		names = append(names, item.Name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("deployment %s has no containers", deployment.Name)
	}
	return names, nil
}

func tracingEnv(serviceName, collectorEndpoint, protocol, samplingRatio, resourceAttrs string) map[string]string {
	return map[string]string{
		EnvTracesExporter:     "otlp",
		EnvOTLPEndpoint:       collectorEndpoint,
		EnvOTLPProtocol:       protocol,
		EnvServiceName:        serviceName,
		EnvResourceAttributes: resourceAttrs,
		EnvTracesSampler:      "parentbased_traceidratio",
		EnvTracesSamplerArg:   samplingRatio,
	}
}

func deploymentHasEnv(deployment appsv1.Deployment, containers []string, expected map[string]string) bool {
	targets := make(map[string]struct{}, len(containers))
	for _, name := range containers {
		targets[name] = struct{}{}
	}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if _, ok := targets[container.Name]; !ok {
			continue
		}
		if !containerHasEnv(container, expected) {
			return false
		}
	}
	return true
}

func containerHasEnv(container corev1.Container, expected map[string]string) bool {
	actual := make(map[string]string, len(container.Env))
	for _, env := range container.Env {
		if env.ValueFrom == nil {
			actual[env.Name] = env.Value
		}
	}
	for key, value := range expected {
		if actual[key] != value {
			return false
		}
	}
	return true
}

func managedJaegerPlan(cfg Config, protocol string) ManagedJaeger {
	return ManagedJaeger{
		Enabled:      true,
		Namespace:    cfg.JaegerNamespace,
		Name:         cfg.JaegerName,
		Image:        cfg.JaegerImage,
		ServiceName:  cfg.JaegerName,
		UIURL:        "http://localhost:16686",
		PortForward:  fmt.Sprintf("kubectl port-forward -n %s svc/%s 16686:16686", cfg.JaegerNamespace, cfg.JaegerName),
		OTLPGRPCPort: 4317,
		OTLPHTTPPort: 4318,
	}
}

func managedCollectorEndpoint(name, namespace, protocol string) string {
	port := 4317
	if protocol == "http/protobuf" {
		port = 4318
	}
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", name, namespace, port)
}

func resourceAttributes(namespace, deployment, serviceName string) string {
	attrs := map[string]string{
		"service.name":        serviceName,
		"k8s.namespace.name":  namespace,
		"k8s.deployment.name": deployment,
	}
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+attrs[key])
	}
	return strings.Join(parts, ",")
}
