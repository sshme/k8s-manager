package plugin

import (
	"context"
)

// PluginRepository defines the interface for plugin persistence
type PluginRepository interface {
	Create(ctx context.Context, plugin *Plugin) (*Plugin, error)
	GetByID(ctx context.Context, id int64) (*Plugin, error)
	GetByIdentifier(ctx context.Context, identifier string) (*Plugin, error)
	List(ctx context.Context, filter *PluginFilter, limit, offset int) ([]*Plugin, int64, error)
	Update(ctx context.Context, plugin *Plugin) (*Plugin, error)
	UpdateStatus(ctx context.Context, id int64, status PluginStatus, reason string) error
	UpdateTrustStatus(ctx context.Context, id int64, trustStatus TrustStatus, reason string) error
}

// PluginFilter represents filtering options for plugins
type PluginFilter struct {
	Name        string
	Query       string
	Category    string
	PublisherID int64
	TrustStatus TrustStatus
	Status      PluginStatus
}

// ReleaseRepository defines the interface for release persistence
type ReleaseRepository interface {
	Create(ctx context.Context, release *Release) (*Release, error)
	GetByID(ctx context.Context, id int64) (*Release, error)
	GetByPluginIDAndVersion(ctx context.Context, pluginID int64, version string) (*Release, error)
	ListByPluginID(ctx context.Context, pluginID int64, limit, offset int) ([]*Release, int64, error)
	SetLatest(ctx context.Context, pluginID int64, releaseID int64) error
	GetLatest(ctx context.Context, pluginID int64) (*Release, error)
}

// ArtifactRepository defines the interface for artifact persistence
type ArtifactRepository interface {
	Create(ctx context.Context, artifact *Artifact) (*Artifact, error)
	GetByID(ctx context.Context, id int64) (*Artifact, error)
	GetByReleaseIDAndPlatform(ctx context.Context, releaseID int64, os, arch string) (*Artifact, error)
	ListByReleaseID(ctx context.Context, releaseID int64) ([]*Artifact, error)
	Delete(ctx context.Context, id int64) error
}

// InstallationRepository defines the interface for user plugin installations.
type InstallationRepository interface {
	Install(ctx context.Context, userID string, pluginID int64) (*PluginInstallation, error)
	Uninstall(ctx context.Context, userID string, pluginID int64) error
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*PluginInstallation, int64, error)
}

// PublisherRepository defines the interface for publisher persistence
type PublisherRepository interface {
	Create(ctx context.Context, publisher *Publisher) (*Publisher, error)
	GetByID(ctx context.Context, id int64) (*Publisher, error)
	GetByName(ctx context.Context, name string) (*Publisher, error)
	List(ctx context.Context, limit, offset int) ([]*Publisher, int64, error)
}
