package aks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"

	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2020-09-01/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-07-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/mcuadros/go-version"
)

type virtualNetworksResponseBody struct {
	Name          string   `json:"name"`
	ResourceGroup string   `json:"resourceGroup"`
	Subnets       []subnet `json:"subnets"`
}

type subnet struct {
	Name         string `json:"name"`
	AddressRange string `json:"addressRange"`
}

var matchAzureNetwork = regexp.MustCompile("/resourceGroups/(.+?)/")

func NewClientAuthorizer(cap *Capabilities) (autorest.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(cap.AuthBaseURL, cap.TenantID)
	if err != nil {
		return nil, err
	}

	spToken, err := adal.NewServicePrincipalToken(*oauthConfig, cap.ClientID, cap.ClientSecret, cap.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("couldn't authenticate to Azure cloud with error: %v", err)
	}

	return autorest.NewBearerAuthorizer(spToken), nil
}

func NewContainerServiceClient(cap *Capabilities) (*containerservice.ContainerServicesClient, error) {
	authorizer, err := NewClientAuthorizer(cap)
	if err != nil {
		return nil, err
	}

	containerService := containerservice.NewContainerServicesClientWithBaseURI(cap.BaseURL, cap.SubscriptionID)
	containerService.Authorizer = authorizer

	return &containerService, nil
}

func NewNetworkServiceClient(cap *Capabilities) (*network.VirtualNetworksClient, error) {
	authorizer, err := NewClientAuthorizer(cap)
	if err != nil {
		return nil, err
	}

	containerService := network.NewVirtualNetworksClientWithBaseURI(cap.BaseURL, cap.SubscriptionID)
	containerService.Authorizer = authorizer

	return &containerService, nil
}

type sortableVersion []string

func (s sortableVersion) Len() int {
	return len(s)
}

func (s sortableVersion) Swap(a, b int) {
	s[a], s[b] = s[b], s[a]
}

func (s sortableVersion) Less(a, b int) bool {
	return version.Compare(s[a], s[b], "<")
}

func listKubernetesVersions(ctx context.Context, cap *Capabilities) ([]byte, int, error) {
	if cap.TenantID == "" || cap.ResourceLocation == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("tenantId and region are required")
	}

	clientContainer, err := NewContainerServiceClient(cap)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	orchestrators, err := clientContainer.ListOrchestrators(ctx, cap.ResourceLocation, "managedClusters")
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to get orchestrators: %v", err)
	}

	if orchestrators.Orchestrators == nil {
		return nil, http.StatusBadRequest, fmt.Errorf("no version profiles returned: %v", err)
	}

	var kubernetesVersions []string

	for _, profile := range *orchestrators.Orchestrators {
		if profile.OrchestratorType == nil || profile.OrchestratorVersion == nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("unexpected nil orchestrator type or version")
		}

		if *profile.OrchestratorType == "Kubernetes" {
			kubernetesVersions = append(kubernetesVersions, *profile.OrchestratorVersion)
		}
	}

	sort.Sort(sortableVersion(kubernetesVersions))

	return encodeOutput(kubernetesVersions)
}

func listVirtualNetworks(ctx context.Context, cap *Capabilities) ([]byte, int, error) {
	if cap.TenantID == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("tenantId is required")
	}

	clientNetwork, err := NewNetworkServiceClient(cap)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	result, err := clientNetwork.ListAll(ctx)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to get networks: %v", err)
	}

	var networks []virtualNetworksResponseBody

	for result.NotDone() {
		var batch []virtualNetworksResponseBody

		for _, azureNetwork := range result.Values() {
			var subnets []subnet

			if azureNetwork.Subnets != nil {
				for _, azureSubnet := range *azureNetwork.Subnets {
					if azureSubnet.Name != nil {
						subnets = append(subnets, subnet{
							Name:         *azureSubnet.Name,
							AddressRange: *azureSubnet.AddressPrefix,
						})
					}
				}
			}

			if azureNetwork.ID == nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("no ID on virtual network")
			}

			match := matchAzureNetwork.FindStringSubmatch(*azureNetwork.ID)

			if len(match) < 2 || match[1] == "" {
				return nil, http.StatusInternalServerError, fmt.Errorf("could not parse virtual network ID")
			}

			if azureNetwork.Name == nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("no name on virtual network")
			}

			batch = append(batch, virtualNetworksResponseBody{
				Name:          *azureNetwork.Name,
				ResourceGroup: match[1],
				Subnets:       subnets,
			})
		}

		networks = append(networks, batch...)

		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	return encodeOutput(networks)
}

func encodeOutput(result interface{}) ([]byte, int, error) {
	data, err := json.Marshal(&result)
	if err != nil {
		return data, http.StatusInternalServerError, err
	}

	return data, http.StatusOK, err
}
