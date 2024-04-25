//go:build (validation || infra.any || cluster.any || extended) && !sanity && !stress

package schemas

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/defaults"
	"github.com/rancher/shepherd/extensions/kubeapi/customresourcedefinitions"
	"github.com/rancher/shepherd/extensions/kubectl"
	"github.com/rancher/shepherd/pkg/session"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	kwait "k8s.io/apimachinery/pkg/util/wait"
)

type SchemaChangesTestSuite struct {
	suite.Suite
	client  *rancher.Client
	session *session.Session
	cluster *management.Cluster
}

func (sc *SchemaChangesTestSuite) TearDownSuite() {
	sc.session.Cleanup()
}

func (sc *SchemaChangesTestSuite) SetupSuite() {
	sc.session = session.NewSession()

	client, err := rancher.NewClient("", sc.session)
	require.NoError(sc.T(), err)

	sc.client = client

	clusterName := client.RancherConfig.ClusterName
	require.NotEmptyf(sc.T(), clusterName, "Cluster name to install should be set")
	clusterID, err := clusters.GetClusterIDByName(sc.client, clusterName)
	require.NoError(sc.T(), err, "Error getting cluster ID")
	sc.cluster, err = sc.client.Management.Cluster.ByID(clusterID)
	require.NoError(sc.T(), err)
}

func (sc *SchemaChangesTestSuite) TestSchemaEndpointLocalCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Access the schemas endpoint on the local cluster and verify that it can be accessed successfully without any errors.")
	schemasCollection, err := sc.client.Steve.SteveType(schemaSteveType).List(nil)
	require.NoError(sc.T(), err)
	log.Infof("Number of schemas: %d", len(schemasCollection.Data))

	log.Info("Verify that each listed schema's resourceFields are set to null and ensure the schemas endpoint exclusively lists top-level types without including child schemas for resources.")
	for _, schema := range schemasCollection.Data {
		schemaID := schema.JSONResp["id"].(string)
		if !exceptionMap[schemaID] {
			require.Nil(sc.T(), schema.JSONResp["resourceFields"], "ResourceFields should be null for schema "+schemaID)

			if match, _ := regexp.MatchString(`\.[vV][0-9]+`, schemaID); match {
				require.Fail(sc.T(), "Child schema should not be listed")
			}
		}
	}
}

func (sc *SchemaChangesTestSuite) TestSchemaEndpointDownstreamCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	steveAdminClient, err := sc.client.Steve.ProxyDownstream(sc.cluster.ID)
	require.NoError(sc.T(), err)

	log.Info("Access the schemas endpoint on the downstream cluster and verify that it can be accessed successfully without any errors.")
	schemasCollection, err := steveAdminClient.SteveType(schemaSteveType).List(nil)
	require.NoError(sc.T(), err)
	log.Infof("Number of schemas: %d", len(schemasCollection.Data))

	log.Info("Verify that each listed schema's resourceFields are set to null and ensure the schemas endpoint exclusively lists top-level types without including child schemas for resources.")
	for _, schema := range schemasCollection.Data {
		schemaID := schema.JSONResp["id"].(string)
		if !exceptionMap[schemaID] {
			require.Nil(sc.T(), schema.JSONResp["resourceFields"], "ResourceFields should be null for schema "+schemaID)

			if match, _ := regexp.MatchString(`\.[vV][0-9]+`, schemaID); match {
				require.Fail(sc.T(), "Child schema should not be listed")
			}
		}
	}
}

func (sc *SchemaChangesTestSuite) TestExistingSchemaLocalCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Access an existing schema on the local cluster and verify that it can be accessed successfully without any errors.")
	schemaResponse, err := getSchemaByID(sc.client, localCluster, projectSchemaID)
	require.NoError(sc.T(), err)

	log.Info("Verify that the schema's resourceFields field is set to null.")
	id := schemaResponse["id"].(string)
	require.Nil(sc.T(), schemaResponse["resourceFields"], "ResourceFields should be null for schema "+id)

	log.Info("Verify that the schemas endpoint for Project does not list the child schemas and trying to access any of the child schemas results in a '404 Not Found' error.")
	childSchemas := map[string]bool{
		"management.cattle.io.v3.project.spec":                                     true,
		"management.cattle.io.v3.project.spec.resourceQuota":                       true,
		"management.cattle.io.v3.project.spec.resourceQuota.usedLimit":             true,
		"management.cattle.io.v3.project.spec.resourceQuota.limit":                 true,
		"management.cattle.io.v3.project.spec.namespaceDefaultResourceQuota":       true,
		"management.cattle.io.v3.project.spec.namespaceDefaultResourceQuota.limit": true,
		"management.cattle.io.v3.project.spec.containerDefaultResourceLimit":       true,
		"management.cattle.io.v3.project.status":                                   true,
		"management.cattle.io.v3.project.status.conditions":                        true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ManagedFieldsEntry":                  true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta":                          true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference":                      true,
	}

	for childSchemaID := range childSchemas {
		_, found := schemaResponse[childSchemaID]
		require.False(sc.T(), found, "Child schema should not be listed")

		_, err = getSchemaByID(sc.client, localCluster, childSchemaID)
		require.Error(sc.T(), err)
		errStatus := strings.Split(err.Error(), ": ")[1]
		require.Equal(sc.T(), "404", errStatus)
	}

	log.Info("Verify that the data returned from the schemas endpoint has the resourceMethods and collectionMethods fields.")
	require.NotNil(sc.T(), schemaResponse["resourceMethods"], "resourceMethods should be present in this schema")
	require.NotNil(sc.T(), schemaResponse["collectionMethods"], "collectionMethods should be present in this schema")
}

