package plugin

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	appplugin "k8s-manager/market/internal/application/plugin"
	domainplugin "k8s-manager/market/internal/domain/plugin"
	"k8s-manager/market/internal/infrastructure/storage"
	"k8s-manager/market/internal/presentation/grpc/auth"
	marketv1 "k8s-manager/proto/gen/v1/market"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler implements the gRPC PluginService
type Handler struct {
	marketv1.UnimplementedPluginServiceServer
	marketv1.UnimplementedPublisherServiceServer
	service *appplugin.Service
	storage storage.ArtifactStorage
}

// NewHandler creates a new gRPC plugin handler
func NewHandler(service *appplugin.Service, storage storage.ArtifactStorage) *Handler {
	return &Handler{
		service: service,
		storage: storage,
	}
}

// CreatePlugin handles CreatePlugin gRPC request
func (h *Handler) CreatePlugin(ctx context.Context, req *marketv1.CreatePluginRequest) (*marketv1.CreatePluginResponse, error) {
	createReq := &appplugin.CreatePluginRequest{
		Identifier:  req.Identifier,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		PublisherID: req.PublisherId,
		TrustStatus: fromProtoTrustStatus(req.TrustStatus),
		SourceURL:   req.SourceUrl,
		DocsURL:     req.DocsUrl,
	}

	p, err := h.service.CreatePlugin(contextWithClaimsUserID(ctx), createReq)
	if err != nil {
		return nil, grpcError("create plugin", err)
	}

	return &marketv1.CreatePluginResponse{
		Plugin: toProtoPlugin(p),
	}, nil
}

// GetPlugin handles GetPlugin gRPC request
func (h *Handler) GetPlugin(ctx context.Context, req *marketv1.GetPluginRequest) (*marketv1.GetPluginResponse, error) {
	p, err := h.service.GetPlugin(ctx, req.Id)
	if err != nil {
		return nil, grpcError("get plugin", err)
	}

	return &marketv1.GetPluginResponse{
		Plugin: toProtoPlugin(p),
	}, nil
}

// ListPlugins handles ListPlugins gRPC request
func (h *Handler) ListPlugins(ctx context.Context, req *marketv1.ListPluginsRequest) (*marketv1.ListPluginsResponse, error) {
	listReq := &appplugin.ListPluginsRequest{
		Name:        req.Name,
		Query:       req.Query,
		Category:    req.Category,
		PublisherID: req.PublisherId,
		TrustStatus: fromProtoTrustStatus(req.TrustStatus),
		Status:      fromProtoPluginStatus(req.Status),
		Limit:       int(req.Limit),
		Offset:      int(req.Offset),
	}

	plugins, total, err := h.service.ListPlugins(ctx, listReq)
	if err != nil {
		return nil, grpcError("list plugins", err)
	}

	protoPlugins := make([]*marketv1.Plugin, len(plugins))
	for i, p := range plugins {
		protoPlugins[i] = toProtoPlugin(p)
	}

	return &marketv1.ListPluginsResponse{
		Plugins: protoPlugins,
		Total:   total,
	}, nil
}

// UpdatePlugin handles UpdatePlugin gRPC request
func (h *Handler) UpdatePlugin(ctx context.Context, req *marketv1.UpdatePluginRequest) (*marketv1.UpdatePluginResponse, error) {
	updateReq := &appplugin.UpdatePluginRequest{
		ID:          req.Id,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		SourceURL:   req.SourceUrl,
		DocsURL:     req.DocsUrl,
	}

	p, err := h.service.UpdatePlugin(ctx, updateReq)
	if err != nil {
		return nil, grpcError("update plugin", err)
	}

	return &marketv1.UpdatePluginResponse{
		Plugin: toProtoPlugin(p),
	}, nil
}

// UpdatePluginStatus handles UpdatePluginStatus gRPC request
func (h *Handler) UpdatePluginStatus(ctx context.Context, req *marketv1.UpdatePluginStatusRequest) (*marketv1.UpdatePluginStatusResponse, error) {
	err := h.service.UpdatePluginStatus(ctx, req.Id, fromProtoPluginStatus(req.Status), req.Reason)
	if err != nil {
		return nil, grpcError("update plugin status", err)
	}

	return &marketv1.UpdatePluginStatusResponse{}, nil
}

