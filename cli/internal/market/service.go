package market

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"k8s-manager/cli/internal/auth"
	marketv1 "k8s-manager/proto/gen/v1/market"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const defaultAddress = "localhost:8080"

// Config - конфигурация сервиса маркетплейса
type Config struct {
	// Address - host:port gRPC-эндпоинта маркета
	Address string
}

// LoadConfig читает MARKET_GRPC_ADDR, иначе берёт дефолт
func LoadConfig() Config {
	address := strings.TrimSpace(os.Getenv("MARKET_GRPC_ADDR"))
	if address == "" {
		address = defaultAddress
	}

	return Config{Address: address}
}

// Service - обёртка над сгенерированным gRPC-клиентом маркетплейса.
type Service struct {
	cfg         Config
	authService *auth.Service
}

func NewService(cfg Config, authService *auth.Service) *Service {
	return &Service{
		cfg:         cfg,
		authService: authService,
	}
}

// PluginSummary - представление marketv1.Plugin, удобное для
// TUI. Без enum-префиксов и с флагами про установку
type PluginSummary struct {
	ID          int64
	Identifier  string
	Name        string
	Description string
	Category    string
	TrustStatus string
	Status      string
	InstalledAt string
	Installed   bool
}

// PluginList - результат ListPlugins. Total приходит от сервера и может
// быть больше len(Items), потому что выдача лимитируется
type PluginList struct {
	Items     []PluginSummary
	Total     int64
	Installed bool // true если это срез "мои установленные", а не результат поиска
}

// UploadedArtifact - ответ сервера после успешного стрим-аплоада
type UploadedArtifact struct {
	ID       int64
	Size     int64
	Checksum string
}

// DeveloperPluginDraft - поля, которые пользователь заполняет
// в форме при создании плагина
type DeveloperPluginDraft struct {
	PublisherName string
	Identifier    string
	Name          string
	Description   string
	Category      string
	Version       string
	ZipPath       string // путь к zip-артефакту на диске
	OS            string
	Arch          string
}

// DeveloperPlugin - модель того, что вернулось после полного флоу создания.
type DeveloperPlugin struct {
	PublisherID int64
	PluginID    int64
	ReleaseID   int64
	ArtifactID  int64
	Name        string
	Version     string
}

// ListPlugins предоставляет список плагинов.
//
// При пустом query возвращает только установленные пользователем плагины.
// При непустом query ищет по каталогу и помечает установленные.
func (s *Service) ListPlugins(ctx context.Context, query string) (*PluginList, error) {
	query = strings.TrimSpace(query)

	conn, ctx, closeClient, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer closeClient()
	client := marketv1.NewPluginServiceClient(conn)

	if query == "" {
		resp, err := client.ListInstalledPlugins(ctx, &marketv1.ListInstalledPluginsRequest{Limit: 50})
		if err != nil {
			return nil, fmt.Errorf("list installed plugins: %w", err)
		}

		items := make([]PluginSummary, 0, len(resp.InstalledPlugins))
		for _, installed := range resp.InstalledPlugins {
			item := pluginSummary(installed.Plugin)
			item.InstalledAt = installed.InstalledAt
			item.Installed = true
			items = append(items, item)
		}

		return &PluginList{
			Items:     items,
			Total:     resp.Total,
			Installed: true,
		}, nil
	}

	resp, err := client.ListPlugins(ctx, &marketv1.ListPluginsRequest{
		Query:  query,
		Status: marketv1.PluginStatus_PLUGIN_STATUS_ACTIVE,
		Limit:  50,
	})
	if err != nil {
		return nil, fmt.Errorf("search plugins: %w", err)
	}

	items := make([]PluginSummary, 0, len(resp.Plugins))
	for _, plugin := range resp.Plugins {
		items = append(items, pluginSummary(plugin))
	}

	// Подгружаем библиотеку пользователя и склеиваем с результатами поиска
	installedResp, err := client.ListInstalledPlugins(ctx, &marketv1.ListInstalledPluginsRequest{Limit: 100})
	if err != nil {
		return nil, fmt.Errorf("list installed plugins: %w", err)
	}
	installedAtByID := make(map[int64]string, len(installedResp.InstalledPlugins))
	for _, installed := range installedResp.InstalledPlugins {
		if installed.Plugin != nil {
			installedAtByID[installed.Plugin.Id] = installed.InstalledAt
		}
	}
	for i := range items {
		installedAt, installed := installedAtByID[items[i].ID]
		items[i].Installed = installed
		items[i].InstalledAt = installedAt
	}

	return &PluginList{
		Items: items,
		Total: resp.Total,
	}, nil
}