func (sc *SchemaChangesTestSuite) TestExistingSchemaDownstreamCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Access an existing schema on the downstream cluster and verify that it can be accessed successfully without any errors.")
	schemaResponse, err := getSchemaByID(sc.client, sc.cluster.ID, namespaceSchemaID)
	require.NoError(sc.T(), err)

	log.Info("Verify that the schema's resourceFields field is set to null.")
	id := schemaResponse["id"].(string)
	require.Nil(sc.T(), schemaResponse["resourceFields"], "ResourceFields should be null for schema "+id)

	log.Info("Verify that the schemas endpoint for Project does not list the child schemas and trying to access any of the child schemas results in a '404 Not Found' error.")
	childSchemas := map[string]bool{
		"io.k8s.api.core.v1.NamespaceCondition":                   true,
		"io.k8s.api.core.v1.NamespaceSpec":                        true,
		"io.k8s.api.core.v1.NamespaceStatus":                      true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ManagedFieldsEntry": true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta":         true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference":     true,
	}

	for childSchemaID := range childSchemas {
		_, found := schemaResponse[childSchemaID]
		require.False(sc.T(), found, "Child schema should not be listed")

		_, err = getSchemaByID(sc.client, sc.cluster.ID, childSchemaID)
		require.Error(sc.T(), err)
		errStatus := strings.Split(err.Error(), ": ")[1]
		require.Equal(sc.T(), "404", errStatus)
	}

	log.Info("Verify that the data returned from the schemas endpoint has the resourceMethods and collectionMethods fields.")
	require.NotNil(sc.T(), schemaResponse["resourceMethods"], "resourceMethods should be present in this schema")
	require.NotNil(sc.T(), schemaResponse["collectionMethods"], "collectionMethods should be present in this schema")
}

func (sc *SchemaChangesTestSuite) TestSchemaDefinitionsEndpointLocalCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Access schemadefinitions endpoint on the local cluster without providing the schema and verify that it fails with an error.")
	_, err := sc.client.Steve.SteveType(schemaDefinitionSteveType).List(nil)
	require.Error(sc.T(), err)
	errMessage := strings.Split(err.Error(), ":")[0]
	require.Equal(sc.T(), "Resource type [schemaDefinition] has no method GET", errMessage)
}

func (sc *SchemaChangesTestSuite) TestSchemaDefinitionsEndpointDownstreamCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	steveAdminClient, err := sc.client.Steve.ProxyDownstream(sc.cluster.ID)
	require.NoError(sc.T(), err)

	log.Info("Access schemadefinitions endpoint on the downstream cluster without providing the schema and verify that it fails with an error.")
	_, err = steveAdminClient.SteveType(schemaDefinitionSteveType).List(nil)
	require.Error(sc.T(), err)
	errMessage := strings.Split(err.Error(), ":")[0]
	require.Equal(sc.T(), "Resource type [schemaDefinition] has no method GET", errMessage)
}

