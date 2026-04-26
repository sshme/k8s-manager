package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const PORT = 8080

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
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/api/hello", metricsMiddleware(helloHandler))

	go selfRequester()

	log.Printf("Server starting on :%d\n", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(10 * time.Millisecond)
	w.Write([]byte("Hello, Prometheus!"))
}

func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		duration := time.Since(start).Seconds()
		helloRequests.WithLabelValues(r.Method).Inc()
		helloDuration.WithLabelValues(r.Method).Observe(duration)
	}
}

func selfRequester() {
	time.Sleep(2 * time.Second)

	client := http.Client{Timeout: 3 * time.Second}
	for {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/api/hello", PORT))
		if err != nil {
			log.Printf("Self-request failed: %v", err)
		} else {
			resp.Body.Close()
			log.Println("Self-request succeeded")
		}
		time.Sleep(5 * time.Second)
	}
}
