package tracing

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
		Command:         args[0],
		Protocol:        DefaultProtocol,
		SamplingRatio:   DefaultSamplingRatio,
		InstallJaeger:   true,
		JaegerName:      DefaultJaegerName,
		JaegerNamespace: DefaultJaegerNamespace,
		JaegerImage:     DefaultJaegerImage,
		Output:          "text",
		Interactive:     true,
	}

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.StringVar(&cfg.Kubeconfig, "kubeconfig", "", "kubeconfig path")
	fs.StringVar(&cfg.Context, "context", "", "kubeconfig context")
	fs.StringVar(&cfg.Namespace, "namespace", "", "Kubernetes namespace")
	fs.StringVar(&cfg.Namespace, "n", "", "Kubernetes namespace")
	fs.StringVar(&cfg.Deployment, "deployment", "", "target Deployment name")
	fs.StringVar(&cfg.Deployment, "d", "", "target Deployment name")
	fs.StringVar(&cfg.Container, "container", "", "target container; defaults to all containers")
	fs.StringVar(&cfg.ServiceName, "service-name", "", "OpenTelemetry service.name")
	fs.StringVar(&cfg.CollectorEndpoint, "collector-endpoint", "", "OTLP collector endpoint")
	fs.StringVar(&cfg.Protocol, "protocol", DefaultProtocol, "OTLP protocol: grpc or http/protobuf")
	fs.StringVar(&cfg.SamplingRatio, "sampling-ratio", DefaultSamplingRatio, "trace sampling ratio from 0.0 to 1.0")
	fs.StringVar(&cfg.JaegerNamespace, "jaeger-namespace", DefaultJaegerNamespace, "namespace for managed Jaeger")
	fs.StringVar(&cfg.JaegerName, "jaeger-name", DefaultJaegerName, "name for managed Jaeger resources")
	fs.StringVar(&cfg.JaegerImage, "jaeger-image", DefaultJaegerImage, "container image for managed Jaeger")
	fs.BoolFunc("skip-jaeger-install", "do not create managed Jaeger collector and UI", func(string) error {
		cfg.InstallJaeger = false
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
	if _, err := normalizeProtocol(cfg.Protocol); err != nil {
		return Config{}, err
	}
	if err := validateSamplingRatio(cfg.SamplingRatio); err != nil {
		return Config{}, err
	}

	cfg.JaegerName = strings.TrimSpace(cfg.JaegerName)
	cfg.JaegerNamespace = strings.TrimSpace(cfg.JaegerNamespace)
	cfg.JaegerImage = strings.TrimSpace(cfg.JaegerImage)
	if cfg.InstallJaeger {
		if cfg.JaegerName == "" {
			return Config{}, errors.New("--jaeger-name cannot be empty")
		}
		if cfg.JaegerNamespace == "" {
			return Config{}, errors.New("--jaeger-namespace cannot be empty")
		}
		if cfg.JaegerImage == "" {
			return Config{}, errors.New("--jaeger-image cannot be empty")
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
	if !cfg.InstallJaeger && cfg.CollectorEndpoint == "" && cfg.Interactive {
		protocol, err := normalizeProtocol(cfg.Protocol)
		if err != nil {
			return err
		}
		cfg.CollectorEndpoint = promptString(reader, stderr, "OTLP collector endpoint", exampleCollectorEndpoint(protocol))
	}

	initialPlan, err := BuildPlan(ctx, client, cfg)
	if err != nil {
		return err
	}

	if cfg.Interactive {
		if cfg.ServiceName == "" {
			cfg.ServiceName = promptString(reader, stderr, "OpenTelemetry service name", initialPlan.ServiceName)
		}
		if cfg.SamplingRatio == "" || cfg.SamplingRatio == DefaultSamplingRatio {
			cfg.SamplingRatio = promptString(reader, stderr, "Trace sampling ratio", initialPlan.SamplingRatio)
			if err := validateSamplingRatio(cfg.SamplingRatio); err != nil {
				return err
			}
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
		Message:    "OpenTelemetry tracing has been configured",
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

func validateSamplingRatio(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || parsed < 0 || parsed > 1 {
		return fmt.Errorf("sampling ratio must be a number from 0.0 to 1.0")
	}
	return nil
}

func exampleCollectorEndpoint(protocol string) string {
	port := 4317
	if protocol == "http/protobuf" {
		port = 4318
	}
	return fmt.Sprintf("http://otel-collector.observability.svc.cluster.local:%d", port)
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
  otel-tracing manifest
  otel-tracing version
  otel-tracing plan [flags]
  otel-tracing apply [flags]

Flags:
  --kubeconfig <path>        kubeconfig path
  --context <name>           kubeconfig context
  --namespace, -n <name>     Kubernetes namespace
  --deployment, -d <name>    target Deployment
  --container <name>         target container; defaults to all containers
  --service-name <name>      OpenTelemetry service.name
  --collector-endpoint <url> OTLP collector endpoint
  --protocol <value>         OTLP protocol: grpc or http/protobuf
  --sampling-ratio <value>   trace sampling ratio from 0.0 to 1.0
  --jaeger-namespace <name>  namespace for managed Jaeger (default tracing)
  --jaeger-name <name>       managed Jaeger resource name (default k8s-manager-jaeger)
  --jaeger-image <image>     managed Jaeger image
  --skip-jaeger-install      use an external collector endpoint
  --output text|json         output format
  --non-interactive          disable prompts
  --dry-run                  preview apply without changes`)
}

func defaultManifest() Manifest {
	return Manifest{
		SchemaVersion: "k8s-manager.plugin/v1",
		ID:            PluginID,
		Name:          PluginName,
		Version:       PluginVersion,
		Category:      "observability",
		Description:   "Настраивает выбранный Deployment на экспорт трасс OpenTelemetry по OTLP и поднимает managed Jaeger collector/UI либо подключает внешний collector endpoint.",
		Runtime: Runtime{
			Type:       "exec",
			Entrypoint: "otel-tracing",
		},
		Commands: []ManifestCommand{
			{Name: "plan", Description: "Анализирует Deployment и показывает план трассировки"},
			{Name: "apply", Description: "Применяет OTEL env vars и managed Jaeger"},
		},
		Parameters: []ManifestParameter{
			{Name: "deployment", Flag: "--deployment", Description: "Целевой Deployment; если не задан, выбирается интерактивно"},
			{Name: "namespace", Flag: "--namespace", Description: "Namespace Kubernetes"},
			{Name: "container", Flag: "--container", Description: "Целевой container; если не задан, настраиваются все containers Deployment"},
			{Name: "serviceName", Flag: "--service-name", Description: "Значение OTEL_SERVICE_NAME; по умолчанию имя Deployment"},
			{Name: "collectorEndpoint", Flag: "--collector-endpoint", Description: "OTLP endpoint; если managed Jaeger включен, вычисляется автоматически"},
			{Name: "protocol", Flag: "--protocol", Default: DefaultProtocol, Description: "OTLP protocol: grpc или http/protobuf"},
			{Name: "samplingRatio", Flag: "--sampling-ratio", Default: DefaultSamplingRatio, Description: "Доля трасс для parentbased_traceidratio sampler"},
			{Name: "jaegerNamespace", Flag: "--jaeger-namespace", Default: DefaultJaegerNamespace, Description: "Namespace для managed Jaeger"},
			{Name: "jaegerName", Flag: "--jaeger-name", Default: DefaultJaegerName, Description: "Имя managed Jaeger ресурсов"},
			{Name: "jaegerImage", Flag: "--jaeger-image", Default: DefaultJaegerImage, Description: "Container image для managed Jaeger"},
			{Name: "skipJaegerInstall", Flag: "--skip-jaeger-install", Default: "false", Description: "Не создавать managed Jaeger, использовать внешний OTLP collector"},
		},
		Kubernetes: KubernetesAccess{
			Reads:  []string{"deployments", "namespaces"},
			Writes: []string{"deployments", "namespaces", "services"},
		},
	}
}