func (sc *SchemaChangesTestSuite) TestExistingSchemaDefinitionLocalCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Access the schema deinition for an existing schema on the local cluster and verify that it can be accessed successfully without any errors.")
	schemaResponse, err := getSchemaDefinitionByID(sc.client, localCluster, projectSchemaID)
	require.NoError(sc.T(), err)

	log.Info("Verify data returned has definitions with resourceFields for each definition type.")
	expectedDefinitions := map[string]bool{
		"io.cattle.management.v3.Project":                                          true,
		"io.cattle.management.v3.Project.spec":                                     true,
		"io.cattle.management.v3.Project.spec.resourceQuota":                       true,
		"io.cattle.management.v3.Project.spec.resourceQuota.usedLimit":             true,
		"io.cattle.management.v3.Project.spec.resourceQuota.limit":                 true,
		"io.cattle.management.v3.Project.spec.namespaceDefaultResourceQuota":       true,
		"io.cattle.management.v3.Project.spec.namespaceDefaultResourceQuota.limit": true,
		"io.cattle.management.v3.Project.spec.containerDefaultResourceLimit":       true,
		"io.cattle.management.v3.Project.status":                                   true,
		"io.cattle.management.v3.Project.status.conditions":                        true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ManagedFieldsEntry":                  true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta":                          true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference":                      true,
	}

	for definitionID := range expectedDefinitions {
		definitionData, exists := schemaResponse["definitions"].(map[string]interface{})[definitionID]
		require.True(sc.T(), exists, "Expected definition "+definitionID+" not found in schemaResponse")
		resourceFields, resourceFieldsExist := definitionData.(map[string]interface{})["resourceFields"]
		require.True(sc.T(), resourceFieldsExist, "ResourceFields are nil for definition "+definitionID)
		require.NotNil(sc.T(), resourceFields, "ResourceFields are nil for definition "+definitionID)
		_, resourceMethodsExist := definitionData.(map[string]interface{})["resourceMethods"]
		require.False(sc.T(), resourceMethodsExist, "ResourceMethods field exists for definition "+definitionID)

		_, collectionMethodsExist := definitionData.(map[string]interface{})["collectionMethods"]
		require.False(sc.T(), collectionMethodsExist, "CollectionMethods field exists for definition "+definitionID)
	}
}

func (sc *SchemaChangesTestSuite) TestExistingSchemaDefinitionDownstreamCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Access the schema deinition for an existing schema on the downstream cluster and verify that it can be accessed successfully without any errors.")
	schemaResponse, err := getSchemaDefinitionByID(sc.client, sc.cluster.ID, namespaceSchemaID)
	require.NoError(sc.T(), err)

	log.Info("Verify data returned has definitions with resourceFields for each definition type.")

	expectedDefinitions := map[string]bool{
		"io.k8s.api.core.v1.NamespaceCondition":                   true,
		"io.k8s.api.core.v1.NamespaceSpec":                        true,
		"io.k8s.api.core.v1.NamespaceStatus":                      true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ManagedFieldsEntry": true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta":         true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference":     true,
	}

	for definitionID := range expectedDefinitions {
		definitionData, exists := schemaResponse["definitions"].(map[string]interface{})[definitionID]
		require.True(sc.T(), exists, "Expected definition "+definitionID+" not found in schemaResponse")
		resourceFields, resourceFieldsExist := definitionData.(map[string]interface{})["resourceFields"]
		require.True(sc.T(), resourceFieldsExist, "ResourceFields are nil for definition "+definitionID)
		require.NotNil(sc.T(), resourceFields, "ResourceFields are nil for definition "+definitionID)

		_, resourceMethodsExist := definitionData.(map[string]interface{})["resourceMethods"]
		require.False(sc.T(), resourceMethodsExist, "ResourceMethods field exists for definition "+definitionID)

		_, collectionMethodsExist := definitionData.(map[string]interface{})["collectionMethods"]
		require.False(sc.T(), collectionMethodsExist, "CollectionMethods field exists for definition "+definitionID)
	}
}

func (sc *SchemaChangesTestSuite) TestSchemaDefinitionsExistenceForSchemaLocalCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Access the schemas and schema definition endpoint on the local cluster and verify that it can be accessed successfully without any errors.")
	schemasCollection, err := sc.client.Steve.SteveType(schemaSteveType).List(nil)
	require.NoError(sc.T(), err)
	log.Infof("Number of schemas: %d", len(schemasCollection.Data))

	log.Info("Access the schema definition for each schema in the list using the schemadefinitions endpoint.")
	failedSchemaDefinitions, err := accessSchemaDefinitionForEachSchema(sc.client, schemasCollection, localCluster)
	require.NoError(sc.T(), err)
	require.Emptyf(sc.T(), failedSchemaDefinitions, "Failed to access schema definitions for schemas: %s", strings.Join(failedSchemaDefinitions, ", "))
}