// InstallPlugin добавляет плагин в библиотеку пользователя. Сервер
// проверит только существование плагина.
func (s *Service) InstallPlugin(ctx context.Context, pluginID int64) (*PluginSummary, error) {
	conn, ctx, closeClient, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer closeClient()
	client := marketv1.NewPluginServiceClient(conn)

	resp, err := client.InstallPlugin(ctx, &marketv1.InstallPluginRequest{PluginId: pluginID})
	if err != nil {
		return nil, fmt.Errorf("install plugin: %w", err)
	}
	if resp.InstalledPlugin == nil {
		return nil, fmt.Errorf("install plugin returned empty response")
	}

	summary := pluginSummary(resp.InstalledPlugin.Plugin)
	summary.InstalledAt = resp.InstalledPlugin.InstalledAt
	return &summary, nil
}

// UninstallPlugin убирает плагин из библиотеки пользователя
func (s *Service) UninstallPlugin(ctx context.Context, pluginID int64) error {
	conn, ctx, closeClient, err := s.connect(ctx)
	if err != nil {
		return err
	}
	defer closeClient()
	client := marketv1.NewPluginServiceClient(conn)

	if _, err := client.UninstallPlugin(ctx, &marketv1.UninstallPluginRequest{PluginId: pluginID}); err != nil {
		return fmt.Errorf("uninstall plugin: %w", err)
	}

	return nil
}

// UploadArtifact заливает zip-файл по client-streaming RPC
func (s *Service) UploadArtifact(ctx context.Context, releaseID int64, zipPath, osName, arch string) (*UploadedArtifact, error) {
	conn, ctx, closeClient, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer closeClient()

	return uploadArtifact(ctx, marketv1.NewPluginServiceClient(conn), releaseID, zipPath, osName, arch)
}

// CreateDeveloperPlugin совершает полную цепочку публикации плагина.
// Trust-статус ставится COMMUNITY.
func (s *Service) CreateDeveloperPlugin(ctx context.Context, draft DeveloperPluginDraft) (*DeveloperPlugin, error) {
	draft.PublisherName = strings.TrimSpace(draft.PublisherName)
	draft.Identifier = strings.TrimSpace(draft.Identifier)
	draft.Name = strings.TrimSpace(draft.Name)
	draft.Version = strings.TrimSpace(draft.Version)
	draft.ZipPath = strings.TrimSpace(draft.ZipPath)

	if draft.PublisherName == "" {
		return nil, fmt.Errorf("publisher name is required")
	}
	if draft.Identifier == "" {
		return nil, fmt.Errorf("plugin identifier is required")
	}
	if draft.Name == "" {
		return nil, fmt.Errorf("plugin name is required")
	}
	if draft.Version == "" {
		return nil, fmt.Errorf("release version is required")
	}
	if draft.ZipPath == "" {
		return nil, fmt.Errorf("artifact zip path is required")
	}

	conn, ctx, closeClient, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer closeClient()

	pluginClient := marketv1.NewPluginServiceClient(conn)
	publisherClient := marketv1.NewPublisherServiceClient(conn)

	publisherResp, err := publisherClient.CreatePublisher(ctx, &marketv1.CreatePublisherRequest{
		Name: draft.PublisherName,
	})
	if err != nil {
		return nil, fmt.Errorf("create publisher: %w", err)
	}
	if publisherResp.Publisher == nil {
		return nil, fmt.Errorf("create publisher returned empty response")
	}

	pluginResp, err := pluginClient.CreatePlugin(ctx, &marketv1.CreatePluginRequest{
		Identifier:  draft.Identifier,
		Name:        draft.Name,
		Description: draft.Description,
		Category:    draft.Category,
		PublisherId: publisherResp.Publisher.Id,
		TrustStatus: marketv1.TrustStatus_TRUST_STATUS_COMMUNITY,
	})
	if err != nil {
		return nil, fmt.Errorf("create plugin: %w", err)
	}
	if pluginResp.Plugin == nil {
		return nil, fmt.Errorf("create plugin returned empty response")
	}

	releaseResp, err := pluginClient.CreateRelease(ctx, &marketv1.CreateReleaseRequest{
		PluginId: pluginResp.Plugin.Id,
		Version:  draft.Version,
		IsLatest: true,
	})
	if err != nil {
		return nil, fmt.Errorf("create release: %w", err)
	}
	if releaseResp.Release == nil {
		return nil, fmt.Errorf("create release returned empty response")
	}

	artifact, err := uploadArtifact(ctx, pluginClient, releaseResp.Release.Id, draft.ZipPath, draft.OS, draft.Arch)
	if err != nil {
		return nil, err
	}

	return &DeveloperPlugin{
		PublisherID: publisherResp.Publisher.Id,
		PluginID:    pluginResp.Plugin.Id,
		ReleaseID:   releaseResp.Release.Id,
		ArtifactID:  artifact.ID,
		Name:        pluginResp.Plugin.Name,
		Version:     releaseResp.Release.Version,
	}, nil
}

