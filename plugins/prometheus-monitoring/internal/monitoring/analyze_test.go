package monitoring

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestResolveMetricsPortPrefersNamedMetricsPort(t *testing.T) {
	deployment := deploymentWithPorts([]corev1.ContainerPort{
		{Name: "http", ContainerPort: 8080},
		{Name: "metrics", ContainerPort: 9090},
	})

	port, source := resolveMetricsPort(deployment, 0)
	if port != 9090 {
		t.Fatalf("expected metrics port 9090, got %d", port)
	}
	if source != "containerPort named metrics" {
		t.Fatalf("unexpected source %q", source)
	}
}

func TestResolveMetricsPortFallsBackTo8080(t *testing.T) {
	deployment := deploymentWithPorts([]corev1.ContainerPort{
		{Name: "http", ContainerPort: 8080},
	})

	port, _ := resolveMetricsPort(deployment, 0)
	if port != 8080 {
		t.Fatalf("expected fallback port 8080, got %d", port)
	}
}

func TestResolveMetricsPortUsesUserOverride(t *testing.T) {
	deployment := deploymentWithPorts([]corev1.ContainerPort{
		{Name: "metrics", ContainerPort: 9090},
	})

	port, source := resolveMetricsPort(deployment, 9443)
	if port != 9443 {
		t.Fatalf("expected override port 9443, got %d", port)
	}
	if source != "user" {
		t.Fatalf("unexpected source %q", source)
	}
}

func TestServiceSelectsDeployment(t *testing.T) {
	deployment := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "api", "tier": "backend"},
				},
			},
		},
	}
	service := corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "api"},
		},
	}

	if !serviceSelectsDeployment(service, deployment) {
		t.Fatal("expected service selector to match deployment pod labels")
	}
}

func TestServiceHasMetricsPort(t *testing.T) {
	service := corev1.Service{
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 80, TargetPort: intstr.FromInt(8080)},
			},
		},
	}

	if !serviceHasMetricsPort(service, 8080) {
		t.Fatal("expected targetPort 8080 to count as metrics endpoint")
	}
}

func TestServiceHasMetricsPortDoesNotTrustServicePortWithDifferentTarget(t *testing.T) {
	service := corev1.Service{
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 9090, TargetPort: intstr.FromInt(8080)},
			},
		},
	}

	if serviceHasMetricsPort(service, 9090) {
		t.Fatal("expected service port 9090 targeting 8080 not to count as container metrics port 9090")
	}
}

func TestEnsureServicePortPreservesExistingServicePort(t *testing.T) {
	service := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 80, TargetPort: intstr.FromInt(8080)},
			},
		},
	}

	ensureServicePort(service, "http", 8080)

	if len(service.Spec.Ports) != 1 {
		t.Fatalf("expected one existing service port, got %d", len(service.Spec.Ports))
	}
	if service.Spec.Ports[0].Port != 80 {
		t.Fatalf("expected service port 80 to be preserved, got %d", service.Spec.Ports[0].Port)
	}
	if service.Spec.Ports[0].TargetPort.IntVal != 8080 {
		t.Fatalf("expected targetPort 8080 to be preserved, got %d", service.Spec.Ports[0].TargetPort.IntVal)
	}
}

func deploymentWithPorts(ports []corev1.ContainerPort) appsv1.Deployment {
	return appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Ports: ports},
					},
				},
			},
		},
	}
}