func (sc *SchemaChangesTestSuite) TestSchemaDefinitionsExistenceForSchemaDownstreamCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	steveAdminClient, err := sc.client.Steve.ProxyDownstream(sc.cluster.ID)
	require.NoError(sc.T(), err)

	log.Info("Access the schemas endpoint on the downstream cluster and verify that it can be accessed successfully without any errors.")
	schemasCollection, err := steveAdminClient.SteveType(schemaSteveType).List(nil)
	require.NoError(sc.T(), err)
	log.Infof("Number of schemas: %d", len(schemasCollection.Data))

	log.Info("Access the schema definition for each schema in the list using the schemadefinitions endpoint.")
	failedSchemaDefinitions, err := accessSchemaDefinitionForEachSchema(sc.client, schemasCollection, sc.cluster.ID)
	require.NoError(sc.T(), err)
	require.Emptyf(sc.T(), failedSchemaDefinitions, "Failed to access schema definitions for schemas: %s", strings.Join(failedSchemaDefinitions, ", "))
}

func (sc *SchemaChangesTestSuite) TestPreferredVersionCheckLocalCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Access the schemas endpoint on the local cluster and verify that it can be accessed successfully without any errors.")
	schemasCollection, err := sc.client.Steve.SteveType(schemaSteveType).List(nil)
	require.NoError(sc.T(), err)
	log.Infof("Number of schemas: %d", len(schemasCollection.Data))

	log.Info("Verify that the version in each schema is the preferred version.")
	failedSchemaPreferredVersionCheck, err := checkPreferredVersion(sc.client, schemasCollection, localCluster)
	require.NoError(sc.T(), err)
	require.Emptyf(sc.T(), failedSchemaPreferredVersionCheck, "Failed preferred version checks: %v", failedSchemaPreferredVersionCheck)
}

func (sc *SchemaChangesTestSuite) TestPreferredVersionCheckDownstreamCluster() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	steveAdminClient, err := sc.client.Steve.ProxyDownstream(sc.cluster.ID)
	require.NoError(sc.T(), err)

	log.Info("Access the schemas endpoint on the downstream cluster and verify that it can be accessed successfully without any errors.")
	schemasCollection, err := steveAdminClient.SteveType(schemaSteveType).List(nil)
	require.NoError(sc.T(), err)
	log.Infof("Number of schemas: %d", len(schemasCollection.Data))

	log.Info("Verify that the version in each schema is the preferred version.")
	failedSchemaPreferredVersionCheck, err := checkPreferredVersion(sc.client, schemasCollection, sc.cluster.ID)
	require.NoError(sc.T(), err)
	require.Emptyf(sc.T(), failedSchemaPreferredVersionCheck, "Failed preferred version checks: %v", failedSchemaPreferredVersionCheck)
}

