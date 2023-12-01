//go:build (validation || infra.any || cluster.any || stress) && !sanity && !extended

package etcd

import (
	"strings"
	"testing"

	rbac "github.com/rancher/shepherd/extensions/rbac"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/etcdsnapshot"
	"github.com/rancher/shepherd/extensions/projects"
	"github.com/rancher/shepherd/extensions/users"
	"github.com/rancher/shepherd/pkg/session"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ETCDRbacBackupTestSuite struct {
	suite.Suite
	client  *rancher.Client
	session *session.Session
	cluster *management.Cluster
}

func (etcd *ETCDRbacBackupTestSuite) TearDownSuite() {
	etcd.session.Cleanup()
}

func (etcd *ETCDRbacBackupTestSuite) SetupSuite() {
	etcd.session = session.NewSession()

	client, err := rancher.NewClient("", etcd.session)
	require.NoError(etcd.T(), err)

	etcd.client = client
	clusterName := client.RancherConfig.ClusterName
	require.NotEmptyf(etcd.T(), clusterName, "Cluster name to install should be set")
	clusterID, err := clusters.GetClusterIDByName(etcd.client, clusterName)
	require.NoError(etcd.T(), err, "Error getting cluster ID")
	etcd.cluster, err = etcd.client.Management.Cluster.ByID(clusterID)
	require.NoError(etcd.T(), err)
}

func (etcd *ETCDRbacBackupTestSuite) testEtcdSnapshotCluster(role, name string, user *management.User) {

	etcd.T().Logf("Created user: %v", user.Username)
	standardUserClient, err := etcd.client.AsUser(user)
	require.NoError(etcd.T(), err)

	adminProject, err := etcd.client.Management.Project.Create(projects.NewProjectConfig(etcd.cluster.ID))
	require.NoError(etcd.T(), err)

	if role == rbac.StandardUser.String() {
		if strings.Contains(role, "project") {
			err := users.AddProjectMember(etcd.client, adminProject, user, role, nil)
			require.NoError(etcd.T(), err)
		} else {
			err := users.AddClusterRoleToUser(etcd.client, etcd.cluster, user, role, nil)
			require.NoError(etcd.T(), err)
		}
	}

	relogin, err := standardUserClient.ReLogin()
	require.NoError(etcd.T(), err)
	standardUserClient = relogin

	log.Info("Test case - Take Etcd snapshot of a cluster as a " + name)

	err = etcdsnapshot.CreateRKE2K3SSnapshot(standardUserClient, etcd.cluster.ID)
	switch role {
	case rbac.ClusterOwner.String(), rbac.RestrictedAdmin.String():
		require.NoError(etcd.T(), err)

	case rbac.ClusterMember.String(), rbac.ProjectOwner.String(), rbac.ProjectMember.String():
		require.Error(etcd.T(), err)
		assert.Equal(etcd.T(), "Resource type [provisioning.cattle.io.cluster] is not updatable", err.Error())
	}
}

func (etcd *ETCDRbacBackupTestSuite) TestETCDRBAC() {
	tests := []struct {
		name   string
		role   string
		member string
	}{
		{"Cluster Owner", rbac.ClusterOwner.String(), rbac.StandardUser.String()},
		{"Cluster Member", rbac.ClusterMember.String(), rbac.StandardUser.String()},
		{"Project Owner", rbac.ProjectOwner.String(), rbac.StandardUser.String()},
		{"Project Member", rbac.ProjectMember.String(), rbac.StandardUser.String()},
		{"Restricted Admin", rbac.RestrictedAdmin.String(), rbac.RestrictedAdmin.String()},
	}
	for _, tt := range tests {
		clusterID, err := clusters.GetClusterIDByName(etcd.client, etcd.cluster.ID)
		require.NoError(etcd.T(), err)
		if !(strings.Contains(clusterID, "c-m-")) {
			etcd.T().Skip("Skipping tests since cluster is not of type - k3s or RKE2")
		}
		etcd.Run("Set up User with Cluster Role "+tt.name, func() {
			newUser, err := users.CreateUserWithRole(etcd.client, users.UserConfig(), tt.member)
			require.NoError(etcd.T(), err)

			etcd.testEtcdSnapshotCluster(tt.role, tt.name, newUser)
			subSession := etcd.session.NewSession()
			defer subSession.Cleanup()
		})
	}
}

func TestETCDRBACBackupTestSuite(t *testing.T) {
	suite.Run(t, new(ETCDRbacBackupTestSuite))
}
