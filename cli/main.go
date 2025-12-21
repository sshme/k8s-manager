package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"k8s-manager/cli/internal/k8s"
	"k8s-manager/cli/internal/tui"
	pb "k8s-manager/proto/market"
)

var (
	marketAddr = flag.String("market-addr", "localhost:50051", "Market service address")
)

func main() {
	flag.Parse()

	// Connect to market service
	conn, err := grpc.NewClient(*marketAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to market service: %v", err)
	}
	defer conn.Close()

	marketClient := pb.NewMarketServiceClient(conn)

	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	// Test market connection
	ctx := context.Background()
	_, err = marketClient.ListPlugins(ctx, &pb.ListPluginsRequest{
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		log.Fatalf("Failed to connect to market service: %v", err)
	}

	// Initialize TUI
	model := tui.NewModel(marketClient, k8sClient)
	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
