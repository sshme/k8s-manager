# K8S-MANAGER plugin contract

This directory contains standalone plugins. A plugin is packaged as an executable artifact and can be invoked by the future K8S-MANAGER CLI without linking plugin code into the CLI binary.

## Standard execution model

Each plugin MUST provide a binary with the following commands:

- `manifest` - prints a JSON manifest to stdout.
- `plan` - analyzes the current Kubernetes state and prints the planned changes.
- `apply` - applies the planned changes to the Kubernetes cluster.
- `version` - prints a plugin version string.

Commands MUST write machine-readable command results to stdout and prompts, progress messages, and diagnostics to stderr. This keeps the contract usable from both an interactive terminal and a parent CLI process.

## Common flags

Plugins SHOULD support these flags for `plan` and `apply`:

- `--kubeconfig <path>` - kubeconfig path. If omitted, the standard Kubernetes loading rules are used.
- `--context <name>` - kubeconfig context override.
- `--namespace, -n <name>` - namespace override. If omitted, the namespace from the current context is used, falling back to `default`.
- `--deployment, -d <name>` - target Deployment. If omitted in interactive mode, the plugin asks the user to choose one.
- `--output text|json` - output format. `json` is intended for CLI integration.
- `--non-interactive` - disables prompts and fails if required values are missing.
- `--dry-run` - supported by `apply`; prints the plan without changing cluster resources.

## Manifest shape

The `manifest` command returns:

```json
{
  "schemaVersion": "k8s-manager.plugin/v1",
  "id": "observability.prometheus-monitoring",
  "name": "Prometheus monitoring",
  "description": "plugin description",
  "version": "0.1.0",
  "category": "observability",
  "runtime": {"type": "exec", "entrypoint": "prometheus-monitoring"},
  "commands": [
    {"name": "plan", "description": "Analyze and show Kubernetes changes"},
    {"name": "apply", "description": "Apply Kubernetes changes"}
  ]
}
```

Future CLI integration can install an artifact, run `manifest`, show available parameters in the TUI, and execute `plan` or `apply` with `--output json --non-interactive` when the CLI owns the UI.

