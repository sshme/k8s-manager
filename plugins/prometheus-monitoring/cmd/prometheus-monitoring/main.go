package main

import (
	"context"
	"fmt"
	"os"

	"k8s-manager/plugins/prometheus-monitoring/internal/monitoring"
)

func main() {
	if err := monitoring.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
