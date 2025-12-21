package plugin

import (
	"time"
)

// PluginStatus represents the status of a plugin
type PluginStatus string

const (
	PluginStatusActive   PluginStatus = "active"
	PluginStatusHidden   PluginStatus = "hidden"
	PluginStatusBlocked  PluginStatus = "blocked"
)

// TrustStatus represents the trust status of a plugin
type TrustStatus string

const (
	TrustStatusOfficial    TrustStatus = "official"
	TrustStatusVerified    TrustStatus = "verified"
	TrustStatusCommunity   TrustStatus = "community"
)

// Plugin represents a plugin entity
type Plugin struct {
	ID          int64
	Identifier  string
	Name        string
	Description string
	Category    string
	PublisherID int64
	Status      PluginStatus
	TrustStatus TrustStatus
	SourceURL   string
	DocsURL     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Release represents a plugin release/version
type Release struct {
	ID                  int64
	PluginID            int64
	Version             string
	PublishedAt         time.Time
	Changelog           string
	MinCLIVersion       string
	MinK8sVersion       string
	IsLatest            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Artifact represents a plugin artifact (executable file)
type Artifact struct {
	ID          int64
	ReleaseID   int64
	OS          string // e.g., "linux", "darwin", "windows"
	Arch        string // e.g., "amd64", "arm64"
	Type        string // e.g., "binary"
	Size        int64
	Checksum    string // SHA-256
	StoragePath string
	Signature   string // optional signature
	KeyID       string // optional signing key ID
	CreatedAt   time.Time
}

// Publisher represents a plugin publisher/provider
type Publisher struct {
	ID          int64
	Name        string
	Description string
	WebsiteURL  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

