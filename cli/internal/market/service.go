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
	"sync"

	"k8s-manager/cli/internal/auth"
	marketv1 "k8s-manager/proto/gen/v1/market"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

	// Кэш publisher_id -> имя. Издателей мало и они редко меняются,
	// поэтому держим простой in-memory кэш на время жизни процесса
	publisherCacheMu sync.Mutex
	publisherCache   map[int64]string
}

func NewService(cfg Config, authService *auth.Service) *Service {
	return &Service{
		cfg:            cfg,
		authService:    authService,
		publisherCache: make(map[int64]string),
	}
}

// PluginSummary - представление marketv1.Plugin, удобное для
// TUI. Без enum-префиксов и с флагами про установку
type PluginSummary struct {
	ID            int64
	Identifier    string
	Name          string
	Description   string
	Category      string
	PublisherID   int64
	PublisherName string // подмешивается через кэш в ListPlugins
	TrustStatus   string
	Status        string
	InstalledAt   string
	Installed     bool
}

// PluginDetails - расширенная инфо о плагине, нужна детальной странице.
// Содержит всё, что есть в Summary, плюс ссылки и таймстампы
type PluginDetails struct {
	PluginSummary
	SourceURL string
	DocsURL   string
	CreatedAt string
	UpdatedAt string
}

// Release - один релиз плагина (версия). Артефакты привязаны к релизу
type Release struct {
	ID            int64
	PluginID      int64
	Version       string
	PublishedAt   string
	Changelog     string
	MinCLIVersion string
	MinK8sVersion string
	IsLatest      bool
}

// Artifact - бинарник под конкретную платформу одного релиза
type Artifact struct {
	ID        int64
	ReleaseID int64
	OS        string
	Arch      string
	Type      string
	Size      int64
	Checksum  string
}

// PluginList - результат ListPlugins. Total приходит от сервера и может
// быть больше len(Items), потому что выдача лимитируется
type PluginList struct {
	Items     []PluginSummary
	Total     int64
	Installed bool // true если это добавленные в библиотеку (при пустом запросе), а не результат поиска
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
			return nil, rpcError("list installed plugins", err)
		}

		items := make([]PluginSummary, 0, len(resp.InstalledPlugins))
		for _, installed := range resp.InstalledPlugins {
			item := pluginSummary(installed.Plugin)
			item.InstalledAt = installed.InstalledAt
			item.Installed = true
			items = append(items, item)
		}
		s.fillPublisherNames(ctx, conn, items)

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
		return nil, rpcError("search plugins", err)
	}

	items := make([]PluginSummary, 0, len(resp.Plugins))
	for _, plugin := range resp.Plugins {
		items = append(items, pluginSummary(plugin))
	}

	// Подгружаем библиотеку пользователя и склеиваем с результатами поиска
	installedResp, err := client.ListInstalledPlugins(ctx, &marketv1.ListInstalledPluginsRequest{Limit: 100})
	if err != nil {
		return nil, rpcError("list installed plugins", err)
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
	s.fillPublisherNames(ctx, conn, items)

	return &PluginList{
		Items: items,
		Total: resp.Total,
	}, nil
}

// GetPlugin возвращает расширенную инфо о плагине для детальной страницы
func (s *Service) GetPlugin(ctx context.Context, pluginID int64) (*PluginDetails, error) {
	conn, ctx, closeClient, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer closeClient()

	pluginClient := marketv1.NewPluginServiceClient(conn)
	resp, err := pluginClient.GetPlugin(ctx, &marketv1.GetPluginRequest{Id: pluginID})
	if err != nil {
		return nil, fmt.Errorf("get plugin: %w", err)
	}
	if resp.Plugin == nil {
		return nil, fmt.Errorf("get plugin returned empty response")
	}

	details := &PluginDetails{
		PluginSummary: pluginSummary(resp.Plugin),
		SourceURL:     resp.Plugin.SourceUrl,
		DocsURL:       resp.Plugin.DocsUrl,
		CreatedAt:     resp.Plugin.CreatedAt,
		UpdatedAt:     resp.Plugin.UpdatedAt,
	}

	// Подтягиваем имя издателя из кэша (или сходим за ним один раз)
	if name, ok := s.publisherName(ctx, conn, details.PublisherID); ok {
		details.PublisherName = name
	}

	return details, nil
}

// ListReleases возвращает релизы плагина, отсортированные сервером
func (s *Service) ListReleases(ctx context.Context, pluginID int64) ([]Release, error) {
	conn, ctx, closeClient, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer closeClient()

	client := marketv1.NewPluginServiceClient(conn)
	resp, err := client.ListReleases(ctx, &marketv1.ListReleasesRequest{
		PluginId: pluginID,
		Limit:    50,
	})
	if err != nil {
		return nil, fmt.Errorf("list releases: %w", err)
	}

	out := make([]Release, 0, len(resp.Releases))
	for _, r := range resp.Releases {
		out = append(out, toRelease(r))
	}
	return out, nil
}

// ListArtifacts возвращает артефакты конкретного релиза (по одному на платформу)
func (s *Service) ListArtifacts(ctx context.Context, releaseID int64) ([]Artifact, error) {
	conn, ctx, closeClient, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer closeClient()

	client := marketv1.NewPluginServiceClient(conn)
	resp, err := client.ListArtifacts(ctx, &marketv1.ListArtifactsRequest{ReleaseId: releaseID})
	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}

	out := make([]Artifact, 0, len(resp.Artifacts))
	for _, a := range resp.Artifacts {
		out = append(out, toArtifact(a))
	}
	return out, nil
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
		return nil, rpcError("install plugin", err)
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
		return rpcError("uninstall plugin", err)
	}

	return nil
}

