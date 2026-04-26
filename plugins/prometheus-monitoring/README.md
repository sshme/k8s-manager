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
- Поддерживает preview через `plan` и `apply --dry-run`.

## Быстрый запуск

```bash
go run ./cmd/prometheus-monitoring manifest
go run ./cmd/prometheus-monitoring plan --namespace default
go run ./cmd/prometheus-monitoring apply --namespace default --deployment my-app
go run ./cmd/prometheus-monitoring apply --namespace default --deployment my-app --dry-run
```

Для будущей интеграции с CLI используйте:

```bash
prometheus-monitoring plan --output json --non-interactive --namespace default --deployment my-app
prometheus-monitoring apply --output json --non-interactive --namespace default --deployment my-app
```

CLI должен передавать kubeconfig обычным способом Kubernetes или через `--kubeconfig`.

