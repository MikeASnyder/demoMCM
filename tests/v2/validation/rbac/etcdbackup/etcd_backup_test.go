//go:build (validation || infra.any || cluster.any || stress) && !sanity && !extended

package rbac

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

func (rb *ETCDRbacBackupTestSuite) TearDownSuite() {
	rb.session.Cleanup()
}

func (rb *ETCDRbacBackupTestSuite) SetupSuite() {
	rb.session = session.NewSession()

	client, err := rancher.NewClient("", rb.session)
	require.NoError(rb.T(), err)

	rb.client = client
	clusterName := client.RancherConfig.ClusterName
	require.NotEmptyf(rb.T(), clusterName, "Cluster name to install should be set")
	clusterID, err := clusters.GetClusterIDByName(rb.client, clusterName)
	require.NoError(rb.T(), err, "Error getting cluster ID")
	rb.cluster, err = rb.client.Management.Cluster.ByID(clusterID)
	require.NoError(rb.T(), err)
}

func (rb *ETCDRbacBackupTestSuite) testEtcdSnapshotCluster(role, name string, user *management.User) {

	rb.T().Logf("Created user: %v", user.Username)
	standardUserClient, err := rb.client.AsUser(user)
	require.NoError(rb.T(), err)

	adminProject, err := rb.client.Management.Project.Create(projects.NewProjectConfig(rb.cluster.ID))
	require.NoError(rb.T(), err)

	if role == rbac.StandardUser.String() {
		if strings.Contains(role, "project") {
			err := users.AddProjectMember(rb.client, adminProject, user, role, nil)
			require.NoError(rb.T(), err)
		} else {
			err := users.AddClusterRoleToUser(rb.client, rb.cluster, user, role, nil)
			require.NoError(rb.T(), err)
		}
	}

	relogin, err := standardUserClient.ReLogin()
	require.NoError(rb.T(), err)
	standardUserClient = relogin

	log.Info("Test case - Take Etcd snapshot of a cluster as a " + name)

	err = etcdsnapshot.CreateRKE2K3SSnapshot(standardUserClient, rb.cluster.ID)
	switch role {
	case rbac.ClusterOwner.String(), rbac.RestrictedAdmin.String():
		require.NoError(rb.T(), err)

	case rbac.ClusterMember.String(), rbac.ProjectOwner.String(), rbac.ProjectMember.String():
		require.Error(rb.T(), err)
		assert.Equal(rb.T(), "Resource type [provisioning.cattle.io.cluster] is not updatable", err.Error())
	}
}

func (rb *ETCDRbacBackupTestSuite) TestETCDRBAC() {
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
		clusterID, err := clusters.GetClusterIDByName(rb.client, rb.cluster.ID)
		require.NoError(rb.T(), err)
		if !(strings.Contains(clusterID, "c-m-")) {
			rb.T().Skip("Skipping tests since cluster is not of type - k3s or RKE2")
		}
		rb.Run("Set up User with Cluster Role "+tt.name, func() {
			newUser, err := users.CreateUserWithRole(rb.client, users.UserConfig(), tt.member)
			require.NoError(rb.T(), err)

			rb.testEtcdSnapshotCluster(tt.role, tt.name, newUser)
			subSession := rb.session.NewSession()
			defer subSession.Cleanup()
		})
	}
}

func TestETCDRBACBackupTestSuite(t *testing.T) {
	suite.Run(t, new(ETCDRbacBackupTestSuite))
}