// UpdatePluginTrustStatus handles UpdatePluginTrustStatus gRPC request
func (h *Handler) UpdatePluginTrustStatus(ctx context.Context, req *marketv1.UpdatePluginTrustStatusRequest) (*marketv1.UpdatePluginTrustStatusResponse, error) {
	err := h.service.UpdatePluginTrustStatus(contextWithClaimsUserID(ctx), req.Id, fromProtoTrustStatus(req.TrustStatus), req.Reason)
	if err != nil {
		return nil, grpcError("update plugin trust status", err)
	}

	return &marketv1.UpdatePluginTrustStatusResponse{}, nil
}

// CreateRelease handles CreateRelease gRPC request
func (h *Handler) CreateRelease(ctx context.Context, req *marketv1.CreateReleaseRequest) (*marketv1.CreateReleaseResponse, error) {
	createReq := &appplugin.CreateReleaseRequest{
		PluginID:      req.PluginId,
		Version:       req.Version,
		Changelog:     req.Changelog,
		MinCLIVersion: req.MinCliVersion,
		MinK8sVersion: req.MinK8SVersion,
		IsLatest:      req.IsLatest,
	}

	rel, err := h.service.CreateRelease(contextWithClaimsUserID(ctx), createReq)
	if err != nil {
		return nil, grpcError("create release", err)
	}

	return &marketv1.CreateReleaseResponse{
		Release: toProtoRelease(rel),
	}, nil
}

// GetRelease handles GetRelease gRPC request
func (h *Handler) GetRelease(ctx context.Context, req *marketv1.GetReleaseRequest) (*marketv1.GetReleaseResponse, error) {
	rel, err := h.service.GetRelease(ctx, req.Id)
	if err != nil {
		return nil, grpcError("get release", err)
	}

	return &marketv1.GetReleaseResponse{
		Release: toProtoRelease(rel),
	}, nil
}

// ListReleases handles ListReleases gRPC request
func (h *Handler) ListReleases(ctx context.Context, req *marketv1.ListReleasesRequest) (*marketv1.ListReleasesResponse, error) {
	releases, total, err := h.service.ListReleases(ctx, req.PluginId, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, grpcError("list releases", err)
	}

	protoReleases := make([]*marketv1.Release, len(releases))
	for i, r := range releases {
		protoReleases[i] = toProtoRelease(r)
	}

	return &marketv1.ListReleasesResponse{
		Releases: protoReleases,
		Total:    total,
	}, nil
}

// GetLatestRelease handles GetLatestRelease gRPC request
func (h *Handler) GetLatestRelease(ctx context.Context, req *marketv1.GetLatestReleaseRequest) (*marketv1.GetLatestReleaseResponse, error) {
	rel, err := h.service.GetLatestRelease(ctx, req.PluginId)
	if err != nil {
		return nil, grpcError("get latest release", err)
	}

	return &marketv1.GetLatestReleaseResponse{
		Release: toProtoRelease(rel),
	}, nil
}

