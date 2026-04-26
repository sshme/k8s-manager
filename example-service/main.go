package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

const (
	PORT = 8080

	// Установите true для отправки трейсов в Jaeger (OTLP gRPC на localhost:4317)
	// Установите false для вывода трейсов в консоль (stdout)
	useOTLP = true
)

var (
	helloRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hello_requests_total",
		Help: "Total number of requests to /api/hello",
	}, []string{"method"})

	helloDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "hello_duration_seconds",
		Help:    "Duration of requests to /api/hello",
		Buckets: prometheus.DefBuckets,
	}, []string{"method"})
)

func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("Failed to init tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer: %v", err)
		}
	}()
	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	helloHandlerWithTrace := otelhttp.NewHandler(http.HandlerFunc(helloHandler), "hello-handler")

	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/api/hello", metricsMiddleware(helloHandlerWithTrace))

	go selfRequester()

	log.Printf("Server starting on :%d\n", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil))
}

func initTracer() (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	var err error

	if useOTLP {
		exporter, err = otlptracegrpc.New(context.Background(),
			otlptracegrpc.WithEndpoint("localhost:4317"),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
		log.Println("Tracing: using OTLP gRPC exporter (Jaeger on localhost:4317)")
	} else {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
		}
		log.Println("Tracing: using stdout exporter (console pretty JSON)")
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("hello-service"),
			semconv.ServiceVersionKey.String("1.0.0"),
		)),
	)
	return tp, nil
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	depthStr := r.Header.Get("X-Depth")
	depth := 0
	if depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil {
			depth = d
		}
	}

	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.Int("depth", depth),
		attribute.Int("max_depth", 5),
	)

	log.Printf("[depth=%d trace_id=%s] Handling request", depth, span.SpanContext().TraceID())

	time.Sleep(10 * time.Millisecond)

	if depth < 5 && rand.Intn(2) == 0 {
		nextDepth := depth + 1
		log.Printf("[depth=%d] Calling next level (depth=%d)", depth, nextDepth)

		reqURL := fmt.Sprintf("http://localhost:%d/api/hello", PORT)
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err == nil {
			req.Header.Set("X-Depth", strconv.Itoa(nextDepth))
			otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

			client := &http.Client{
				Transport: otelhttp.NewTransport(http.DefaultTransport),
				Timeout:   3 * time.Second,
			}
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("[depth=%d] Recursive call failed: %v", depth, err)
				span.RecordError(err)
			} else {
				resp.Body.Close()
				log.Printf("[depth=%d] Recursive call succeeded", depth)
			}
		} else {
			log.Printf("[depth=%d] Failed to create request: %v", depth, err)
			span.RecordError(err)
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Hello, Prometheus! (depth=%d)", depth)
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start).Seconds()
		helloRequests.WithLabelValues(r.Method).Inc()
		helloDuration.WithLabelValues(r.Method).Observe(duration)
	})
}

func selfRequester() {
	time.Sleep(2 * time.Second)

	ctx := context.Background()
	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   3 * time.Second,
	}

	for {
		req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%d/api/hello", PORT), nil)
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Self-request failed: %v", err)
		} else {
			resp.Body.Close()
			log.Println("Self-request succeeded")
		}
		time.Sleep(5 * time.Second)
	}
}
