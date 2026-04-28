# Plugin: Сбор и визуализация трасс OpenTelemetry

Плагин настраивает выбранный Kubernetes `Deployment` на экспорт distributed traces через OTLP и, по умолчанию, поднимает managed Jaeger all-in-one для приема и визуализации трасс.

## Возможности

- Интерактивно выбирает `Deployment`, если `--deployment` не указан.
- Настраивает все containers в `Deployment` или один container через `--container`.
- Добавляет стандартные OpenTelemetry env vars: `OTEL_TRACES_EXPORTER`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_PROTOCOL`, `OTEL_SERVICE_NAME`, `OTEL_RESOURCE_ATTRIBUTES`, `OTEL_TRACES_SAMPLER`, `OTEL_TRACES_SAMPLER_ARG`.
- Поддерживает OTLP `grpc` и `http/protobuf`.
- Поднимает managed Jaeger collector/UI в отдельном namespace.
- Может работать с внешним collector через `--skip-jaeger-install --collector-endpoint`.
- Поддерживает preview через `plan` и `apply --dry-run`.

Важно: плагин настраивает экспорт трасс на уровне переменных окружения. Приложение должно быть инструментировано OpenTelemetry SDK/agent, который читает эти переменные.

## Быстрый запуск

```bash
go run ./cmd/otel-tracing manifest
go run ./cmd/otel-tracing plan --namespace default
go run ./cmd/otel-tracing apply --namespace default --deployment my-app
go run ./cmd/otel-tracing apply --namespace default --deployment my-app --dry-run
```

После успешного `apply` открыть Jaeger UI можно так:

```bash
kubectl port-forward -n tracing svc/k8s-manager-jaeger 16686:16686
```

Затем открыть `http://localhost:16686`.

## Внешний collector

Если в кластере уже есть OpenTelemetry Collector, Jaeger, Tempo или другой OTLP-compatible backend, managed Jaeger можно отключить:

```bash
otel-tracing apply \
  --namespace default \
  --deployment my-app \
  --skip-jaeger-install \
  --collector-endpoint http://otel-collector.observability.svc.cluster.local:4317 \
  --protocol grpc
```

Для HTTP/protobuf обычно используется порт `4318`:

```bash
otel-tracing apply \
  --namespace default \
  --deployment my-app \
  --skip-jaeger-install \
  --collector-endpoint http://otel-collector.observability.svc.cluster.local:4318 \
  --protocol http/protobuf
```

## Managed resources

Плагин создает следующие managed resources, если не указан `--skip-jaeger-install`:

- `Namespace/tracing`, если namespace отсутствует.
- `Deployment/k8s-manager-jaeger`.
- `Service/k8s-manager-jaeger` с портами `ui`, `otlp-grpc`, `otlp-http`.

Повторный запуск `apply` идемпотентно обновляет Deployment приложения и managed Jaeger ресурсы.

Для будущей интеграции с CLI используйте:

```bash
otel-tracing plan --output json --non-interactive --namespace default --deployment my-app
otel-tracing apply --output json --non-interactive --namespace default --deployment my-app
```

CLI должен передавать kubeconfig обычным способом Kubernetes или через `--kubeconfig`.
