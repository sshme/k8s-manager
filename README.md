# K8s Manager

A Kubernetes management CLI tool with a TUI (Terminal User Interface) similar to k9s, featuring a plugin marketplace for managing Kubernetes features.

## Features

- **TUI Interface**: Beautiful terminal UI for browsing and managing plugins
- **Plugin Marketplace**: Browse, search, and install Kubernetes plugins
- **Kubernetes Integration**: Uses existing kubeconfig for cluster management
- **Plugin Categories**:
  - Load Balancer
  - Ingress Controller
  - Dashboard
  - Service Mesh
  - Observability & Monitoring
  - Tracing

## Architecture

The project consists of two main components:

1. **CLI** (`cli/`): Terminal UI application for users
2. **Market Service** (`market/`): gRPC service for plugin management with PostgreSQL backend

## Tech Stack

- **Go 1.25+**
- **PostgreSQL**: Database for plugin storage
- **gRPC**: Communication between CLI and market service
- **Docker**: Containerization
- **Bubbletea**: TUI framework
- **Kubernetes Client-Go**: Kubernetes API integration

## Getting Started

### Prerequisites

- Go 1.25 or higher
- Docker and Docker Compose
- Kubernetes cluster (or kubeconfig configured)
- PostgreSQL (or use Docker Compose)

### Running the Market Service

1. Navigate to the market directory:
```bash
cd market
```

2. Start the services with Docker Compose:
```bash
docker-compose up -d
```

This will start:
- PostgreSQL database on port 5432
- Market service on port 50051

3. The database will be automatically initialized with sample plugins.

### Building and Running the CLI

1. Generate gRPC code (if not already done):
```bash
cd proto
make generate
```

2. Navigate to CLI directory:
```bash
cd ../cli
```

3. Install dependencies:
```bash
go mod download
```

4. Build the CLI:
```bash
go build -o k8s-manager ./main.go
```

5. Run the CLI:
```bash
./k8s-manager
```

Or with custom market service address:
```bash
./k8s-manager -market-addr localhost:50051
```

## Usage

### CLI Controls

- `↑/↓` or `j/k`: Navigate through plugins
- `Enter`: View plugin details
- `i`: Install selected plugin
- `b` or `Esc`: Go back
- `q`: Quit

### Market Service Configuration

The market service can be configured via command-line flags:

```bash
-market-service -port 50051 -db-host localhost -db-port 5432 -db-user postgres -db-pass postgres -db-name k8s_market
```

## Project Structure

```
k8s-manager/
├── cli/                    # CLI application
│   ├── main.go            # Entry point
│   ├── internal/
│   │   ├── tui/           # TUI components
│   │   └── k8s/           # Kubernetes client
│   └── go.mod
├── market/                 # Market service
│   ├── main.go            # gRPC server
│   ├── migrations/        # Database migrations
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── go.mod
└── proto/                  # gRPC definitions
    ├── market.proto
    └── Makefile
```

## Development

### Generating gRPC Code

```bash
cd proto
make generate
```

### Running Tests

```bash
# CLI tests
cd cli
go test ./...

# Market service tests
cd market
go test ./...
```

### Adding New Plugins

Plugins can be added to the database via SQL migrations or through the market service API.

## License

MIT

