package cluster

import (
	"context"

	apisV1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	rkev1 "github.com/rancher/rancher/pkg/apis/rke.cattle.io/v1"
	"github.com/rancher/rancher/tests/automation-framework/clients"
	"github.com/rancher/rancher/tests/automation-framework/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewRKE2ClusterConfig(clusterName, namespace, cni, cloudCredentialSecretName, kubernetesVersion string, machinePools []apisV1.RKEMachinePool) *apisV1.Cluster {
	typeMeta := metav1.TypeMeta{
		Kind:       "Cluster",
		APIVersion: "provisioning.cattle.io/v1",
	}

	//metav1.ObjectMeta
	objectMeta := metav1.ObjectMeta{
		Name:      clusterName,
		Namespace: namespace,
	}

	etcd := &rkev1.ETCD{
		DisableSnapshots:     false,
		S3:                   nil,
		SnapshotRetention:    5,
		SnapshotScheduleCron: "0 */5 * * *",
	}

	chartValuesMap := rkev1.GenericMap{
		Data: map[string]interface{}{},
	}

	machineGlobalConfigMap := rkev1.GenericMap{
		Data: map[string]interface{}{
			"cni":                 cni,
			"disable-kube-proxy":  false,
			"etcd-expose-metrics": false,
			"profile":             nil,
		},
	}

	localClusterAuthEndpoint := rkev1.LocalClusterAuthEndpoint{
		CACerts: "",
		Enabled: false,
		FQDN:    "",
	}

	upgradeStrategy := rkev1.ClusterUpgradeStrategy{
		ControlPlaneConcurrency:  "10%",
		ControlPlaneDrainOptions: rkev1.DrainOptions{},
		WorkerConcurrency:        "10%",
		WorkerDrainOptions:       rkev1.DrainOptions{},
	}

	rkeSpecCommon := rkev1.RKEClusterSpecCommon{
		ChartValues:              chartValuesMap,
		MachineGlobalConfig:      machineGlobalConfigMap,
		ETCD:                     etcd,
		LocalClusterAuthEndpoint: localClusterAuthEndpoint,
		UpgradeStrategy:          upgradeStrategy,
		MachineSelectorConfig:    []rkev1.RKESystemConfig{},
	}

	rkeConfig := &apisV1.RKEConfig{
		RKEClusterSpecCommon: rkeSpecCommon,
		MachinePools:         machinePools,
	}

	spec := apisV1.ClusterSpec{
		CloudCredentialSecretName:            cloudCredentialSecretName,
		KubernetesVersion:                    kubernetesVersion,
		DefaultPodSecurityPolicyTemplateName: "",

		RKEConfig: rkeConfig,
	}

	v1Cluster := &apisV1.Cluster{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Spec:       spec,
	}

	return v1Cluster
}

func ProvisionCluster(clusterName, bearerToken, cloudCredentialName string, client *clients.Client, rkeMachinePools []apisV1.RKEMachinePool) (func() error, error) {
	configuration := config.GetInstance()
	ctx := context.Background()

	clusterConfig := NewRKE2ClusterConfig(clusterName, configuration.GetDefaultNamespace(), configuration.GetCNI(), cloudCredentialName, configuration.GetKubernetesVersion(), rkeMachinePools)

	v1Cluster, err := client.Provisioning.Clusters(configuration.GetDefaultNamespace()).Create(ctx, clusterConfig, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	clusterCleanup := func() error {
		err := client.Provisioning.Clusters(configuration.GetDefaultNamespace()).Delete(ctx, v1Cluster.Name, metav1.DeleteOptions{})
		return err
	}

	return clusterCleanup, nil
}
