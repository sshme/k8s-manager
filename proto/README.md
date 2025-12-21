# Proto Definitions

Protocol Buffer definitions for K8S-MANAGER services.

## Structure

- `v1/` - Version 1 API definitions
  - `user.proto` - User service definitions
  - `market.proto` - Market service definitions (plugins, releases, artifacts)

## Generation

### Prerequisites

Install required tools:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Generate Go code

From project root:

```bash
# Generate user service
protoc --go_out=. --go_opt=paths=import --go-grpc_out=. --go-grpc_opt=paths=import proto/v1/user.proto

# Generate market service
protoc --go_out=. --go_opt=paths=import --go-grpc_out=. --go-grpc_opt=paths=import proto/v1/market.proto

# Or use Task
cd proto/v1
task all
```

Generated code will be in `proto/gen/v1/`.

## Market Service API

The market service provides gRPC API for managing plugins:

### PluginService

- `CreatePlugin` - Create a new plugin
- `GetPlugin` - Get plugin by ID
- `ListPlugins` - List plugins with filtering and pagination
- `UpdatePlugin` - Update plugin metadata
- `UpdatePluginStatus` - Update plugin status (active/hidden/blocked)

### Release Management

- `CreateRelease` - Create a new plugin release
- `GetRelease` - Get release by ID
- `ListReleases` - List releases for a plugin
- `GetLatestRelease` - Get latest release for a plugin

### Artifact Management

- `UploadArtifact` - Upload artifact (streaming)
- `GetArtifact` - Get artifact by ID
- `GetArtifactByPlatform` - Get artifact by OS/Arch
- `ListArtifacts` - List artifacts for a release
- `DownloadArtifact` - Download artifact (streaming)
- `DeleteArtifact` - Delete artifact

### PublisherService

- `CreatePublisher` - Create a new publisher
- `GetPublisher` - Get publisher by ID
- `ListPublishers` - List publishers
