package plugin

import (
	"context"
	"errors"
	"fmt"
	"time"

	appaudit "k8s-manager/market/internal/application/audit"
	"k8s-manager/market/internal/domain/plugin"
)

var (
	ErrPluginNotFound      = errors.New("plugin not found")
	ErrReleaseNotFound     = errors.New("release not found")
	ErrArtifactNotFound    = errors.New("artifact not found")
	ErrPublisherNotFound   = errors.New("publisher not found")
	ErrInvalidInput        = errors.New("invalid input")
	ErrVersionExists       = errors.New("version already exists")
	ErrArtifactExists      = errors.New("artifact for this platform already exists")
)

// Service handles plugin business logic
type Service struct {
	pluginRepo   plugin.PluginRepository
	releaseRepo  plugin.ReleaseRepository
	artifactRepo plugin.ArtifactRepository
	publisherRepo plugin.PublisherRepository
	auditService appaudit.AuditLogger
}

// NewService creates a new plugin service
func NewService(
	pluginRepo plugin.PluginRepository,
	releaseRepo plugin.ReleaseRepository,
	artifactRepo plugin.ArtifactRepository,
	publisherRepo plugin.PublisherRepository,
	auditService appaudit.AuditLogger,
) *Service {
	return &Service{
		pluginRepo:   pluginRepo,
		releaseRepo:  releaseRepo,
		artifactRepo: artifactRepo,
		publisherRepo: publisherRepo,
		auditService: auditService,
	}
}

// getUserID extracts user ID from context
func (s *Service) getUserID(ctx context.Context) string {
	// This will be set by middleware
	userID, _ := ctx.Value("user_id").(string)
	return userID
}

// CreatePluginRequest represents a request to create a plugin
type CreatePluginRequest struct {
	Identifier  string
	Name        string
	Description string
	Category    string
	PublisherID int64
	TrustStatus plugin.TrustStatus
	SourceURL   string
	DocsURL     string
}

// CreatePlugin creates a new plugin
func (s *Service) CreatePlugin(ctx context.Context, req *CreatePluginRequest) (*plugin.Plugin, error) {
	if req.Identifier == "" {
		return nil, fmt.Errorf("%w: identifier is required", ErrInvalidInput)
	}
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	
	// Check if publisher exists
	pub, err := s.publisherRepo.GetByID(ctx, req.PublisherID)
	if err != nil {
		return nil, fmt.Errorf("failed to get publisher: %w", err)
	}
	if pub == nil {
		return nil, ErrPublisherNotFound
	}
	
	// Check if identifier already exists
	existing, err := s.pluginRepo.GetByIdentifier(ctx, req.Identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing plugin: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("plugin with identifier %s already exists", req.Identifier)
	}
	
	p := &plugin.Plugin{
		Identifier:  req.Identifier,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		PublisherID: req.PublisherID,
		Status:      plugin.PluginStatusActive,
		TrustStatus: req.TrustStatus,
		SourceURL:   req.SourceURL,
		DocsURL:     req.DocsURL,
	}
	
	if p.TrustStatus == "" {
		p.TrustStatus = plugin.TrustStatusCommunity
	}
	
	created, err := s.pluginRepo.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	
	// Log audit
	userID := s.getUserID(ctx)
	if userID != "" {
		s.auditService.LogCreate(ctx, "plugin", created.ID, userID, created)
	}
	
	return created, nil
}

// GetPlugin retrieves a plugin by ID
func (s *Service) GetPlugin(ctx context.Context, id int64) (*plugin.Plugin, error) {
	p, err := s.pluginRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}
	if p == nil {
		return nil, ErrPluginNotFound
	}
	return p, nil
}

// ListPluginsRequest represents a request to list plugins
type ListPluginsRequest struct {
	Name        string
	Category    string
	PublisherID int64
	TrustStatus plugin.TrustStatus
	Status      plugin.PluginStatus
	Limit       int
	Offset      int
}

