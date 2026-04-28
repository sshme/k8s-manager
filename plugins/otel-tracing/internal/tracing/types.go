package tracing

const (
	PluginID      = "observability.otel-tracing"
	PluginName    = "Сбор и визуализация трасс OpenTelemetry"
	PluginVersion = "0.1.0"

	DefaultProtocol        = "grpc"
	DefaultSamplingRatio   = "1.0"
	DefaultJaegerName      = "k8s-manager-jaeger"
	DefaultJaegerNamespace = "tracing"
	DefaultJaegerImage     = "jaegertracing/all-in-one:1.62.0"

	ManagedByLabel     = "app.kubernetes.io/managed-by"
	PluginIDLabel      = "k8s-manager.io/plugin"
	TracingTargetLabel = "k8s-manager.io/tracing-target"

	EnvTracesExporter     = "OTEL_TRACES_EXPORTER"
	EnvOTLPEndpoint       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	EnvOTLPProtocol       = "OTEL_EXPORTER_OTLP_PROTOCOL"
	EnvServiceName        = "OTEL_SERVICE_NAME"
	EnvResourceAttributes = "OTEL_RESOURCE_ATTRIBUTES"
	EnvTracesSampler      = "OTEL_TRACES_SAMPLER"
	EnvTracesSamplerArg   = "OTEL_TRACES_SAMPLER_ARG"
)

type Config struct {
	Command           string
	Kubeconfig        string
	Context           string
	Namespace         string
	Deployment        string
	Container         string
	ServiceName       string
	CollectorEndpoint string
	Protocol          string
	SamplingRatio     string
	InstallJaeger     bool
	JaegerName        string
	JaegerNamespace   string
	JaegerImage       string
	Output            string
	Interactive       bool
	DryRun            bool
}

type Manifest struct {
	SchemaVersion string              `json:"schemaVersion"`
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Version       string              `json:"version"`
	Category      string              `json:"category"`
	Description   string              `json:"description"`
	Runtime       Runtime             `json:"runtime"`
	Commands      []ManifestCommand   `json:"commands"`
	Parameters    []ManifestParameter `json:"parameters"`
	Kubernetes    KubernetesAccess    `json:"kubernetes"`
}

type Runtime struct {
	Type       string `json:"type"`
	Entrypoint string `json:"entrypoint"`
}

type ManifestCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ManifestParameter struct {
	Name        string `json:"name"`
	Flag        string `json:"flag"`
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

type KubernetesAccess struct {
	Reads  []string `json:"reads"`
	Writes []string `json:"writes"`
}

type Plan struct {
	Plugin            string            `json:"plugin"`
	Namespace         string            `json:"namespace"`
	Deployment        string            `json:"deployment"`
	TargetContainers  []string          `json:"targetContainers"`
	ServiceName       string            `json:"serviceName"`
	CollectorEndpoint string            `json:"collectorEndpoint"`
	Protocol          string            `json:"protocol"`
	SamplingRatio     string            `json:"samplingRatio"`
	ResourceAttrs     string            `json:"resourceAttributes"`
	DeploymentReady   bool              `json:"deploymentReady"`
	ManagedJaeger     ManagedJaeger     `json:"managedJaeger"`
	Env               map[string]string `json:"env"`
	Operations        []Operation       `json:"operations"`
}

type ManagedJaeger struct {
	Enabled      bool   `json:"enabled"`
	Namespace    string `json:"namespace,omitempty"`
	Name         string `json:"name,omitempty"`
	Image        string `json:"image,omitempty"`
	ServiceName  string `json:"serviceName,omitempty"`
	UIURL        string `json:"uiUrl,omitempty"`
	PortForward  string `json:"portForward,omitempty"`
	OTLPGRPCPort int    `json:"otlpGrpcPort,omitempty"`
	OTLPHTTPPort int    `json:"otlpHttpPort,omitempty"`
}

type Operation struct {
	Action    string            `json:"action"`
	Kind      string            `json:"kind"`
	Namespace string            `json:"namespace,omitempty"`
	Name      string            `json:"name"`
	Reason    string            `json:"reason"`
	Details   map[string]string `json:"details,omitempty"`
}

type ApplyResult struct {
	DryRun     bool   `json:"dryRun"`
	Plan       Plan   `json:"plan"`
	Message    string `json:"message"`
	Operations int    `json:"operations"`
}
