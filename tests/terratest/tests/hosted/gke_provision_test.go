package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	"github.com/rancher/rancher/tests/framework/pkg/config"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/rancher/rancher/tests/terratest/functions"
	"github.com/rancher/rancher/tests/terratest/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGKEProvisionAndScale(t *testing.T) {
	t.Parallel()

	module := "gke"
	active := "active"

	clusterConfig := new(tests.TerratestConfig)
	config.LoadConfig("terratest", clusterConfig)

	expectedKubernetesVersion := `v` + clusterConfig.KubernetesVersion

	// Set terraform.tfvars file
	functions.SetVarsTF(module)

	// Set initial infrastructure by building TFs declarative config file - [main.tf]
	successful, err := functions.SetConfigTF(module, clusterConfig.KubernetesVersion, clusterConfig.Nodepools)
	require.NoError(t, err)
	assert.Equal(t, true, successful)

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{

		TerraformDir: "../../modules/hosted/" + module,
		NoColor:      true,
	})

	cleanup := func() {
		terraform.Destroy(t, terraformOptions)
		functions.CleanupConfigTF(module)
		functions.CleanupVarsTF(module)
	}

	// Deploys [main.tf] infrastructure and sets up resource cleanup
	defer cleanup()
	terraform.InitAndApply(t, terraformOptions)

	// Grab cluster name from TF outputs
	clusterName := terraform.Output(t, terraformOptions, "cluster_name")

	// Create session, client, and grab cluster specs
	testSession := session.NewSession(t)

	client, err := rancher.NewClient("", testSession)
	require.NoError(t, err)

	clusterID, err := clusters.GetClusterIDByName(client, clusterName)
	require.NoError(t, err)

	cluster, err := client.Management.Cluster.ByID(clusterID)
	require.NoError(t, err)

	// Test cluster
	assert.Equal(t, clusterName, cluster.Name)
	assert.Equal(t, clusterConfig.NodeCount, cluster.NodeCount)
	assert.Equal(t, module, cluster.Provider)
	assert.Equal(t, active, cluster.State)
	assert.Equal(t, expectedKubernetesVersion, cluster.Version.GitVersion)

}
