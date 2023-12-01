package roles

import (
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/extensions/provisioning"
	"github.com/rancher/shepherd/extensions/provisioninginput"
	rbac "github.com/rancher/shepherd/extensions/rbac"
	"github.com/rancher/shepherd/extensions/settings"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RestrictedAdminTestSuite struct {
	suite.Suite
	client  *rancher.Client
	session *session.Session
}

func (ra *RestrictedAdminTestSuite) TestRestrictedAdminCreateCluster() {
	subSession := ra.session.NewSession()
	defer subSession.Cleanup()

	_, restrictedAdminClient, err := rbac.SetupUser(ra.client, rbac.RestrictedAdmin.String())
	require.NoError(ra.T(), err)
	ra.T().Logf("Validating restricted admin can create an RKE1 cluster")
	userConfig := new(provisioninginput.Config)
	config.LoadConfig(provisioninginput.ConfigurationFileKey, userConfig)
	nodeProviders := userConfig.NodeProviders[0]
	nodeAndRoles := []provisioninginput.NodePools{
		provisioninginput.AllRolesNodePool,
	}
	externalNodeProvider := provisioning.ExternalNodeProviderSetup(nodeProviders)
	clusterConfig := clusters.ConvertConfigToClusterConfig(userConfig)
	clusterConfig.NodePools = nodeAndRoles
	kubernetesVersion, err := kubernetesversions.Default(restrictedAdminClient, clusters.RKE1ClusterType.String(), []string{})
	require.NoError(ra.T(), err)

	clusterConfig.KubernetesVersion = kubernetesVersion[0]
	clusterConfig.CNI = userConfig.CNIs[0]
	clusterObject, _, err := provisioning.CreateProvisioningRKE1CustomCluster(restrictedAdminClient, &externalNodeProvider, clusterConfig)
	require.NoError(ra.T(), err)
	provisioning.VerifyRKE1Cluster(ra.T(), restrictedAdminClient, clusterConfig, clusterObject)

}

func (ra *RestrictedAdminTestSuite) TestRestrictedAdminGlobalSettings(t *testing.T) {

	subSession := ra.session.NewSession()
	defer subSession.Cleanup()

	_, restrictedAdminClient, err := rbac.SetupUser(ra.client, rbac.RestrictedAdmin.String())
	require.NoError(ra.T(), err)
	ra.T().Log("Validating restricted Admin can list global settings")
	steveRestrictedAdminclient := restrictedAdminClient.Steve
	steveAdminClient := ra.client.Steve

	adminListSettings, err := steveAdminClient.SteveType(settings.ManagementSetting).List(nil)
	require.NoError(ra.T(), err)
	adminSettings := adminListSettings.Names()

	resAdminListSettings, err := steveRestrictedAdminclient.SteveType(settings.ManagementSetting).List(nil)
	require.NoError(ra.T(), err)
	resAdminSettings := resAdminListSettings.Names()

	assert.Equal(ra.T(), len(adminSettings), len(resAdminSettings))
	assert.Equal(ra.T(), adminListSettings, resAdminListSettings)
}

func (ra *RestrictedAdminTestSuite) TestRestrictedAdminUpdateGlobalSettings(t *testing.T) {
	ra.T().Logf("Validating restrictedAdmin cannot edit global settings")

	_, restrictedAdminClient, err := rbac.SetupUser(ra.client, rbac.RestrictedAdmin.String())
	steveRestrictedAdminclient := restrictedAdminClient.Steve
	steveAdminClient := ra.client.Steve

	kubeConfigTokenSetting, err := steveAdminClient.SteveType(settings.ManagementSetting).ByID(settings.KubeConfigToken)
	require.NoError(ra.T(), err)

	_, err = settings.UpdateGlobalSettings(steveRestrictedAdminclient, kubeConfigTokenSetting, "3")
	require.Error(ra.T(), err)
	assert.Contains(ra.T(), err.Error(), "Resource type [management.cattle.io.setting] is not updatable")
}

func TestRestrictedAdminP1TestSuite(t *testing.T) {
	suite.Run(t, new(RestrictedAdminTestSuite))
}