func (sc *SchemaChangesTestSuite) TestCRDCreateDeleteOperation() {
	subSession := sc.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Create a new Custom Resource Definition (CRD).")
	createYAML, err := os.ReadFile(crdCreateFilePath)
	require.NoError(sc.T(), err)

	yamlInput := &management.ImportClusterYamlInput{
		YAML: string(createYAML),
	}
	apply := []string{"kubectl", "apply", "-f", "/root/.kube/my-pod.yaml"}
	_, err = kubectl.Command(sc.client, yamlInput, localCluster, apply, "")
	require.NoError(sc.T(), err)

	log.Info("Access the schemas endpoint for the newly created CRD and verify that it can be accessed successfully without any errors.")
	err = kwait.PollUntilContextTimeout(context.Background(), defaults.FiveSecondTimeout, defaults.OneMinuteTimeout, true, func(ctx context.Context) (done bool, pollErr error) {
		_, pollErr = getSchemaByID(sc.client, localCluster, customSchemaID)
		if pollErr != nil {
			return false, pollErr
		}
		return true, nil
	})
	schemaResponse, err := getSchemaByID(sc.client, localCluster, customSchemaID)
	require.NoError(sc.T(), err)

	log.Info("Verify that the schema's resourceFields field is set to null.")
	id := schemaResponse["id"].(string)
	require.Nil(sc.T(), schemaResponse["resourceFields"], "ResourceFields should be null for schema "+id)

	log.Info("Verify that the schemas endpoint for the new CRD does not list the child schemas and trying to access any of the child schemas results in a '404 Not Found' error.")
	childSchemas := map[string]bool{
		"stable.example.com.v2.CronTab.spec":                      true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ManagedFieldsEntry": true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta":         true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference":     true,
	}

	for childSchemaID := range childSchemas {
		_, found := schemaResponse[childSchemaID]
		require.False(sc.T(), found, "Child schema should not be listed")

		_, err = getSchemaByID(sc.client, localCluster, childSchemaID)
		require.Error(sc.T(), err)
		errStatus := strings.Split(err.Error(), ": ")[1]
		require.Equal(sc.T(), "404", errStatus)
	}

	log.Info("Verify that the data returned from the schemas endpoint has the resourceMethods and collectionMethods fields.")
	require.NotNil(sc.T(), schemaResponse["resourceMethods"], "resourceMethods should be present in this schema")
	require.NotNil(sc.T(), schemaResponse["collectionMethods"], "collectionMethods should be present in this schema")

	log.Info("Access the schema definition for the new CRD and verify that it can be accessed successfully without any errors.")
	err = kwait.PollUntilContextTimeout(context.Background(), defaults.FiveSecondTimeout, defaults.OneMinuteTimeout, true, func(ctx context.Context) (done bool, pollErr error) {
		_, pollErr = getSchemaDefinitionByID(sc.client, localCluster, customSchemaID)
		if pollErr != nil {
			return false, pollErr
		}
		return true, nil
	})
	schemaResponse, err = getSchemaDefinitionByID(sc.client, localCluster, customSchemaID)
	require.NoError(sc.T(), err)

	log.Info("Verify that the data returned has definitions with resourceFields for each definition type, and that the resourceMethods and collectionMethods fields are present. Also, ensure that fields of type 'integer' in the YAML are parsed as 'int' in the schema definitions.")
	expectedDefinitions := map[string]bool{
		"com.example.stable.v2.CronTab":                           true,
		"com.example.stable.v2.CronTab.spec":                      true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ManagedFieldsEntry": true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta":         true,
		"io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference":     true,
	}

	for definitionID := range expectedDefinitions {
		definitionData, exists := schemaResponse["definitions"].(map[string]interface{})[definitionID]
		require.True(sc.T(), exists, "Expected definition %s not found in schemaResponse", definitionID)

		resourceFields, resourceFieldsExist := definitionData.(map[string]interface{})["resourceFields"]
		require.True(sc.T(), resourceFieldsExist, "ResourceFields should be present for definition %s", definitionID)
		require.NotNil(sc.T(), resourceFields, "ResourceFields should not be null for definition %s", definitionID)

		_, resourceMethodsExist := definitionData.(map[string]interface{})["resourceMethods"]
		require.False(sc.T(), resourceMethodsExist, "ResourceMethods field exists for definition %s", definitionID)

		_, collectionMethodsExist := definitionData.(map[string]interface{})["collectionMethods"]
		require.False(sc.T(), collectionMethodsExist, "CollectionMethods field exists for definition %s", definitionID)

		if resourceFields != nil {
			for fieldName, fieldData := range resourceFields.(map[string]interface{}) {
				fieldType := fieldData.(map[string]interface{})["type"].(string)
				require.NotEqual(sc.T(), "integer", fieldType, "Field %s should not be of type integer in definition %s", fieldName, definitionID)
				require.NotEqual(sc.T(), "number", fieldType, "Field %s should not be of type number in definition %s", fieldName, definitionID)
			}
		}
	}

	log.Info("Delete the CRD.")
	err = customresourcedefinitions.DeleteCustomResourceDefinition(sc.client, localCluster, "", customCrdName)
	require.NoError(sc.T(), err)

	log.Info("Access the schemas endpoint for the CRD and verify that it fails with a 404 error.")
	err = kwait.PollUntilContextTimeout(context.Background(), defaults.FiveSecondTimeout, defaults.OneMinuteTimeout, true, func(ctx context.Context) (done bool, pollErr error) {
		_, pollErr = getSchemaByID(sc.client, localCluster, customSchemaID)
		if pollErr != nil {
			return true, pollErr
		}
		return false, nil
	})
	require.Error(sc.T(), err)
	errStatus := strings.Split(err.Error(), ": ")[1]
	require.Equal(sc.T(), "404", errStatus)

	log.Info("Access the schema definition for the CRD and verify that it fails with a 404 error.")
	err = kwait.PollUntilContextTimeout(context.Background(), defaults.FiveSecondTimeout, defaults.OneMinuteTimeout, true, func(ctx context.Context) (done bool, pollErr error) {
		_, pollErr = getSchemaDefinitionByID(sc.client, localCluster, customSchemaID)
		if pollErr != nil {
			return true, pollErr
		}
		return false, nil
	})
	require.Error(sc.T(), err)
	errStatus = strings.Split(err.Error(), ": ")[1]
	require.Equal(sc.T(), "404", errStatus)
}

func TestSchemaChangesTestSuite(t *testing.T) {
	suite.Run(t, new(SchemaChangesTestSuite))
}
