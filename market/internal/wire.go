//go:build wireinject
// +build wireinject

package internal

import (
	"database/sql"

	"github.com/google/wire"

	appaudit "k8s-manager/market/internal/application/audit"
	appplugin "k8s-manager/market/internal/application/plugin"
	appuser "k8s-manager/market/internal/application/user"
	domainaudit "k8s-manager/market/internal/domain/audit"
	domainplugin "k8s-manager/market/internal/domain/plugin"
	domainuser "k8s-manager/market/internal/domain/user"
	artifactrepo "k8s-manager/market/internal/infrastructure/repository/artifact"
	auditrepo "k8s-manager/market/internal/infrastructure/repository/audit"
	installationrepo "k8s-manager/market/internal/infrastructure/repository/installation"
	pluginrepo "k8s-manager/market/internal/infrastructure/repository/plugin"
	publisherrepo "k8s-manager/market/internal/infrastructure/repository/publisher"
	releaserepo "k8s-manager/market/internal/infrastructure/repository/release"
	userrepo "k8s-manager/market/internal/infrastructure/repository/user"
	"k8s-manager/market/internal/infrastructure/storage"
	grpcserver "k8s-manager/market/internal/presentation/grpc"
	grpcplugin "k8s-manager/market/internal/presentation/grpc/plugin"
	grpcuser "k8s-manager/market/internal/presentation/grpc/user"
)

// InitializeApp initializes the application with all dependencies
func InitializeApp(db *sql.DB, grpcPort int, metricsPort MetricsPort, storagePath string) (*App, error) {
	wire.Build(
		// Repositories
		pluginrepo.NewPostgresPluginRepository,
		wire.Bind(new(domainplugin.PluginRepository), new(*pluginrepo.PostgresPluginRepository)),

		releaserepo.NewPostgresReleaseRepository,
		wire.Bind(new(domainplugin.ReleaseRepository), new(*releaserepo.PostgresReleaseRepository)),

		artifactrepo.NewPostgresArtifactRepository,
		wire.Bind(new(domainplugin.ArtifactRepository), new(*artifactrepo.PostgresArtifactRepository)),

		installationrepo.NewPostgresInstallationRepository,
		wire.Bind(new(domainplugin.InstallationRepository), new(*installationrepo.PostgresInstallationRepository)),

		publisherrepo.NewPostgresPublisherRepository,
		wire.Bind(new(domainplugin.PublisherRepository), new(*publisherrepo.PostgresPublisherRepository)),

		storage.NewArtifactStorage,

		auditrepo.NewPostgresAuditRepository,
		wire.Bind(new(domainaudit.AuditRepository), new(*auditrepo.PostgresAuditRepository)),

		userrepo.NewPostgresRepository,
		wire.Bind(new(domainuser.Repository), new(*userrepo.PostgresRepository)),

		// Services
		appaudit.NewService,
		wire.Bind(new(appaudit.AuditLogger), new(*appaudit.Service)),
		appplugin.NewService,
		appuser.NewService,

		// Handlers
		grpcplugin.NewHandler,
		grpcuser.NewHandler,

		// Server
		grpcserver.NewServer,

		// App
		NewApp,
	)

	return nil, nil
}