// ListPlugins retrieves a list of plugins
func (s *Service) ListPlugins(ctx context.Context, req *ListPluginsRequest) ([]*plugin.Plugin, int64, error) {
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	
	filter := &plugin.PluginFilter{
		Name:        req.Name,
		Category:    req.Category,
		PublisherID: req.PublisherID,
		TrustStatus: req.TrustStatus,
		Status:      req.Status,
	}
	
	return s.pluginRepo.List(ctx, filter, req.Limit, req.Offset)
}

// UpdatePluginRequest represents a request to update a plugin
type UpdatePluginRequest struct {
	ID          int64
	Name        string
	Description string
	Category    string
	SourceURL   string
	DocsURL     string
}

// UpdatePlugin updates a plugin
func (s *Service) UpdatePlugin(ctx context.Context, req *UpdatePluginRequest) (*plugin.Plugin, error) {
	p, err := s.pluginRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}
	if p == nil {
		return nil, ErrPluginNotFound
	}
	
	oldValue := *p
	
	p.Name = req.Name
	p.Description = req.Description
	p.Category = req.Category
	p.SourceURL = req.SourceURL
	p.DocsURL = req.DocsURL
	
	updated, err := s.pluginRepo.Update(ctx, p)
	if err != nil {
		return nil, err
	}
	
	// Log audit
	userID := s.getUserID(ctx)
	if userID != "" {
		s.auditService.LogUpdate(ctx, "plugin", updated.ID, userID, oldValue, updated)
	}
	
	return updated, nil
}

// UpdatePluginStatus updates plugin status
func (s *Service) UpdatePluginStatus(ctx context.Context, id int64, status plugin.PluginStatus, reason string) error {
	p, err := s.pluginRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get plugin: %w", err)
	}
	if p == nil {
		return ErrPluginNotFound
	}
	
	oldStatus := string(p.Status)
	
	err = s.pluginRepo.UpdateStatus(ctx, id, status, reason)
	if err != nil {
		return err
	}
	
	// Log audit
	userID := s.getUserID(ctx)
	if userID != "" {
		s.auditService.LogStatusChange(ctx, "plugin", id, userID, reason, oldStatus, string(status))
	}
	
	return nil
}

// CreateReleaseRequest represents a request to create a release
type CreateReleaseRequest struct {
	PluginID      int64
	Version       string
	Changelog     string
	MinCLIVersion string
	MinK8sVersion string
	IsLatest      bool
}

// CreateRelease creates a new release
func (s *Service) CreateRelease(ctx context.Context, req *CreateReleaseRequest) (*plugin.Release, error) {
	if req.Version == "" {
		return nil, fmt.Errorf("%w: version is required", ErrInvalidInput)
	}
	
	// Check if plugin exists
	p, err := s.pluginRepo.GetByID(ctx, req.PluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}
	if p == nil {
		return nil, ErrPluginNotFound
	}
	
	// Check if version already exists
	existing, err := s.releaseRepo.GetByPluginIDAndVersion(ctx, req.PluginID, req.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing release: %w", err)
	}
	if existing != nil {
		return nil, ErrVersionExists
	}
	
	// If this is the first release, automatically set it as latest
	if req.IsLatest {
		// Check if there are any existing releases
		existing, _, err := s.releaseRepo.ListByPluginID(ctx, req.PluginID, 1, 0)
		if err == nil && len(existing) == 0 {
			req.IsLatest = true
		}
	}
	
	rel := &plugin.Release{
		PluginID:      req.PluginID,
		Version:       req.Version,
		PublishedAt:   time.Now(),
		Changelog:     req.Changelog,
		MinCLIVersion: req.MinCLIVersion,
		MinK8sVersion: req.MinK8sVersion,
		IsLatest:      req.IsLatest,
	}
	
	created, err := s.releaseRepo.Create(ctx, rel)
	if err != nil {
		return nil, err
	}
	
	// Log audit
	userID := s.getUserID(ctx)
	if userID != "" {
		s.auditService.LogCreate(ctx, "release", created.ID, userID, created)
	}
	
	return created, nil
}

