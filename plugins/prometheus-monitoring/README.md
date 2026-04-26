# Plugin: Добавление Prometheus-мониторинга

Плагин подключает Prometheus scraping к выбранному Kubernetes `Deployment`.

## Возможности

- Интерактивно выбирает `Deployment`, если `--deployment` не указан.
- Проверяет аннотации `prometheus.io/scrape`, `prometheus.io/port`, `prometheus.io/path`.
- Автоматически определяет порт метрик: сначала `containerPort` с именем `metrics`, затем порт `8080`, затем fallback `9090`.
- Находит существующий `Service`, selector которого указывает на pod labels выбранного `Deployment`.
- Создает минимальный `Service`, если подходящего сервиса нет.
- Добавляет metrics port в существующий `Service`, если он отсутствует.
- Проверяет наличие CRD `servicemonitors.monitoring.coreos.com`.
- Создает или обновляет `ServiceMonitor`, если в кластере установлен Prometheus Operator.
- Создает managed Prometheus в namespace `monitoring`, если не отключить это через `--skip-prometheus-install`.
- Настраивает Prometheus UI, доступный через `kubectl port-forward -n monitoring svc/k8s-manager-prometheus 9090:9090`.
- Поддерживает preview через `plan` и `apply --dry-run`.

## Быстрый запуск

```bash
go run ./cmd/prometheus-monitoring manifest
go run ./cmd/prometheus-monitoring plan --namespace default
go run ./cmd/prometheus-monitoring apply --namespace default --deployment my-app
go run ./cmd/prometheus-monitoring apply --namespace default --deployment my-app --dry-run
```

После успешного `apply` открыть Prometheus UI можно так:

```bash
kubectl port-forward -n monitoring svc/k8s-manager-prometheus 9090:9090
```

Затем открыть `http://localhost:9090/targets`.

Плагин создает следующие managed resources:

- `Namespace/monitoring`, если namespace отсутствует.
- `ServiceAccount/k8s-manager-prometheus`.
- `ClusterRole/k8s-manager-prometheus`.
- `ClusterRoleBinding/k8s-manager-prometheus`.
- `ConfigMap/k8s-manager-prometheus` с `prometheus.yml`.
- `Deployment/k8s-manager-prometheus`.
- `Service/k8s-manager-prometheus`.

Managed Prometheus автоматически ищет `Service` с labels и annotations, которые добавляет этот плагин, поэтому повторный запуск `apply` для другого Deployment будет добавлять новый target без ручной правки конфигурации Prometheus.

Для будущей интеграции с CLI используйте:

```bash
prometheus-monitoring plan --output json --non-interactive --namespace default --deployment my-app
prometheus-monitoring apply --output json --non-interactive --namespace default --deployment my-app
```

CLI должен передавать kubeconfig обычным способом Kubernetes или через `--kubeconfig`.
