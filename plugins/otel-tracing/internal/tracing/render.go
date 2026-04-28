package tracing

import (
	"fmt"
	"strings"
)

func renderPlan(plan Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "OpenTelemetry tracing plan\n")
	fmt.Fprintf(&b, "Namespace: %s\n", plan.Namespace)
	fmt.Fprintf(&b, "Deployment: %s\n", plan.Deployment)
	fmt.Fprintf(&b, "Containers: %s\n", strings.Join(plan.TargetContainers, ", "))
	fmt.Fprintf(&b, "Service name: %s\n", plan.ServiceName)
	fmt.Fprintf(&b, "Collector endpoint: %s\n", plan.CollectorEndpoint)
	fmt.Fprintf(&b, "Protocol: %s\n", plan.Protocol)
	fmt.Fprintf(&b, "Sampling ratio: %s\n", plan.SamplingRatio)
	if plan.ManagedJaeger.Enabled {
		fmt.Fprintf(&b, "Jaeger UI: %s\n", plan.ManagedJaeger.UIURL)
		fmt.Fprintf(&b, "Port-forward: %s\n", plan.ManagedJaeger.PortForward)
	} else {
		fmt.Fprintf(&b, "Jaeger: skipped, external collector endpoint is used\n")
	}
	fmt.Fprintf(&b, "\nOperations:\n")
	for _, operation := range plan.Operations {
		namespace := operation.Namespace
		if namespace != "" {
			namespace += "/"
		}
		fmt.Fprintf(&b, "- %s %s %s%s: %s\n", operation.Action, operation.Kind, namespace, operation.Name, operation.Reason)
	}
	return b.String()
}

func renderApplyResult(result ApplyResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", result.Message)
	fmt.Fprint(&b, renderPlan(result.Plan))
	if result.Plan.ManagedJaeger.Enabled {
		fmt.Fprintf(&b, "\nOpen Jaeger UI:\n%s\n%s\n", result.Plan.ManagedJaeger.PortForward, result.Plan.ManagedJaeger.UIURL)
	}
	return b.String()
}