// GetRelease retrieves a release by ID
func (s *Service) GetRelease(ctx context.Context, id int64) (*plugin.Release, error) {
	rel, err := s.releaseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}
	if rel == nil {
		return nil, ErrReleaseNotFound
	}
	return rel, nil
}

// ListReleases retrieves releases for a plugin
func (s *Service) ListReleases(ctx context.Context, pluginID int64, limit, offset int) ([]*plugin.Release, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	
	return s.releaseRepo.ListByPluginID(ctx, pluginID, limit, offset)
}

// GetLatestRelease retrieves the latest release for a plugin
func (s *Service) GetLatestRelease(ctx context.Context, pluginID int64) (*plugin.Release, error) {
	rel, err := s.releaseRepo.GetLatest(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}
	if rel == nil {
		return nil, ErrReleaseNotFound
	}
	return rel, nil
}

// CreateArtifactRequest represents a request to create an artifact
type CreateArtifactRequest struct {
	ReleaseID   int64
	OS          string
	Arch        string
	Type        string
	Size        int64
	Checksum    string
	StoragePath string
	Signature   string
	KeyID       string
}

// CreateArtifact creates a new artifact
func (s *Service) CreateArtifact(ctx context.Context, req *CreateArtifactRequest) (*plugin.Artifact, error) {
	if req.OS == "" || req.Arch == "" {
		return nil, fmt.Errorf("%w: OS and Arch are required", ErrInvalidInput)
	}
	
	// Check if release exists
	rel, err := s.releaseRepo.GetByID(ctx, req.ReleaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}
	if rel == nil {
		return nil, ErrReleaseNotFound
	}
	
	// Check if artifact already exists for this platform
	existing, err := s.artifactRepo.GetByReleaseIDAndPlatform(ctx, req.ReleaseID, req.OS, req.Arch)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing artifact: %w", err)
	}
	if existing != nil {
		return nil, ErrArtifactExists
	}
	
	art := &plugin.Artifact{
		ReleaseID:   req.ReleaseID,
		OS:          req.OS,
		Arch:        req.Arch,
		Type:        req.Type,
		Size:        req.Size,
		Checksum:    req.Checksum,
		StoragePath: req.StoragePath,
		Signature:   req.Signature,
		KeyID:       req.KeyID,
	}
	
	if art.Type == "" {
		art.Type = "binary"
	}
	
	created, err := s.artifactRepo.Create(ctx, art)
	if err != nil {
		return nil, err
	}
	
	// Log audit
	userID := s.getUserID(ctx)
	if userID != "" {
		s.auditService.LogCreate(ctx, "artifact", created.ID, userID, created)
	}
	
	return created, nil
}

// GetArtifact retrieves an artifact by ID
func (s *Service) GetArtifact(ctx context.Context, id int64) (*plugin.Artifact, error) {
	art, err := s.artifactRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}
	if art == nil {
		return nil, ErrArtifactNotFound
	}
	return art, nil
}

// ListArtifacts retrieves artifacts for a release
func (s *Service) ListArtifacts(ctx context.Context, releaseID int64) ([]*plugin.Artifact, error) {
	return s.artifactRepo.ListByReleaseID(ctx, releaseID)
}

// DeleteArtifact deletes an artifact
func (s *Service) DeleteArtifact(ctx context.Context, id int64, reason string) error {
	art, err := s.artifactRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get artifact: %w", err)
	}
	if art == nil {
		return ErrArtifactNotFound
	}
	
	err = s.artifactRepo.Delete(ctx, id)
	if err != nil {
		return err
	}
	
	// Log audit
	userID := s.getUserID(ctx)
	if userID != "" {
		s.auditService.LogDelete(ctx, "artifact", id, userID, reason, art)
	}
	
	return nil
}

// GetArtifactByPlatform retrieves an artifact by release ID and platform
func (s *Service) GetArtifactByPlatform(ctx context.Context, releaseID int64, os, arch string) (*plugin.Artifact, error) {
	art, err := s.artifactRepo.GetByReleaseIDAndPlatform(ctx, releaseID, os, arch)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}
	if art == nil {
		return nil, ErrArtifactNotFound
	}
	return art, nil
}

