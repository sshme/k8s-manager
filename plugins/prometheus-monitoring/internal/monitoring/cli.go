package monitoring

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("command is required")
	}

	switch args[0] {
	case "manifest":
		return writeJSON(stdout, defaultManifest())
	case "version":
		_, err := fmt.Fprintln(stdout, PluginVersion)
		return err
	case "plan", "apply":
		cfg, err := parseConfig(args)
		if err != nil {
			return err
		}
		return runKubernetesCommand(ctx, cfg, stdin, stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func parseConfig(args []string) (Config, error) {
	cfg := Config{
		Command:             args[0],
		MetricsPath:         "",
		ScrapeInterval:      DefaultScrapeInterval,
		InstallPrometheus:   true,
		PrometheusName:      DefaultPrometheusName,
		PrometheusNamespace: DefaultPrometheusNamespace,
		PrometheusImage:     DefaultPrometheusImage,
		Output:              "text",
		Interactive:         true,
	}

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.StringVar(&cfg.Kubeconfig, "kubeconfig", "", "kubeconfig path")
	fs.StringVar(&cfg.Context, "context", "", "kubeconfig context")
	fs.StringVar(&cfg.Namespace, "namespace", "", "Kubernetes namespace")
	fs.StringVar(&cfg.Namespace, "n", "", "Kubernetes namespace")
	fs.StringVar(&cfg.Deployment, "deployment", "", "target Deployment name")
	fs.StringVar(&cfg.Deployment, "d", "", "target Deployment name")
	fs.StringVar(&cfg.MetricsPath, "metrics-path", "", "metrics endpoint path")
	fs.IntVar(&cfg.MetricsPort, "port", 0, "metrics endpoint port")
	fs.StringVar(&cfg.ScrapeInterval, "scrape-interval", DefaultScrapeInterval, "ServiceMonitor scrape interval")
	fs.StringVar(&cfg.PrometheusNamespace, "prometheus-namespace", DefaultPrometheusNamespace, "namespace for managed Prometheus")
	fs.StringVar(&cfg.PrometheusName, "prometheus-name", DefaultPrometheusName, "name for managed Prometheus resources")
	fs.StringVar(&cfg.PrometheusImage, "prometheus-image", DefaultPrometheusImage, "container image for managed Prometheus")
	fs.BoolFunc("skip-prometheus-install", "do not create managed Prometheus UI", func(string) error {
		cfg.InstallPrometheus = false
		return nil
	})
	fs.StringVar(&cfg.Output, "output", "text", "output format: text or json")
	fs.BoolFunc("non-interactive", "disable interactive prompts", func(string) error {
		cfg.Interactive = false
		return nil
	})
	if args[0] == "apply" {
		fs.BoolVar(&cfg.DryRun, "dry-run", false, "show the plan without applying changes")
	}

	if err := fs.Parse(args[1:]); err != nil {
		return Config{}, err
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	if cfg.Output != "text" && cfg.Output != "json" {
		return Config{}, fmt.Errorf("unsupported output %q; use text or json", cfg.Output)
	}
	cfg.PrometheusName = strings.TrimSpace(cfg.PrometheusName)
	cfg.PrometheusNamespace = strings.TrimSpace(cfg.PrometheusNamespace)
	cfg.PrometheusImage = strings.TrimSpace(cfg.PrometheusImage)
	if cfg.InstallPrometheus {
		if cfg.PrometheusName == "" {
			return Config{}, errors.New("--prometheus-name cannot be empty")
		}
		if cfg.PrometheusNamespace == "" {
			return Config{}, errors.New("--prometheus-namespace cannot be empty")
		}
		if cfg.PrometheusImage == "" {
			return Config{}, errors.New("--prometheus-image cannot be empty")
		}
	}
	return cfg, nil
}

func runKubernetesCommand(ctx context.Context, cfg Config, stdin io.Reader, stdout, stderr io.Writer) error {
	client, namespace, err := NewClusterClient(cfg)
	if err != nil {
		return err
	}
	cfg.Namespace = namespace

	reader := bufio.NewReader(stdin)
	if cfg.Deployment == "" {
		if !cfg.Interactive {
			return errors.New("--deployment is required in non-interactive mode")
		}
		deployment, err := promptDeployment(ctx, client, cfg.Namespace, reader, stderr)
		if err != nil {
			return err
		}
		cfg.Deployment = deployment
	}

	initialPlan, err := BuildPlan(ctx, client, cfg)
	if err != nil {
		return err
	}

	if cfg.Interactive {
		if cfg.MetricsPath == "" {
			cfg.MetricsPath = promptString(reader, stderr, "Metrics path", initialPlan.MetricsPath)
		}
		if cfg.MetricsPort == 0 {
			cfg.MetricsPort = promptInt(reader, stderr, "Metrics port", initialPlan.MetricsPort)
		}
		if strings.TrimSpace(cfg.ScrapeInterval) == "" || cfg.ScrapeInterval == DefaultScrapeInterval {
			cfg.ScrapeInterval = promptString(reader, stderr, "Scrape interval", initialPlan.ScrapeInterval)
		}
	}

	plan, err := BuildPlan(ctx, client, cfg)
	if err != nil {
		return err
	}

	if cfg.Command == "plan" || cfg.DryRun {
		if cfg.Command == "apply" {
			result := ApplyResult{
				DryRun:     true,
				Plan:       plan,
				Message:    "dry-run: no changes were applied",
				Operations: len(plan.Operations),
			}
			return writeResult(stdout, cfg.Output, result, renderApplyResult(result))
		}
		return writeResult(stdout, cfg.Output, plan, renderPlan(plan))
	}

	if err := Apply(ctx, client, plan); err != nil {
		return err
	}

	result := ApplyResult{
		Plan:       plan,
		Message:    "Prometheus monitoring has been configured",
		Operations: len(plan.Operations),
	}
	return writeResult(stdout, cfg.Output, result, renderApplyResult(result))
}

func promptDeployment(ctx context.Context, client *ClusterClient, namespace string, reader *bufio.Reader, out io.Writer) (string, error) {
	deployments, err := ListDeploymentNames(ctx, client, namespace)
	if err != nil {
		return "", err
	}
	if len(deployments) == 0 {
		return "", fmt.Errorf("no deployments found in namespace %q", namespace)
	}

	fmt.Fprintf(out, "Select Deployment in namespace %s:\n", namespace)
	for i, name := range deployments {
		fmt.Fprintf(out, "  %d) %s\n", i+1, name)
	}

	for {
		fmt.Fprint(out, "Deployment number: ")
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		index, convErr := strconv.Atoi(strings.TrimSpace(line))
		if convErr == nil && index >= 1 && index <= len(deployments) {
			return deployments[index-1], nil
		}
		fmt.Fprintln(out, "Please enter a valid number.")
		if errors.Is(err, io.EOF) {
			return "", errors.New("deployment selection was not provided")
		}
	}
}

func promptString(reader *bufio.Reader, out io.Writer, label, defaultValue string) string {
	fmt.Fprintf(out, "%s [%s]: ", label, defaultValue)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return defaultValue
	}
	value := strings.TrimSpace(line)
	if value == "" {
		return defaultValue
	}
	return value
}

func promptInt(reader *bufio.Reader, out io.Writer, label string, defaultValue int) int {
	for {
		value := promptString(reader, out, label, strconv.Itoa(defaultValue))
		parsed, err := strconv.Atoi(value)
		if err == nil && parsed > 0 && parsed <= 65535 {
			return parsed
		}
		fmt.Fprintln(out, "Please enter a valid TCP port.")
	}
}

func writeResult(stdout io.Writer, output string, value any, text string) error {
	if output == "json" {
		return writeJSON(stdout, value)
	}
	_, err := fmt.Fprint(stdout, text)
	return err
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, `Usage:
  prometheus-monitoring manifest
  prometheus-monitoring version
  prometheus-monitoring plan [flags]
  prometheus-monitoring apply [flags]

Flags:
  --kubeconfig <path>       kubeconfig path
  --context <name>          kubeconfig context
  --namespace, -n <name>    Kubernetes namespace
  --deployment, -d <name>   target Deployment
  --metrics-path <path>     metrics endpoint path (default /metrics)
  --port <number>           metrics endpoint port
  --scrape-interval <dur>   ServiceMonitor scrape interval (default 30s)
  --prometheus-namespace    namespace for managed Prometheus (default monitoring)
  --prometheus-name         managed Prometheus resource name (default k8s-manager-prometheus)
  --prometheus-image        managed Prometheus image
  --skip-prometheus-install skip managed Prometheus UI setup
  --output text|json        output format
  --non-interactive         disable prompts
  --dry-run                 preview apply without changes`)
}

func defaultManifest() Manifest {
	return Manifest{
		SchemaVersion: "k8s-manager.plugin/v1",
		ID:            PluginID,
		Name:          PluginName,
		Version:       PluginVersion,
		Category:      "observability",
		Description:   "Автоматически подключает сбор метрик Prometheus к выбранному Deployment, настраивает Service/ServiceMonitor и поднимает managed Prometheus UI, доступный через port-forward.",
		Runtime: Runtime{
			Type:       "exec",
			Entrypoint: "prometheus-monitoring",
		},
		Commands: []ManifestCommand{
			{Name: "plan", Description: "Анализирует Deployment и показывает план изменений"},
			{Name: "apply", Description: "Применяет аннотации, Service и ServiceMonitor"},
		},
		Parameters: []ManifestParameter{
			{Name: "deployment", Flag: "--deployment", Description: "Целевой Deployment; если не задан, выбирается интерактивно"},
			{Name: "namespace", Flag: "--namespace", Description: "Namespace Kubernetes"},
			{Name: "metricsPath", Flag: "--metrics-path", Default: DefaultMetricsPath, Description: "HTTP path endpoint метрик"},
			{Name: "port", Flag: "--port", Description: "Порт метрик; если не задан, определяется автоматически"},
			{Name: "scrapeInterval", Flag: "--scrape-interval", Default: DefaultScrapeInterval, Description: "Интервал scrape для ServiceMonitor и managed Prometheus"},
			{Name: "prometheusNamespace", Flag: "--prometheus-namespace", Default: DefaultPrometheusNamespace, Description: "Namespace для managed Prometheus с UI"},
			{Name: "prometheusName", Flag: "--prometheus-name", Default: DefaultPrometheusName, Description: "Имя managed Prometheus ресурсов"},
			{Name: "prometheusImage", Flag: "--prometheus-image", Default: DefaultPrometheusImage, Description: "Container image для managed Prometheus"},
			{Name: "skipPrometheusInstall", Flag: "--skip-prometheus-install", Default: "false", Description: "Не создавать managed Prometheus UI"},
		},
		Kubernetes: KubernetesAccess{
			Reads:  []string{"deployments", "services", "customresourcedefinitions", "namespaces"},
			Writes: []string{"deployments", "services", "servicemonitors.monitoring.coreos.com", "namespaces", "serviceaccounts", "configmaps", "clusterroles", "clusterrolebindings"},
		},
	}
}
