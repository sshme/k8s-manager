package tracing

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNormalizeProtocolAcceptsHTTPAlias(t *testing.T) {
	protocol, err := normalizeProtocol("http")
	if err != nil {
		t.Fatalf("expected http alias to be accepted: %v", err)
	}
	if protocol != "http/protobuf" {
		t.Fatalf("expected http/protobuf, got %q", protocol)
	}
}

func TestManagedCollectorEndpointUsesProtocolPort(t *testing.T) {
	grpc := managedCollectorEndpoint("jaeger", "tracing", "grpc")
	if grpc != "http://jaeger.tracing.svc.cluster.local:4317" {
		t.Fatalf("unexpected grpc endpoint %q", grpc)
	}

	http := managedCollectorEndpoint("jaeger", "tracing", "http/protobuf")
	if http != "http://jaeger.tracing.svc.cluster.local:4318" {
		t.Fatalf("unexpected http endpoint %q", http)
	}
}

func TestResolveTargetContainersDefaultsToAllContainers(t *testing.T) {
	deployment := deploymentWithContainers("api", "worker")

	containers, err := resolveTargetContainers(deployment, "")
	if err != nil {
		t.Fatalf("resolve target containers: %v", err)
	}
	if len(containers) != 2 || containers[0] != "api" || containers[1] != "worker" {
		t.Fatalf("unexpected containers %#v", containers)
	}
}

func TestResolveTargetContainersValidatesRequestedContainer(t *testing.T) {
	deployment := deploymentWithContainers("api")

	_, err := resolveTargetContainers(deployment, "missing")
	if err == nil {
		t.Fatal("expected missing container error")
	}
}

func TestDeploymentHasEnvChecksAllTargetContainers(t *testing.T) {
	deployment := deploymentWithContainers("api", "worker")
	env := tracingEnv("checkout", "http://jaeger.tracing.svc.cluster.local:4317", "grpc", "1.0", "service.name=checkout")
	deployment.Spec.Template.Spec.Containers[0].Env = envVars(env)

	if deploymentHasEnv(deployment, []string{"api", "worker"}, env) {
		t.Fatal("expected deployment to be incomplete when one target container is missing env")
	}
	if !deploymentHasEnv(deployment, []string{"api"}, env) {
		t.Fatal("expected deployment to be ready for configured target container")
	}
}

func TestMergeEnvOverwritesManagedKeysAndPreservesOthers(t *testing.T) {
	current := []corev1.EnvVar{
		{Name: "APP_MODE", Value: "prod"},
		{Name: EnvOTLPEndpoint, Value: "http://old:4317"},
	}
	next := mergeEnv(current, map[string]string{EnvOTLPEndpoint: "http://new:4317", EnvOTLPProtocol: "grpc"})

	values := map[string]string{}
	for _, item := range next {
		values[item.Name] = item.Value
	}
	if values["APP_MODE"] != "prod" {
		t.Fatalf("expected custom env to be preserved, got %#v", values)
	}
	if values[EnvOTLPEndpoint] != "http://new:4317" {
		t.Fatalf("expected endpoint to be overwritten, got %#v", values)
	}
	if values[EnvOTLPProtocol] != "grpc" {
		t.Fatalf("expected protocol to be appended, got %#v", values)
	}
}

func deploymentWithContainers(names ...string) appsv1.Deployment {
	containers := make([]corev1.Container, 0, len(names))
	for _, name := range names {
		containers = append(containers, corev1.Container{Name: name})
	}
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "checkout"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: containers},
			},
		},
	}
}

func envVars(values map[string]string) []corev1.EnvVar {
	result := make([]corev1.EnvVar, 0, len(values))
	for key, value := range values {
		result = append(result, corev1.EnvVar{Name: key, Value: value})
	}
	return result
}
