package plugins

import "k8s-manager/cli/internal/market"

// InstalledArtifact описывает один скачанный артефакт, как он живёт
// в локальном реестре. Сериализуется в registry.json и install.json.
type InstalledArtifact struct {
	PluginID         int64  `json:"plugin_id"`
	PluginIdentifier string `json:"plugin_identifier"`
	PluginName       string `json:"plugin_name"`
	PublisherID      int64  `json:"publisher_id"`
	PublisherName    string `json:"publisher_name"`
	ReleaseID        int64  `json:"release_id"`
	Version          string `json:"version"`
	ArtifactID       int64  `json:"artifact_id"`
	OS               string `json:"os"`
	Arch             string `json:"arch"`
	Type             string `json:"type"`
	Size             int64  `json:"size"`
	Checksum         string `json:"checksum"`
	Signature        string `json:"signature,omitempty"`
	KeyID            string `json:"key_id,omitempty"`
	InstallDir       string `json:"install_dir"`
	DownloadedAt     string `json:"downloaded_at"`
}

// InstallRef - запрос на установку артефакта
type InstallRef struct {
	PluginID         int64
	PluginIdentifier string
	PluginName       string
	PublisherID      int64
	PublisherName    string
	ReleaseID        int64
	Version          string
	Artifact         market.Artifact
}
