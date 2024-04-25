package schemas

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rancher/shepherd/clients/rancher"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
)

const (
	schemaSteveType           = "schema"
	schemaDefinitionSteveType = "schemaDefinition"
	schemaEndpoint            = "schema"
	schemaDefinitionEndpoint  = "schemaDefinition"
	apiGroupEndpoint          = "apigroup"
	localCluster              = "local"
	projectSchemaID           = "management.cattle.io.project"
	namespaceSchemaID         = "namespace"
	autoscalingSchemaID       = "autoscaling.horizontalpodautoscaler"
	customSchemaID            = "stable.example.com.crontab"
	customCrdName             = "crontabs.stable.example.com"
	crdCreateFilePath         = "./resources/crd_create.yaml"
)

var exceptionMap = map[string]bool{
	"applyOutput":              true,
	"applyInput":               true,
	"apiRoot":                  true,
	"chartActionOutput":        true,
	"chartInstall":             true,
	"chartInstallAction":       true,
	"chartUpgrade":             true,
	"chartUpgradeAction":       true,
	"chartUninstallAction":     true,
	"count":                    true,
	"generateKubeconfigOutput": true,
	"schema":                   true,
	"schemaDefinition":         true,
	"subscribe":                true,
	"userpreference":           true,
}

func getSchemaByID(client *rancher.Client, clusterID, existingSchemaID string) (map[string]interface{}, error) {
	return getJSONResponse(client, clusterID, schemaEndpoint, existingSchemaID)
}

func getSchemaDefinitionByID(client *rancher.Client, clusterID, existingSchemaID string) (map[string]interface{}, error) {
	return getJSONResponse(client, clusterID, schemaDefinitionEndpoint, existingSchemaID)
}

func getAPIGroupInfoByAPIGroupName(client *rancher.Client, apiGroupName string) (map[string]interface{}, error) {
	return getJSONResponse(client, "", apiGroupEndpoint, apiGroupName)
}

func accessSchemaDefinitionForEachSchema(client *rancher.Client, schemasCollection *v1.SteveCollection, clusterID string) ([]string, error) {
	var failedSchemaDefinitions []string
	var err error
	var schemaID string

	for _, schema := range schemasCollection.Data {
		schemaID = schema.JSONResp["id"].(string)

		if _, exists := exceptionMap[schemaID]; !exists {
			_, err = getSchemaByID(client, clusterID, schemaID)
			if err != nil {
				return nil, err
			}
			_, err = getSchemaDefinitionByID(client, clusterID, schemaID)
			if err != nil {
				failedSchemaDefinitions = append(failedSchemaDefinitions, schemaID)
			}
		}
	}
	return failedSchemaDefinitions, nil
}

func checkPreferredVersion(client *rancher.Client, schemasCollection *v1.SteveCollection, clusterID string) ([][]string, error) {
	var failedSchemaPreferredVersionCheck [][]string

	for _, schema := range schemasCollection.Data {
		schemaID := schema.JSONResp["id"].(string)

		if _, exists := exceptionMap[schemaID]; !exists {
			schemaInfo, err := getSchemaByID(client, clusterID, schemaID)
			if err != nil {
				return nil, err
			}
			schemaVersion := schemaInfo["attributes"].(map[string]interface{})["version"].(string)

			apiGroupName := schemaInfo["attributes"].(map[string]interface{})["group"].(string)
			if apiGroupName == "" {
				continue
			}
			apiGroupInfo, err := getAPIGroupInfoByAPIGroupName(client, apiGroupName)
			if err != nil {
				return nil, err
			}
			preferredVersion := apiGroupInfo["preferredVersion"].(map[string]interface{})["version"].(string)

			if schemaVersion != preferredVersion {
				failedSchemaPreferredVersionCheck = append(failedSchemaPreferredVersionCheck, []string{
					"Schema ID: " + schemaID + ",",
					"Schema Version: " + schemaVersion + ";",
					"API Group Name: " + apiGroupName + ",",
					"API Group Version: " + preferredVersion,
				})
			}
		}
	}
	return failedSchemaPreferredVersionCheck, nil
}

func getJSONResponse(client *rancher.Client, clusterID, endpointType, existingID string) (map[string]interface{}, error) {
	rancherURL := client.RancherConfig.Host
	token := client.RancherConfig.AdminToken

	var httpURL string
	if endpointType == apiGroupEndpoint {
		httpURL = fmt.Sprintf("https://%s/apis/%s", rancherURL, existingID)
	} else if clusterID == "local" {
		httpURL = fmt.Sprintf("https://%s/v1/%s/%s", rancherURL, endpointType, existingID)
	} else {
		httpURL = fmt.Sprintf("https://%s/k8s/clusters/%s/v1/%s/%s", rancherURL, clusterID, endpointType, existingID)
	}

	req, err := http.NewRequest("GET", httpURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	byteObject, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jsonObject map[string]interface{}
	if err := json.Unmarshal(byteObject, &jsonObject); err != nil {
		return nil, err
	}

	return jsonObject, nil
}
