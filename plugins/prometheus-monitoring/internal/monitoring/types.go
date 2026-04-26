package monitoring

const (
	PluginID      = "observability.prometheus-monitoring"
	PluginName    = "Добавление Prometheus-мониторинга"
	PluginVersion = "0.1.0"

	DefaultMetricsPath    = "/metrics"
	DefaultMetricsPort    = 9090
	DefaultScrapeInterval = "30s"

	PrometheusScrapeAnnotation = "prometheus.io/scrape"
	PrometheusPortAnnotation   = "prometheus.io/port"
	PrometheusPathAnnotation   = "prometheus.io/path"

	ManagedByLabel           = "app.kubernetes.io/managed-by"
	PluginIDLabel            = "k8s-manager.io/plugin"
	PrometheusTargetLabel    = "k8s-manager.io/prometheus-target"
	ServiceMonitorCRDName    = "servicemonitors.monitoring.coreos.com"
	ServiceMonitorAPIGroup   = "monitoring.coreos.com"
	ServiceMonitorAPIVersion = "v1"
	ServiceMonitorKind       = "ServiceMonitor"
)

type Config struct {
	Command        string
	Kubeconfig     string
	Context        string
	Namespace      string
	Deployment     string
	MetricsPath    string
	MetricsPort    int
	ScrapeInterval string
	Output         string
	Interactive    bool
	DryRun         bool
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
	Plugin              string      `json:"plugin"`
	Namespace           string      `json:"namespace"`
	Deployment          string      `json:"deployment"`
	MetricsPath         string      `json:"metricsPath"`
	MetricsPort         int         `json:"metricsPort"`
	MetricsPortSource   string      `json:"metricsPortSource"`
	ScrapeInterval      string      `json:"scrapeInterval"`
	HasScrapeAnnotation bool        `json:"hasScrapeAnnotation"`
	ServiceExists       bool        `json:"serviceExists"`
	ServiceName         string      `json:"serviceName"`
	ServicePortName     string      `json:"servicePortName"`
	PrometheusOperator  bool        `json:"prometheusOperator"`
	ServiceMonitorName  string      `json:"serviceMonitorName,omitempty"`
	Operations          []Operation `json:"operations"`
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