// DownloadArtifact стримит байты артефакта в w и возвращает количество скачанных байт.
// Верификация checksum и размера лежит на вызывающей стороне.
func (s *Service) DownloadArtifact(ctx context.Context, artifactID int64, w io.Writer) (int64, error) {
	if w == nil {
		return 0, fmt.Errorf("download artifact: writer is required")
	}

	conn, ctx, closeClient, err := s.connect(ctx)
	if err != nil {
		return 0, err
	}
	defer closeClient()

	client := marketv1.NewPluginServiceClient(conn)
	stream, err := client.DownloadArtifact(ctx, &marketv1.DownloadArtifactRequest{Id: artifactID})
	if err != nil {
		return 0, rpcError("start artifact download", err)
	}

	var written int64
	for {
		resp, recvErr := stream.Recv()
		if recvErr == io.EOF {
			return written, nil
		}
		if recvErr != nil {
			return written, rpcError("receive artifact chunk", recvErr)
		}
		if len(resp.Chunk) == 0 {
			continue
		}
		n, writeErr := w.Write(resp.Chunk)
		written += int64(n)
		if writeErr != nil {
			return written, fmt.Errorf("write artifact chunk: %w", writeErr)
		}
	}
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
		return nil, rpcError("create publisher", err)
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
		return nil, rpcError("create plugin", err)
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
		return nil, rpcError("create release", err)
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
		return nil, rpcError("start artifact upload", err)
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
		return nil, rpcError("send artifact metadata", err)
	}

	buf := make([]byte, 64*1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&marketv1.UploadArtifactRequest{
				Data: &marketv1.UploadArtifactRequest_Chunk{Chunk: buf[:n]},
			}); sendErr != nil {
				return nil, rpcError("send artifact chunk", sendErr)
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
		return nil, rpcError("finish artifact upload", err)
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

// toRelease конвертирует protobuf-релиз в локальный тип
func toRelease(r *marketv1.Release) Release {
	if r == nil {
		return Release{}
	}
	return Release{
		ID:            r.Id,
		PluginID:      r.PluginId,
		Version:       r.Version,
		PublishedAt:   r.PublishedAt,
		Changelog:     r.Changelog,
		MinCLIVersion: r.MinCliVersion,
		MinK8sVersion: r.MinK8SVersion,
		IsLatest:      r.IsLatest,
	}
}

// toArtifact конвертирует protobuf-артефакт в локальный тип
func toArtifact(a *marketv1.Artifact) Artifact {
	if a == nil {
		return Artifact{}
	}
	return Artifact{
		ID:        a.Id,
		ReleaseID: a.ReleaseId,
		OS:        a.Os,
		Arch:      a.Arch,
		Type:      a.Type,
		Size:      a.Size,
		Checksum:  a.Checksum,
	}
}

// fillPublisherNames проставляет PublisherName в каждом элементе списка,
// используя существующее соединение. При ошибке тихо оставляет имена пустыми -
// поле опциональное, лучше показать список без издателей чем уронить запрос
func (s *Service) fillPublisherNames(ctx context.Context, conn *grpc.ClientConn, items []PluginSummary) {
	if len(items) == 0 {
		return
	}

	// Собираем уникальные id, которых ещё нет в кэше
	s.publisherCacheMu.Lock()
	missing := make(map[int64]struct{})
	for _, item := range items {
		if item.PublisherID == 0 {
			continue
		}
		if _, ok := s.publisherCache[item.PublisherID]; !ok {
			missing[item.PublisherID] = struct{}{}
		}
	}
	s.publisherCacheMu.Unlock()

	if len(missing) > 0 {
		client := marketv1.NewPublisherServiceClient(conn)
		resp, err := client.ListPublishers(ctx, &marketv1.ListPublishersRequest{Limit: 200})
		if err == nil && resp != nil {
			s.publisherCacheMu.Lock()
			for _, p := range resp.Publishers {
				s.publisherCache[p.Id] = p.Name
			}
			s.publisherCacheMu.Unlock()
		}
	}

	// Проставляем имена из кэша
	s.publisherCacheMu.Lock()
	defer s.publisherCacheMu.Unlock()
	for i := range items {
		if name, ok := s.publisherCache[items[i].PublisherID]; ok {
			items[i].PublisherName = name
		}
	}
}

// publisherName возвращает имя издателя из кэша или дёргает сервер если в кэше нет
func (s *Service) publisherName(ctx context.Context, conn *grpc.ClientConn, publisherID int64) (string, bool) {
	if publisherID == 0 {
		return "", false
	}

	s.publisherCacheMu.Lock()
	if name, ok := s.publisherCache[publisherID]; ok {
		s.publisherCacheMu.Unlock()
		return name, true
	}
	s.publisherCacheMu.Unlock()

	client := marketv1.NewPublisherServiceClient(conn)
	resp, err := client.GetPublisher(ctx, &marketv1.GetPublisherRequest{Id: publisherID})
	if err != nil || resp == nil || resp.Publisher == nil {
		return "", false
	}

	s.publisherCacheMu.Lock()
	s.publisherCache[publisherID] = resp.Publisher.Name
	s.publisherCacheMu.Unlock()
	return resp.Publisher.Name, true
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
		PublisherID: plugin.PublisherId,
		TrustStatus: strings.TrimPrefix(plugin.TrustStatus.String(), "TRUST_STATUS_"),
		Status:      strings.TrimPrefix(plugin.Status.String(), "PLUGIN_STATUS_"),
	}
}

func rpcError(operation string, err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("%s: %w", operation, err)
	}

	return fmt.Errorf("%s: %s (%s)", operation, st.Message(), st.Code())
}