// uploadArtifact заливает zip-файл по client-streaming RPC.
//
// Сначала отправляет метаданные артефакта, затем чанки по 64KB
func uploadArtifact(ctx context.Context, client marketv1.PluginServiceClient, releaseID int64, zipPath, osName, arch string) (*UploadedArtifact, error) {
	zipFile, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("artifact must be a readable zip archive: %w", err)
	}
	_ = zipFile.Close()

	file, err := os.Open(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open artifact: %w", err)
	}
	defer file.Close()

	// Если пользователь не указал платформу явно, то считаем, что собирал под текущую
	if osName == "" {
		osName = runtime.GOOS
	}
	if arch == "" {
		arch = runtime.GOARCH
	}

	stream, err := client.UploadArtifact(ctx)
	if err != nil {
		return nil, fmt.Errorf("start artifact upload: %w", err)
	}

	if err := stream.Send(&marketv1.UploadArtifactRequest{
		Data: &marketv1.UploadArtifactRequest_Metadata{
			Metadata: &marketv1.ArtifactMetadata{
				ReleaseId:        releaseID,
				Os:               osName,
				Arch:             arch,
				Type:             "zip",
				OriginalFilename: filepath.Base(zipPath),
			},
		},
	}); err != nil {
		return nil, fmt.Errorf("send artifact metadata: %w", err)
	}

	buf := make([]byte, 64*1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&marketv1.UploadArtifactRequest{
				Data: &marketv1.UploadArtifactRequest_Chunk{Chunk: buf[:n]},
			}); sendErr != nil {
				return nil, fmt.Errorf("send artifact chunk: %w", sendErr)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read artifact: %w", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("finish artifact upload: %w", err)
	}
	if resp.Artifact == nil {
		return nil, fmt.Errorf("artifact upload returned empty response")
	}

	return &UploadedArtifact{
		ID:       resp.Artifact.Id,
		Size:     resp.Artifact.Size,
		Checksum: resp.Artifact.Checksum,
	}, nil
}

// connect готовит всё для одного RPC-вызова.
//
// Достаёт действительный access-токен, открывает gRPC-соединение и
// добавляет к контексту access-токен.
func (s *Service) connect(ctx context.Context) (*grpc.ClientConn, context.Context, func(), error) {
	if s.authService == nil {
		return nil, ctx, nil, fmt.Errorf("auth service is not configured")
	}

	token, err := s.authService.AttachToken(ctx)
	if err != nil {
		return nil, ctx, nil, fmt.Errorf("load auth token: %w", err)
	}

	conn, err := grpc.NewClient(
		s.cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, ctx, nil, fmt.Errorf("connect to market service: %w", err)
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	return conn, ctx, func() { _ = conn.Close() }, nil
}

// pluginSummary конвертирует marketv1.Plugin в PluginSummary.
// Удаляет enum-префиксы.
func pluginSummary(plugin *marketv1.Plugin) PluginSummary {
	if plugin == nil {
		return PluginSummary{}
	}

	return PluginSummary{
		ID:          plugin.Id,
		Identifier:  plugin.Identifier,
		Name:        plugin.Name,
		Description: plugin.Description,
		Category:    plugin.Category,
		TrustStatus: strings.TrimPrefix(plugin.TrustStatus.String(), "TRUST_STATUS_"),
		Status:      strings.TrimPrefix(plugin.Status.String(), "PLUGIN_STATUS_"),
	}
}
