package secretmigrator

import (
	"context"

	v1 "github.com/rancher/rancher/pkg/generated/norman/core/v1"
	v3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
	"github.com/rancher/rancher/pkg/types/config"
)

type Migrator struct {
	secretLister v1.SecretLister
	secrets      v1.SecretInterface
}

type handler struct {
	migrator *Migrator
	clusters v3.ClusterInterface
}

func NewMigrator(secretLister v1.SecretLister, secrets v1.SecretInterface) *Migrator {
	return &Migrator{
		secretLister: secretLister,
		secrets:      secrets,
	}
}

func Register(ctx context.Context, management *config.ManagementContext) {
	h := handler{
		migrator: NewMigrator(
			management.Core.Secrets("").Controller().Lister(),
			management.Core.Secrets(""),
		),
		clusters: management.Management.Clusters(""),
	}
	management.Management.Clusters("").AddHandler(ctx, "cluster-secret-migrator", h.sync)
}
