package main

import (
	"context"
	"fmt"
	"os"

	"k8s-manager/plugins/otel-tracing/internal/tracing"
)

func main() {
	if err := tracing.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
