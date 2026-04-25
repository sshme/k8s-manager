package plugin

import (
	"context"
	"fmt"
	"log"

	appplugin "k8s-manager/market/internal/application/plugin"
	domainplugin "k8s-manager/market/internal/domain/plugin"
	"k8s-manager/market/internal/presentation/grpc/auth"
	marketv1 "k8s-manager/proto/gen/v1/market"
)

// Handler implements the gRPC PluginService
type Handler struct {
	marketv1.UnimplementedPluginServiceServer
	service *appplugin.Service
}

// NewHandler creates a new gRPC plugin handler
func NewHandler(service *appplugin.Service) *Handler {
	return &Handler{
		service: service,
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

	p, err := h.service.CreatePlugin(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}

	return &marketv1.CreatePluginResponse{
		Plugin: toProtoPlugin(p),
	}, nil
}

// GetPlugin handles GetPlugin gRPC request
func (h *Handler) GetPlugin(ctx context.Context, req *marketv1.GetPluginRequest) (*marketv1.GetPluginResponse, error) {
	p, err := h.service.GetPlugin(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}

	return &marketv1.GetPluginResponse{
		Plugin: toProtoPlugin(p),
	}, nil
}

// ListPlugins handles ListPlugins gRPC request
func (h *Handler) ListPlugins(ctx context.Context, req *marketv1.ListPluginsRequest) (*marketv1.ListPluginsResponse, error) {
	if claims, ok := auth.GetClaims(ctx); ok {
		log.Printf("userID=%s username=%s email=%s", claims.UserID, claims.Username, claims.Email)
	}

	listReq := &appplugin.ListPluginsRequest{
		Name:        req.Name,
		Category:    req.Category,
		PublisherID: req.PublisherId,
		TrustStatus: fromProtoTrustStatus(req.TrustStatus),
		Status:      fromProtoPluginStatus(req.Status),
		Limit:       int(req.Limit),
		Offset:      int(req.Offset),
	}

	plugins, total, err := h.service.ListPlugins(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
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
		return nil, fmt.Errorf("failed to update plugin: %w", err)
	}

	return &marketv1.UpdatePluginResponse{
		Plugin: toProtoPlugin(p),
	}, nil
}

// UpdatePluginStatus handles UpdatePluginStatus gRPC request
func (h *Handler) UpdatePluginStatus(ctx context.Context, req *marketv1.UpdatePluginStatusRequest) (*marketv1.UpdatePluginStatusResponse, error) {
	err := h.service.UpdatePluginStatus(ctx, req.Id, fromProtoPluginStatus(req.Status), req.Reason)
	if err != nil {
		return nil, fmt.Errorf("failed to update plugin status: %w", err)
	}

	return &marketv1.UpdatePluginStatusResponse{}, nil
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

	rel, err := h.service.CreateRelease(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create release: %w", err)
	}

	return &marketv1.CreateReleaseResponse{
		Release: toProtoRelease(rel),
	}, nil
}

// GetRelease handles GetRelease gRPC request
func (h *Handler) GetRelease(ctx context.Context, req *marketv1.GetReleaseRequest) (*marketv1.GetReleaseResponse, error) {
	rel, err := h.service.GetRelease(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	return &marketv1.GetReleaseResponse{
		Release: toProtoRelease(rel),
	}, nil
}

// ListReleases handles ListReleases gRPC request
func (h *Handler) ListReleases(ctx context.Context, req *marketv1.ListReleasesRequest) (*marketv1.ListReleasesResponse, error) {
	releases, total, err := h.service.ListReleases(ctx, req.PluginId, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
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
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	return &marketv1.GetLatestReleaseResponse{
		Release: toProtoRelease(rel),
	}, nil
}

// Helper functions to convert domain models to proto
func toProtoPlugin(p *domainplugin.Plugin) *marketv1.Plugin {
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

// UploadArtifact handles UploadArtifact gRPC stream request
func (h *Handler) UploadArtifact(stream marketv1.PluginService_UploadArtifactServer) error {
	var metadata *marketv1.ArtifactMetadata
	var chunks []byte

	// Receive stream
	for {
		req, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to receive: %w", err)
		}

		switch data := req.Data.(type) {
		case *marketv1.UploadArtifactRequest_Metadata:
			metadata = data.Metadata
		case *marketv1.UploadArtifactRequest_Chunk:
			chunks = append(chunks, data.Chunk...)
		}
	}

	if metadata == nil {
		return fmt.Errorf("metadata is required")
	}

	// TODO: Save file to storage and create artifact
	// This is a simplified version - in production, you would:
	// 1. Save chunks to temporary file
	// 2. Calculate checksum
	// 3. Move to final storage location
	// 4. Create artifact record

	return stream.SendAndClose(&marketv1.UploadArtifactResponse{
		// Artifact: artifact,
	})
}

// GetArtifact handles GetArtifact gRPC request
func (h *Handler) GetArtifact(ctx context.Context, req *marketv1.GetArtifactRequest) (*marketv1.GetArtifactResponse, error) {
	art, err := h.service.GetArtifact(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	return &marketv1.GetArtifactResponse{
		Artifact: toProtoArtifact(art),
	}, nil
}

// GetArtifactByPlatform handles GetArtifactByPlatform gRPC request
func (h *Handler) GetArtifactByPlatform(ctx context.Context, req *marketv1.GetArtifactByPlatformRequest) (*marketv1.GetArtifactResponse, error) {
	art, err := h.service.GetArtifactByPlatform(ctx, req.ReleaseId, req.Os, req.Arch)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	return &marketv1.GetArtifactResponse{
		Artifact: toProtoArtifact(art),
	}, nil
}

// ListArtifacts handles ListArtifacts gRPC request
func (h *Handler) ListArtifacts(ctx context.Context, req *marketv1.ListArtifactsRequest) (*marketv1.ListArtifactsResponse, error) {
	artifacts, err := h.service.ListArtifacts(ctx, req.ReleaseId)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
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
	// TODO: Implement file streaming
	// This would:
	// 1. Get artifact from service
	// 2. Open file from storage
	// 3. Stream chunks to client
	return fmt.Errorf("not implemented")
}

// DeleteArtifact handles DeleteArtifact gRPC request
func (h *Handler) DeleteArtifact(ctx context.Context, req *marketv1.DeleteArtifactRequest) (*marketv1.DeleteArtifactResponse, error) {
	err := h.service.DeleteArtifact(ctx, req.Id, req.Reason)
	if err != nil {
		return nil, fmt.Errorf("failed to delete artifact: %w", err)
	}

	return &marketv1.DeleteArtifactResponse{}, nil
}
