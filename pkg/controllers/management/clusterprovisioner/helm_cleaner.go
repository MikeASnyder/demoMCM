package clusterprovisioner

import (
	"fmt"
	"strings"

	"github.com/helm/helm-mapkubeapis/pkg/mapping"
	"github.com/pkg/errors"
	v3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
	"github.com/rancher/rancher/pkg/wrangler"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	helmv3time "helm.sh/helm/v3/pkg/time"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type DeprecatedAPIData struct {
	DeprecatedAPIVersion string
	NewAPIVersion        string
	Kind                 string
	KubernetesVersion    string
}

var (
	// deprecatedAPIs is the list of deprecated APIs for which mappings should be generated
	deprecatedAPIs = []DeprecatedAPIData{
		{
			DeprecatedAPIVersion: "policy/v1beta1",
			Kind:                 "PodSecurityPolicy",
			KubernetesVersion:    "v1.25",
		},
		{
			DeprecatedAPIVersion: "batch/v1beta1",
			NewAPIVersion:        "batch/v1",
			Kind:                 "CronJob",
			KubernetesVersion:    "v1.25",
		},
		{
			DeprecatedAPIVersion: "autoscaling/v2beta1",
			NewAPIVersion:        "autoscaling/v2",
			Kind:                 "HorizontalPodAutoscaler",
			KubernetesVersion:    "v1.25",
		},
		{
			DeprecatedAPIVersion: "policy/v1beta1",
			NewAPIVersion:        "policy/v1",
			Kind:                 "PodDisruptionBudget",
			KubernetesVersion:    "v1.25",
		},
	}

	apiMappings = generateAPIMappings(deprecatedAPIs)

	// FeatureAppNS is a list of feature namespaces to clean up Helm releases from.
	FeatureAppNS = []string{
		"kube-system",                // Harvester, vSphere CPI, vSphere CSI
		"cattle-system",              // AKS/GKE/EKS Operator, Webhook, System Upgrade Controller
		"cattle-epinio-system",       // Epinio
		"cattle-fleet-system",        // Fleet
		"longhorn-system",            // Longhorn
		"cattle-neuvector-system",    // Neuvector
		"cattle-monitoring-system",   // Monitoring and Sub-charts
		"rancher-alerting-drivers",   // Alert Driver
		"cis-operator-system",        // CIS Benchmark
		"cattle-csp-adapter-system",  // CSP Adapter
		"cattle-externalip-system",   // External IP Webhook
		"cattle-gatekeeper-system",   // Gatekeeper
		"istio-system",               // Istio and Sub-charts
		"cattle-istio-system",        // Kiali
		"cattle-logging-system",      // Logging
		"cattle-windows-gmsa-system", // Windows GMSA
		"cattle-sriov-system",        // Sriov
		"cattle-ui-plugin-system",    // UI Plugin System
	}
)

// EmptyHelmDriverName is a placeholder for the empty Helm driver.
const EmptyHelmDriverName = ""

var (
	ErrorInvalidMapping = errors.New("invalid API version in mapping")
)

// generateAPIMappings generates the API mappings for replacement in Helm releases.
func generateAPIMappings(deprecatedAPIs []DeprecatedAPIData) *mapping.Metadata {
	var (
		apiVersionFormat = "apiVersion: %[1]s"
		kindFormat       = "kind: %[2]s"
		windowsLineBreak = "\r\n"
		linuxLineBreak   = "\n"

		formats = []string{
			apiVersionFormat + linuxLineBreak + kindFormat + linuxLineBreak,     // apiVersion: ... \n kind: ... \n
			kindFormat + linuxLineBreak + apiVersionFormat + linuxLineBreak,     // kind: ... \n apiVersion: ... \n
			apiVersionFormat + windowsLineBreak + kindFormat + windowsLineBreak, // apiVersion: ... \r\n kind: ... \r\n
			kindFormat + windowsLineBreak + apiVersionFormat + windowsLineBreak, // kind: ... \r\n apiVersion: ... \r\n
		}
	)

	mappings := mapping.Metadata{}

	for _, api := range deprecatedAPIs {
		for _, format := range formats {
			mappingItem := &mapping.Mapping{
				DeprecatedAPI:    fmt.Sprintf(format, api.DeprecatedAPIVersion, api.Kind),
				RemovedInVersion: api.KubernetesVersion,
			}

			if api.NewAPIVersion != "" {
				mappingItem.NewAPI = fmt.Sprintf(format, api.NewAPIVersion, api.Kind)
			}

			mappings.Mappings = append(mappings.Mappings, mappingItem)
		}
	}

	return &mappings
}

func newClientGetter(k8sClient kubernetes.Interface, restConfig rest.Config) *wrangler.SimpleRESTClientGetter {
	return &wrangler.SimpleRESTClientGetter{
		ClientConfig:    &clientcmd.DefaultClientConfig,
		RESTConfig:      &restConfig,
		CachedDiscovery: memory.NewMemCacheClient(k8sClient.Discovery()),
		RESTMapper:      meta.NewDefaultRESTMapper(nil),
	}
}

func (p *Provisioner) cleanupHelmReleases(cluster *v3.Cluster) error {
	clusterManager := p.clusterManager
	userContext, err := clusterManager.UserContextNoControllers(cluster.Name)
	if err != nil {
		return fmt.Errorf("[cleanupHelmReleases] failed to obtain the Kubernetes client instance: %w", err)
	}

	clientGetter := newClientGetter(userContext.K8sClient, userContext.RESTConfig)

	for _, namespace := range FeatureAppNS {
		actionConfig := &action.Configuration{}
		if err = actionConfig.Init(clientGetter, namespace, EmptyHelmDriverName, logrus.Infof); err != nil {
			return fmt.Errorf("[cleanupHelmReleases] failed to create ActionConfiguration instance for Helm: %w", err)
		}

		listAction := action.NewList(actionConfig)
		releases, err := listAction.Run()
		if err != nil {
			return fmt.Errorf("[cleanupHelmReleases] failed to list Helm releases for namespace %v: %w", namespace, err)
		}

		for _, helmRelease := range releases {
			lastRelease, err := actionConfig.Releases.Last(helmRelease.Name)
			if err != nil {
				return fmt.Errorf("[cleanupHelmReleases] failed to find latest release version for release %v: %w", helmRelease.Name, err)
			}

			// TODO consume the function from helm-mapkubeapis once that is merged in
			replaced, modifiedManifest, err := ReplaceManifestData(apiMappings, lastRelease.Manifest, cluster.Status.Version.GitVersion)
			if err != nil {
				// If this fails, it probably means we don't have adequate write permissions
				return fmt.Errorf("[cleanupHelmReleases] failed to replace deprecated/removed APIs on cluster %v: %w", cluster.Name, err)
			}

			if !replaced {
				logrus.Infof("[cleanupHelmReleases] release %v in namespace %v has no deprecated or removed APIs", lastRelease.Name, namespace)
				continue
			}

			if err := updateRelease(lastRelease, modifiedManifest, actionConfig); err != nil {
				return fmt.Errorf("[cleanupHelmReleases] failed to update release %v in namespace %v: %w", lastRelease.Name, lastRelease.Namespace, err)
			}
		}
	}

	return nil
}

// ReplaceManifestData replaces the out-of-date APIs with their respective valid successors, or removes an API that
// does not have a successor.
// Logic extracted from https://github.com/stormqueen1990/helm-mapkubeapis/blob/0245b7a7837a36fd164d83e496c453811d62c083/pkg/common/common.go#L81-L142
func ReplaceManifestData(mapMetadata *mapping.Metadata, manifest string, kubeVersion string) (bool, string, error) {
	if !semver.IsValid(kubeVersion) {
		return false, "", errors.Errorf("Invalid format for Kubernetes semantic version: %v", kubeVersion)
	}

	var replaced = false
	for _, mappingData := range mapMetadata.Mappings {
		deprecatedAPI := mappingData.DeprecatedAPI
		supportedAPI := mappingData.NewAPI
		var apiVersion string

		if mappingData.DeprecatedInVersion != "" {
			apiVersion = mappingData.DeprecatedInVersion
		} else {
			apiVersion = mappingData.RemovedInVersion
		}

		if !semver.IsValid(apiVersion) {
			logrus.Errorf("Failed to get the deprecated or removed Kubernetes version for API: %s", strings.ReplaceAll(deprecatedAPI, "\n", " "))
			return replaced, "", ErrorInvalidMapping
		}

		var count int
		if count = strings.Count(manifest, deprecatedAPI); count <= 0 {
			continue
		}

		if semver.Compare(apiVersion, kubeVersion) > 0 {
			logrus.Debugf("The following API does not require mapping as the "+
				"API is not deprecated or removed in Kubernetes '%s':\n\"%s\"\n", apiVersion,
				deprecatedAPI)
			continue
		}

		if supportedAPI == "" {
			logrus.Debugf("Found %d instances of deprecated or removed Kubernetes API:\n\"%s\"\n", count, deprecatedAPI)
			manifest = removeResourceWithNoSuccessors(count, manifest, deprecatedAPI)
		} else {
			logrus.Debugf("Found %d instances of deprecated or removed Kubernetes API:\n\"%s\"\nSupported API equivalent:\n\"%s\"\n", count, deprecatedAPI, supportedAPI)
			manifest = strings.ReplaceAll(manifest, deprecatedAPI, supportedAPI)
		}

		replaced = true
	}

	return replaced, manifest, nil
}

// removeResourceWithNoSuccessors removes a resource for which its respective API has no successors.
func removeResourceWithNoSuccessors(count int, manifest string, deprecatedAPI string) string {
	for repl := 0; repl < count; repl++ {
		// find the position where the API header is
		apiIndex := strings.Index(manifest, deprecatedAPI)

		// find the next separator index
		separatorIndex := strings.Index(manifest[apiIndex:], "---\n")

		// find the previous separator index
		previousSeparatorIndex := strings.LastIndex(manifest[:apiIndex], "---\n")

		/*
		 * if no previous separator index was found, it means the resource is at the beginning and not
		 * prefixed by ---
		 */
		if previousSeparatorIndex == -1 {
			previousSeparatorIndex = 0
		}

		if separatorIndex == -1 { // this means we reached the end of input
			manifest = manifest[:previousSeparatorIndex]
		} else {
			manifest = manifest[:previousSeparatorIndex] + manifest[separatorIndex+apiIndex:]
		}
	}

	manifest = strings.Trim(manifest, "\n")
	return manifest
}

// updateRelease updates a release in the cluster with an equivalent with the superseded APIs replaced or removed as
// needed.
// Logic extracted from https://github.com/helm/helm-mapkubeapis/blob/main/pkg/v3/release.go#L71-L94
func updateRelease(originalRelease *release.Release, modifiedManifest string, config *action.Configuration) error {
	originalRelease.SetStatus(release.StatusSuperseded, "")
	if err := config.Releases.Update(originalRelease); err != nil {
		return fmt.Errorf("[updateRelease] failed to update original release %v in namespace %v: %w", originalRelease.Name, originalRelease.Namespace, err)
	}

	newRelease := copyReleaseData(originalRelease, modifiedManifest, config.Now())

	logrus.Infof("[updateRelease] add release version %v for release %v with updated supported APIs in namespace %v", originalRelease.Version, originalRelease.Name, originalRelease.Namespace)

	if err := config.Releases.Create(newRelease); err != nil {
		originalRelease.SetStatus(release.StatusDeployed, "")
		updateErr := config.Releases.Update(newRelease)
		if updateErr != nil {
			return fmt.Errorf("[updateRelease] failed to create new release version %v for release %v in namespace %v and failed to rollback to previous version: %w", newRelease.Version, newRelease.Name, newRelease.Namespace, err)
		}

		return fmt.Errorf("[updateRelease] failed to create new release version %v for release %v in namespace %v: %w", newRelease.Version, newRelease.Name, newRelease.Namespace, err)
	}

	logrus.Infof("[updateRelease] successfully created new version for release %v in namespace %v", newRelease.Name, newRelease.Namespace)

	return nil
}

// copyReleaseData copies the original release data into a new object in order to retain previous release information
// for rollback, if needed.
func copyReleaseData(originalRelease *release.Release, newManifest string, lastDeployed helmv3time.Time) *release.Release {
	newRelease := release.Release{
		Name:      originalRelease.Name,
		Info:      originalRelease.Info,
		Chart:     originalRelease.Chart,
		Config:    originalRelease.Config,
		Manifest:  newManifest,
		Hooks:     originalRelease.Hooks,
		Version:   originalRelease.Version + 1,
		Namespace: originalRelease.Namespace,
		Labels:    originalRelease.Labels,
	}

	newRelease.Info.Description = UpgradeDescription
	newRelease.Info.LastDeployed = lastDeployed
	newRelease.SetStatus(release.StatusDeployed, "")

	return &newRelease
}