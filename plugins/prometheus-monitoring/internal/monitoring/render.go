package monitoring

import (
	"fmt"
	"strings"
)

func renderPlan(plan Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Prometheus monitoring plan\n")
	fmt.Fprintf(&b, "Namespace: %s\n", plan.Namespace)
	fmt.Fprintf(&b, "Deployment: %s\n", plan.Deployment)
	fmt.Fprintf(&b, "Metrics: http://<pod>:%d%s (%s)\n", plan.MetricsPort, plan.MetricsPath, plan.MetricsPortSource)
	fmt.Fprintf(&b, "Scrape interval: %s\n", plan.ScrapeInterval)
	fmt.Fprintf(&b, "Service: %s", plan.ServiceName)
	if plan.ServiceExists {
		fmt.Fprintf(&b, " (existing)\n")
	} else {
		fmt.Fprintf(&b, " (will be created)\n")
	}
	if plan.PrometheusOperator {
		fmt.Fprintf(&b, "ServiceMonitor: %s\n", plan.ServiceMonitorName)
	} else {
		fmt.Fprintf(&b, "ServiceMonitor: skipped, Prometheus Operator CRD is not installed\n")
	}
	if plan.ManagedPrometheus.Enabled {
		fmt.Fprintf(&b, "Prometheus UI: %s\n", plan.ManagedPrometheus.UIURL)
		fmt.Fprintf(&b, "Port-forward: %s\n", plan.ManagedPrometheus.PortForward)
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
	if result.Plan.ManagedPrometheus.Enabled {
		fmt.Fprintf(&b, "\nOpen Prometheus UI:\n%s\n%s\n", result.Plan.ManagedPrometheus.PortForward, result.Plan.ManagedPrometheus.UIURL)
	}
	return b.String()
}