// Helper functions to convert domain models to proto
func toProtoPlugin(p *domainplugin.Plugin) *marketv1.Plugin {
	if p == nil {
		return nil
	}

	return &marketv1.Plugin{
		Id:          p.ID,
		Identifier:  p.Identifier,
		Name:        p.Name,
		Description: p.Description,
		Category:    p.Category,
		PublisherId: p.PublisherID,
		Status:      toProtoPluginStatus(p.Status),
		TrustStatus: toProtoTrustStatus(p.TrustStatus),
		SourceUrl:   p.SourceURL,
		DocsUrl:     p.DocsURL,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProtoPluginStatus(s domainplugin.PluginStatus) marketv1.PluginStatus {
	switch s {
	case domainplugin.PluginStatusActive:
		return marketv1.PluginStatus_PLUGIN_STATUS_ACTIVE
	case domainplugin.PluginStatusHidden:
		return marketv1.PluginStatus_PLUGIN_STATUS_HIDDEN
	case domainplugin.PluginStatusBlocked:
		return marketv1.PluginStatus_PLUGIN_STATUS_BLOCKED
	default:
		return marketv1.PluginStatus_PLUGIN_STATUS_UNKNOWN
	}
}

func toProtoTrustStatus(s domainplugin.TrustStatus) marketv1.TrustStatus {
	switch s {
	case domainplugin.TrustStatusOfficial:
		return marketv1.TrustStatus_TRUST_STATUS_OFFICIAL
	case domainplugin.TrustStatusVerified:
		return marketv1.TrustStatus_TRUST_STATUS_VERIFIED
	case domainplugin.TrustStatusCommunity:
		return marketv1.TrustStatus_TRUST_STATUS_COMMUNITY
	default:
		return marketv1.TrustStatus_TRUST_STATUS_UNKNOWN
	}
}

func fromProtoPluginStatus(s marketv1.PluginStatus) domainplugin.PluginStatus {
	switch s {
	case marketv1.PluginStatus_PLUGIN_STATUS_ACTIVE:
		return domainplugin.PluginStatusActive
	case marketv1.PluginStatus_PLUGIN_STATUS_HIDDEN:
		return domainplugin.PluginStatusHidden
	case marketv1.PluginStatus_PLUGIN_STATUS_BLOCKED:
		return domainplugin.PluginStatusBlocked
	default:
		return domainplugin.PluginStatusActive
	}
}

func fromProtoTrustStatus(s marketv1.TrustStatus) domainplugin.TrustStatus {
	switch s {
	case marketv1.TrustStatus_TRUST_STATUS_OFFICIAL:
		return domainplugin.TrustStatusOfficial
	case marketv1.TrustStatus_TRUST_STATUS_VERIFIED:
		return domainplugin.TrustStatusVerified
	case marketv1.TrustStatus_TRUST_STATUS_COMMUNITY:
		return domainplugin.TrustStatusCommunity
	default:
		return domainplugin.TrustStatusCommunity
	}
}

func toProtoRelease(r *domainplugin.Release) *marketv1.Release {
	return &marketv1.Release{
		Id:            r.ID,
		PluginId:      r.PluginID,
		Version:       r.Version,
		PublishedAt:   r.PublishedAt.Format("2006-01-02T15:04:05Z07:00"),
		Changelog:     r.Changelog,
		MinCliVersion: r.MinCLIVersion,
		MinK8SVersion: r.MinK8sVersion,
		IsLatest:      r.IsLatest,
		CreatedAt:     r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProtoArtifact(a *domainplugin.Artifact) *marketv1.Artifact {
	return &marketv1.Artifact{
		Id:          a.ID,
		ReleaseId:   a.ReleaseID,
		Os:          a.OS,
		Arch:        a.Arch,
		Type:        a.Type,
		Size:        a.Size,
		Checksum:    a.Checksum,
		StoragePath: a.StoragePath,
		Signature:   a.Signature,
		KeyId:       a.KeyID,
		CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProtoInstalledPlugin(installed *domainplugin.PluginInstallation) *marketv1.InstalledPlugin {
	if installed == nil {
		return nil
	}

	return &marketv1.InstalledPlugin{
		Plugin:      toProtoPlugin(installed.Plugin),
		InstalledAt: installed.InstalledAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProtoPublisher(pub *domainplugin.Publisher) *marketv1.Publisher {
	if pub == nil {
		return nil
	}

	return &marketv1.Publisher{
		Id:          pub.ID,
		Name:        pub.Name,
		Description: pub.Description,
		WebsiteUrl:  pub.WebsiteURL,
		CreatedAt:   pub.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   pub.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// UploadArtifact handles UploadArtifact gRPC stream request
func (h *Handler) UploadArtifact(stream marketv1.PluginService_UploadArtifactServer) error {
	var metadata *marketv1.ArtifactMetadata
	var chunks bytes.Buffer

	// Receive stream
	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return grpcError("receive artifact upload", err)
		}

		switch data := req.Data.(type) {
		case *marketv1.UploadArtifactRequest_Metadata:
			metadata = data.Metadata
		case *marketv1.UploadArtifactRequest_Chunk:
			if _, err := chunks.Write(data.Chunk); err != nil {
				return status.Errorf(codes.Internal, "buffer artifact chunk: %v", err)
			}
		}
	}

	if metadata == nil {
		return status.Error(codes.InvalidArgument, "artifact metadata is required")
	}
	if metadata.ReleaseId == 0 {
		return status.Error(codes.InvalidArgument, "release_id is required")
	}
	if metadata.Os == "" || metadata.Arch == "" {
		return status.Error(codes.InvalidArgument, "os and arch are required")
	}

	data := chunks.Bytes()
	if len(data) == 0 {
		return status.Error(codes.InvalidArgument, "artifact content is required")
	}
	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
		return status.Errorf(codes.InvalidArgument, "artifact must be a valid zip archive: %v", err)
	}

	rel, err := h.service.GetRelease(stream.Context(), metadata.ReleaseId)
	if err != nil {
		return grpcError("get release", err)
	}

	filename := metadata.OriginalFilename
	if filename == "" {
		filename = "artifact.zip"
	}
	storagePath, checksum, size, err := h.storage.Save(rel.PluginID, rel.ID, metadata.Os, metadata.Arch, filename, bytes.NewReader(data))
	if err != nil {
		return status.Errorf(codes.Unavailable, "save artifact: %v", err)
	}

	artifactType := metadata.Type
	if artifactType == "" {
		artifactType = "zip"
	}

	artifact, err := h.service.CreateArtifact(stream.Context(), &appplugin.CreateArtifactRequest{
		ReleaseID:   metadata.ReleaseId,
		OS:          metadata.Os,
		Arch:        metadata.Arch,
		Type:        artifactType,
		Size:        size,
		Checksum:    checksum,
		StoragePath: storagePath,
		Signature:   metadata.Signature,
		KeyID:       metadata.KeyId,
	})
	if err != nil {
		_ = h.storage.Delete(storagePath)
		return grpcError("create artifact", err)
	}

	return stream.SendAndClose(&marketv1.UploadArtifactResponse{
		Artifact: toProtoArtifact(artifact),
	})
}

// GetArtifact handles GetArtifact gRPC request
func (h *Handler) GetArtifact(ctx context.Context, req *marketv1.GetArtifactRequest) (*marketv1.GetArtifactResponse, error) {
	art, err := h.service.GetArtifact(ctx, req.Id)
	if err != nil {
		return nil, grpcError("get artifact", err)
	}

	return &marketv1.GetArtifactResponse{
		Artifact: toProtoArtifact(art),
	}, nil
}

// GetArtifactByPlatform handles GetArtifactByPlatform gRPC request
func (h *Handler) GetArtifactByPlatform(ctx context.Context, req *marketv1.GetArtifactByPlatformRequest) (*marketv1.GetArtifactResponse, error) {
	art, err := h.service.GetArtifactByPlatform(ctx, req.ReleaseId, req.Os, req.Arch)
	if err != nil {
		return nil, grpcError("get artifact", err)
	}

	return &marketv1.GetArtifactResponse{
		Artifact: toProtoArtifact(art),
	}, nil
}

// ListArtifacts handles ListArtifacts gRPC request
func (h *Handler) ListArtifacts(ctx context.Context, req *marketv1.ListArtifactsRequest) (*marketv1.ListArtifactsResponse, error) {
	artifacts, err := h.service.ListArtifacts(ctx, req.ReleaseId)
	if err != nil {
		return nil, grpcError("list artifacts", err)
	}

	protoArtifacts := make([]*marketv1.Artifact, len(artifacts))
	for i, a := range artifacts {
		protoArtifacts[i] = toProtoArtifact(a)
	}

	return &marketv1.ListArtifactsResponse{
		Artifacts: protoArtifacts,
	}, nil
}

// DownloadArtifact handles DownloadArtifact gRPC stream request
func (h *Handler) DownloadArtifact(req *marketv1.DownloadArtifactRequest, stream marketv1.PluginService_DownloadArtifactServer) error {
	artifact, err := h.service.GetArtifact(stream.Context(), req.Id)
	if err != nil {
		return grpcError("get artifact", err)
	}

	file, err := h.storage.Get(artifact.StoragePath)
	if err != nil {
		return status.Errorf(codes.Unavailable, "open artifact: %v", err)
	}
	defer file.Close()

	buf := make([]byte, 64*1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&marketv1.DownloadArtifactResponse{Chunk: buf[:n]}); sendErr != nil {
				return grpcError("stream artifact", sendErr)
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Unavailable, "read artifact: %v", err)
		}
	}
}

// DeleteArtifact handles DeleteArtifact gRPC request
func (h *Handler) DeleteArtifact(ctx context.Context, req *marketv1.DeleteArtifactRequest) (*marketv1.DeleteArtifactResponse, error) {
	artifact, err := h.service.GetArtifact(ctx, req.Id)
	if err != nil {
		return nil, grpcError("get artifact", err)
	}

	err = h.service.DeleteArtifact(ctx, req.Id, req.Reason)
	if err != nil {
		return nil, grpcError("delete artifact", err)
	}
	_ = h.storage.Delete(artifact.StoragePath)

	return &marketv1.DeleteArtifactResponse{}, nil
}

// InstallPlugin handles InstallPlugin gRPC request.
func (h *Handler) InstallPlugin(ctx context.Context, req *marketv1.InstallPluginRequest) (*marketv1.InstallPluginResponse, error) {
	installed, err := h.service.InstallPlugin(contextWithClaimsUserID(ctx), req.PluginId)
	if err != nil {
		return nil, grpcError("install plugin", err)
	}

	return &marketv1.InstallPluginResponse{
		InstalledPlugin: toProtoInstalledPlugin(installed),
	}, nil
}

// UninstallPlugin handles UninstallPlugin gRPC request.
func (h *Handler) UninstallPlugin(ctx context.Context, req *marketv1.UninstallPluginRequest) (*marketv1.UninstallPluginResponse, error) {
	if err := h.service.UninstallPlugin(contextWithClaimsUserID(ctx), req.PluginId); err != nil {
		return nil, grpcError("uninstall plugin", err)
	}

	return &marketv1.UninstallPluginResponse{}, nil
}

// ListInstalledPlugins handles ListInstalledPlugins gRPC request.
func (h *Handler) ListInstalledPlugins(ctx context.Context, req *marketv1.ListInstalledPluginsRequest) (*marketv1.ListInstalledPluginsResponse, error) {
	installations, total, err := h.service.ListInstalledPlugins(contextWithClaimsUserID(ctx), int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, grpcError("list installed plugins", err)
	}

	protoInstallations := make([]*marketv1.InstalledPlugin, len(installations))
	for i, installed := range installations {
		protoInstallations[i] = toProtoInstalledPlugin(installed)
	}

	return &marketv1.ListInstalledPluginsResponse{
		InstalledPlugins: protoInstallations,
		Total:            total,
	}, nil
}

func contextWithClaimsUserID(ctx context.Context) context.Context {
	claims, ok := auth.GetClaims(ctx)
	if !ok || claims.UserID == "" {
		return ctx
	}
	return context.WithValue(ctx, "user_id", claims.UserID)
}

// CreatePublisher handles CreatePublisher gRPC request.
func (h *Handler) CreatePublisher(ctx context.Context, req *marketv1.CreatePublisherRequest) (*marketv1.CreatePublisherResponse, error) {
	pub, err := h.service.CreatePublisher(contextWithClaimsUserID(ctx), &appplugin.CreatePublisherRequest{
		Name:        req.Name,
		Description: req.Description,
		WebsiteURL:  req.WebsiteUrl,
	})
	if err != nil {
		return nil, grpcError("create publisher", err)
	}

	return &marketv1.CreatePublisherResponse{
		Publisher: toProtoPublisher(pub),
	}, nil
}

// GetPublisher handles GetPublisher gRPC request.
func (h *Handler) GetPublisher(ctx context.Context, req *marketv1.GetPublisherRequest) (*marketv1.GetPublisherResponse, error) {
	pub, err := h.service.GetPublisher(ctx, req.Id)
	if err != nil {
		return nil, grpcError("get publisher", err)
	}

	return &marketv1.GetPublisherResponse{
		Publisher: toProtoPublisher(pub),
	}, nil
}

// ListPublishers handles ListPublishers gRPC request.
func (h *Handler) ListPublishers(ctx context.Context, req *marketv1.ListPublishersRequest) (*marketv1.ListPublishersResponse, error) {
	publishers, total, err := h.service.ListPublishers(ctx, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, grpcError("list publishers", err)
	}

	protoPublishers := make([]*marketv1.Publisher, len(publishers))
	for i, pub := range publishers {
		protoPublishers[i] = toProtoPublisher(pub)
	}

	return &marketv1.ListPublishersResponse{
		Publishers: protoPublishers,
		Total:      total,
	}, nil
}

func grpcError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Errorf(codes.Canceled, "%s: %v", operation, err)
	case errors.Is(err, context.DeadlineExceeded):
		return status.Errorf(codes.DeadlineExceeded, "%s: %v", operation, err)
	case errors.Is(err, appplugin.ErrInvalidInput):
		return status.Errorf(codes.InvalidArgument, "%s: %v", operation, err)
	case errors.Is(err, appplugin.ErrPluginNotFound),
		errors.Is(err, appplugin.ErrReleaseNotFound),
		errors.Is(err, appplugin.ErrArtifactNotFound),
		errors.Is(err, appplugin.ErrPublisherNotFound):
		return status.Errorf(codes.NotFound, "%s: %v", operation, err)
	case errors.Is(err, appplugin.ErrVersionExists),
		errors.Is(err, appplugin.ErrArtifactExists),
		strings.Contains(strings.ToLower(err.Error()), "already exists"):
		return status.Errorf(codes.AlreadyExists, "%s: %v", operation, err)
	default:
		return status.Errorf(codes.Internal, "%s: %v", operation, err)
	}
}
