//go:build (validation || infra.any || cluster.any || extended) && !sanity && !stress

package projects

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/charts"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/kubeapi/namespaces"
	"github.com/rancher/shepherd/extensions/kubeapi/projects"
	"github.com/rancher/shepherd/extensions/users"
	"github.com/rancher/shepherd/pkg/session"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProjectsContainerResourceLimitValidationTestSuite struct {
	suite.Suite
	client  *rancher.Client
	session *session.Session
	cluster *management.Cluster
}

func (pcrlv *ProjectsContainerResourceLimitValidationTestSuite) TearDownSuite() {
	pcrlv.session.Cleanup()
}

func (pcrlv *ProjectsContainerResourceLimitValidationTestSuite) SetupSuite() {
	pcrlv.session = session.NewSession()

	client, err := rancher.NewClient("", pcrlv.session)
	require.NoError(pcrlv.T(), err)

	pcrlv.client = client

	clusterName := client.RancherConfig.ClusterName
	require.NotEmptyf(pcrlv.T(), clusterName, "Cluster name to install should be set")
	clusterID, err := clusters.GetClusterIDByName(pcrlv.client, clusterName)
	require.NoError(pcrlv.T(), err, "Error getting cluster ID")
	pcrlv.cluster, err = pcrlv.client.Management.Cluster.ByID(clusterID)
	require.NoError(pcrlv.T(), err)
}

func (pcrlv *ProjectsContainerResourceLimitValidationTestSuite) TestCpuAndMemoryLimitLessThanRequest() {
	subSession := pcrlv.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Create a project in the downstream cluster with CPU and Memory request set greater than CPU and Memory limit. Verify that the webhook rejects the request.")
	cpuLimit := "100m"
	cpuReservation := "200m"
	memoryLimit := "32Mi"
	memoryReservation := "64Mi"

	standardUser, err := users.CreateUserWithRole(pcrlv.client, users.UserConfig(), projects.StandardUser)
	require.NoError(pcrlv.T(), err, "Failed to create standard user")
	standardUserClient, err := pcrlv.client.AsUser(standardUser)
	require.NoError(pcrlv.T(), err)
	err = users.AddClusterRoleToUser(pcrlv.client, pcrlv.cluster, standardUser, clusterOwner, nil)
	require.NoError(pcrlv.T(), err, "Failed to add the user as a cluster owner to the downstream cluster")

	_, err = createProject(standardUserClient, pcrlv.cluster.ID, "", "", cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.Error(pcrlv.T(), err)
	pattern := fmt.Sprintf(`admission webhook "rancher.cattle.io.projects.management.cattle.io" denied the request: project.spec.containerDefaultResourceLimit: Invalid value: v3.ContainerResourceLimit{RequestsCPU:"%s", RequestsMemory:"%s", LimitsCPU:"%s", LimitsMemory:"%s"}: requested CPU %s is greater than limit %s\nproject.spec.containerDefaultResourceLimit: Invalid value: v3.ContainerResourceLimit{RequestsCPU:"%s", RequestsMemory:"%s", LimitsCPU:"%s", LimitsMemory:"%s"}: requested memory %s is greater than limit %s`, cpuReservation, memoryReservation, cpuLimit, memoryLimit, cpuReservation, cpuLimit, cpuReservation, memoryReservation, cpuLimit, memoryLimit, memoryReservation, memoryLimit)
	require.Regexp(pcrlv.T(), regexp.MustCompile(pattern), err.Error())
}

func (pcrlv *ProjectsContainerResourceLimitValidationTestSuite) TestCpuLimitLessThanRequest() {
	subSession := pcrlv.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Create a project in the downstream cluster with CPU request set greater than the CPU limit. Verify that the webhook rejects the request.")
	cpuLimit := "100m"
	cpuReservation := "200m"
	memoryLimit := ""
	memoryReservation := ""

	standardUser, err := users.CreateUserWithRole(pcrlv.client, users.UserConfig(), projects.StandardUser)
	require.NoError(pcrlv.T(), err, "Failed to create standard user")
	standardUserClient, err := pcrlv.client.AsUser(standardUser)
	require.NoError(pcrlv.T(), err)
	err = users.AddClusterRoleToUser(pcrlv.client, pcrlv.cluster, standardUser, clusterOwner, nil)
	require.NoError(pcrlv.T(), err, "Failed to add the user as a cluster owner to the downstream cluster")

	_, err = createProject(standardUserClient, pcrlv.cluster.ID, "", "", cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.Error(pcrlv.T(), err)
	pattern := fmt.Sprintf(`admission webhook "rancher.cattle.io.projects.management.cattle.io" denied the request: project.spec.containerDefaultResourceLimit: Invalid value: v3.ContainerResourceLimit{RequestsCPU:"%s", RequestsMemory:"%s", LimitsCPU:"%s", LimitsMemory:"%s"}: requested CPU %s is greater than limit %s`, cpuReservation, memoryReservation, cpuLimit, memoryLimit, cpuReservation, cpuLimit)
	require.Regexp(pcrlv.T(), regexp.MustCompile(pattern), err.Error())
}

func (pcrlv *ProjectsContainerResourceLimitValidationTestSuite) TestMemoryLimitLessThanRequest() {
	subSession := pcrlv.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Create a project in the downstream cluster with Memory request set greater than the Memory limit. Verify that the webhook rejects the request.")
	cpuLimit := ""
	cpuReservation := ""
	memoryLimit := "32Mi"
	memoryReservation := "64Mi"

	standardUser, err := users.CreateUserWithRole(pcrlv.client, users.UserConfig(), projects.StandardUser)
	require.NoError(pcrlv.T(), err, "Failed to create standard user")
	standardUserClient, err := pcrlv.client.AsUser(standardUser)
	require.NoError(pcrlv.T(), err)
	err = users.AddClusterRoleToUser(pcrlv.client, pcrlv.cluster, standardUser, clusterOwner, nil)
	require.NoError(pcrlv.T(), err, "Failed to add the user as a cluster owner to the downstream cluster")

	_, err = createProject(standardUserClient, pcrlv.cluster.ID, "", "", cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.Error(pcrlv.T(), err)
	pattern := fmt.Sprintf(`admission webhook "rancher.cattle.io.projects.management.cattle.io" denied the request: project.spec.containerDefaultResourceLimit: Invalid value: v3.ContainerResourceLimit{RequestsCPU:"%s", RequestsMemory:"%s", LimitsCPU:"%s", LimitsMemory:"%s"}: requested memory %s is greater than limit %s`, cpuReservation, memoryReservation, cpuLimit, memoryLimit, memoryReservation, memoryLimit)
	require.Regexp(pcrlv.T(), regexp.MustCompile(pattern), err.Error())
}

func (pcrlv *ProjectsContainerResourceLimitValidationTestSuite) TestValidCpuLimitButMemoryLimitLessThanRequest() {
	subSession := pcrlv.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Create a project in the downstream cluster with CPU request set lower than the CPU limit but Memory request set greater than the Memory Request. Verify that the webhook rejects the request.")
	cpuLimit := "200m"
	cpuReservation := "100m"
	memoryLimit := "32Mi"
	memoryReservation := "64Mi"

	standardUser, err := users.CreateUserWithRole(pcrlv.client, users.UserConfig(), projects.StandardUser)
	require.NoError(pcrlv.T(), err, "Failed to create standard user")
	standardUserClient, err := pcrlv.client.AsUser(standardUser)
	require.NoError(pcrlv.T(), err)
	err = users.AddClusterRoleToUser(pcrlv.client, pcrlv.cluster, standardUser, clusterOwner, nil)
	require.NoError(pcrlv.T(), err, "Failed to add the user as a cluster owner to the downstream cluster")

	_, err = createProject(standardUserClient, pcrlv.cluster.ID, "", "", cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.Error(pcrlv.T(), err)
	pattern := fmt.Sprintf(`admission webhook "rancher.cattle.io.projects.management.cattle.io" denied the request: project.spec.containerDefaultResourceLimit: Invalid value: v3.ContainerResourceLimit{RequestsCPU:"%s", RequestsMemory:"%s", LimitsCPU:"%s", LimitsMemory:"%s"}: requested memory %s is greater than limit %s`, cpuReservation, memoryReservation, cpuLimit, memoryLimit, memoryReservation, memoryLimit)
	require.Regexp(pcrlv.T(), regexp.MustCompile(pattern), err.Error())
}

func (pcrlv *ProjectsContainerResourceLimitValidationTestSuite) TestCpuAndMemoryLimitEqualToRequest() {
	subSession := pcrlv.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Create a project (with CPU and Memory limit equal to Request) and a namespace in the project.")
	cpuLimit := "200m"
	cpuReservation := "200m"
	memoryLimit := "64Mi"
	memoryReservation := "64Mi"

	standardUser, err := users.CreateUserWithRole(pcrlv.client, users.UserConfig(), projects.StandardUser)
	require.NoError(pcrlv.T(), err, "Failed to create standard user")
	standardUserClient, err := pcrlv.client.AsUser(standardUser)
	require.NoError(pcrlv.T(), err)
	err = users.AddClusterRoleToUser(pcrlv.client, pcrlv.cluster, standardUser, clusterOwner, nil)
	require.NoError(pcrlv.T(), err, "Failed to add the user as a cluster owner to the downstream cluster")

	createdProject, createdNamespace, err := createProjectAndNamespace(standardUserClient, pcrlv.cluster.ID, "", "", cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)

	log.Info("Verify that the container default resource limit in the Project spec is accurate.")
	projectSpec := createdProject.Spec.ContainerDefaultResourceLimit
	require.Equal(pcrlv.T(), cpuLimit, projectSpec.LimitsCPU, "CPU limit mismatch")
	require.Equal(pcrlv.T(), cpuReservation, projectSpec.RequestsCPU, "CPU reservation mismatch")
	require.Equal(pcrlv.T(), memoryLimit, projectSpec.LimitsMemory, "Memory limit mismatch")
	require.Equal(pcrlv.T(), memoryReservation, projectSpec.RequestsMemory, "Memory reservation mismatch")

	log.Info("Verify that the namespace has the label and annotation referencing the project.")
	updatedNamespace, err := namespaces.GetNamespaceByName(standardUserClient, pcrlv.cluster.ID, createdNamespace.Name)
	require.NoError(pcrlv.T(), err)
	err = checkNamespaceLabelsAndAnnotations(pcrlv.cluster.ID, createdProject.Name, updatedNamespace)
	require.NoError(pcrlv.T(), err)

	log.Info("Verify that the limit range object is created for the namespace and the resource limit in the limit range is accurate.")
	err = checkLimitRange(standardUserClient, pcrlv.cluster.ID, updatedNamespace.Name, cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)

	log.Info("Create a deployment in the namespace with one replica and verify that a pod is created.")
	createdDeployment, err := createDeployment(standardUserClient, pcrlv.cluster.ID, updatedNamespace.Name, 1)
	require.NoError(pcrlv.T(), err, "Failed to create deployment in the namespace")
	err = charts.WatchAndWaitDeployments(standardUserClient, pcrlv.cluster.ID, updatedNamespace.Name, metav1.ListOptions{
		FieldSelector: "metadata.name=" + createdDeployment.Name,
	})
	require.NoError(pcrlv.T(), err)

	log.Info("Verify that the resource limits and requests for the container in the pod spec is accurate.")
	err = checkContainerResources(standardUserClient, pcrlv.cluster.ID, updatedNamespace.Name, createdDeployment.Name, cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)
}

func (pcrlv *ProjectsContainerResourceLimitValidationTestSuite) TestCpuAndMemoryLimitGreaterThanRequest() {
	subSession := pcrlv.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Create a project (with CPU and Memory limit greater than Request) and a namespace in the project.")
	cpuLimit := "200m"
	cpuReservation := "100m"
	memoryLimit := "64Mi"
	memoryReservation := "32Mi"

	standardUser, err := users.CreateUserWithRole(pcrlv.client, users.UserConfig(), projects.StandardUser)
	require.NoError(pcrlv.T(), err, "Failed to create standard user")
	standardUserClient, err := pcrlv.client.AsUser(standardUser)
	require.NoError(pcrlv.T(), err)
	err = users.AddClusterRoleToUser(pcrlv.client, pcrlv.cluster, standardUser, clusterOwner, nil)
	require.NoError(pcrlv.T(), err, "Failed to add the user as a cluster owner to the downstream cluster")

	createdProject, createdNamespace, err := createProjectAndNamespace(standardUserClient, pcrlv.cluster.ID, "", "", cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)

	log.Info("Verify that the container default resource limit in the Project spec is accurate.")
	projectSpec := createdProject.Spec.ContainerDefaultResourceLimit
	require.Equal(pcrlv.T(), cpuLimit, projectSpec.LimitsCPU, "CPU limit mismatch")
	require.Equal(pcrlv.T(), cpuReservation, projectSpec.RequestsCPU, "CPU reservation mismatch")
	require.Equal(pcrlv.T(), memoryLimit, projectSpec.LimitsMemory, "Memory limit mismatch")
	require.Equal(pcrlv.T(), memoryReservation, projectSpec.RequestsMemory, "Memory reservation mismatch")

	log.Info("Verify that the namespace has the label and annotation referencing the project.")
	updatedNamespace, err := namespaces.GetNamespaceByName(standardUserClient, pcrlv.cluster.ID, createdNamespace.Name)
	require.NoError(pcrlv.T(), err)
	err = checkNamespaceLabelsAndAnnotations(pcrlv.cluster.ID, createdProject.Name, updatedNamespace)
	require.NoError(pcrlv.T(), err)

	log.Info("Verify that the limit range object is created for the namespace and the resource limit in the limit range is accurate.")
	err = checkLimitRange(standardUserClient, pcrlv.cluster.ID, updatedNamespace.Name, cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)

	log.Info("Create a deployment in the namespace with one replica and verify that a pod is created.")
	createdDeployment, err := createDeployment(standardUserClient, pcrlv.cluster.ID, updatedNamespace.Name, 1)
	require.NoError(pcrlv.T(), err, "Failed to create deployment in the namespace")
	err = charts.WatchAndWaitDeployments(standardUserClient, pcrlv.cluster.ID, updatedNamespace.Name, metav1.ListOptions{
		FieldSelector: "metadata.name=" + createdDeployment.Name,
	})
	require.NoError(pcrlv.T(), err)

	log.Info("Verify that the resource limits and requests for the container in the pod spec is accurate.")
	err = checkContainerResources(standardUserClient, pcrlv.cluster.ID, updatedNamespace.Name, createdDeployment.Name, cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)
}

func (pcrlv *ProjectsContainerResourceLimitValidationTestSuite) TestUpdateProjectWithCpuAndMemoryLimitLessThanRequest() {
	subSession := pcrlv.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Create a project (with valid container default resource limit) and a namespace in the project.")
	cpuLimit := "100m"
	cpuReservation := "50m"
	memoryLimit := "64Mi"
	memoryReservation := "32Mi"

	standardUser, err := users.CreateUserWithRole(pcrlv.client, users.UserConfig(), projects.StandardUser)
	require.NoError(pcrlv.T(), err, "Failed to create standard user")
	standardUserClient, err := pcrlv.client.AsUser(standardUser)
	require.NoError(pcrlv.T(), err)
	err = users.AddClusterRoleToUser(pcrlv.client, pcrlv.cluster, standardUser, clusterOwner, nil)
	require.NoError(pcrlv.T(), err, "Failed to add the user as a cluster owner to the downstream cluster")

	createdProject, createdNamespace, err := createProjectAndNamespace(standardUserClient, pcrlv.cluster.ID, "", "", cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)

	log.Info("Verify that the container default resource limit in the Project spec is accurate.")
	projectSpec := createdProject.Spec.ContainerDefaultResourceLimit
	require.Equal(pcrlv.T(), cpuLimit, projectSpec.LimitsCPU, "CPU limit mismatch")
	require.Equal(pcrlv.T(), cpuReservation, projectSpec.RequestsCPU, "CPU reservation mismatch")
	require.Equal(pcrlv.T(), memoryLimit, projectSpec.LimitsMemory, "Memory limit mismatch")
	require.Equal(pcrlv.T(), memoryReservation, projectSpec.RequestsMemory, "Memory reservation mismatch")

	log.Info("Verify that the limit range object is created for the namespace and the resource limit in the limit range is accurate.")
	err = checkLimitRange(standardUserClient, pcrlv.cluster.ID, createdNamespace.Name, cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)

	log.Info("Update the project with CPU and Memory request set greater than CPU and Memory limit. Verify that the webhook rejects the request.")
	cpuLimit = "100m"
	cpuReservation = "200m"
	memoryLimit = "32Mi"
	memoryReservation = "64Mi"
	_, err = updateProjectContainerResourceLimit(standardUserClient, createdProject, cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.Error(pcrlv.T(), err)
	pattern := fmt.Sprintf(`admission webhook "rancher.cattle.io.projects.management.cattle.io" denied the request: project.spec.containerDefaultResourceLimit: Invalid value: v3.ContainerResourceLimit{RequestsCPU:"%s", RequestsMemory:"%s", LimitsCPU:"%s", LimitsMemory:"%s"}: requested CPU %s is greater than limit %s\nproject.spec.containerDefaultResourceLimit: Invalid value: v3.ContainerResourceLimit{RequestsCPU:"%s", RequestsMemory:"%s", LimitsCPU:"%s", LimitsMemory:"%s"}: requested memory %s is greater than limit %s`, cpuReservation, memoryReservation, cpuLimit, memoryLimit, cpuReservation, cpuLimit, cpuReservation, memoryReservation, cpuLimit, memoryLimit, memoryReservation, memoryLimit)
	require.Regexp(pcrlv.T(), regexp.MustCompile(pattern), err.Error())
}

func (pcrlv *ProjectsContainerResourceLimitValidationTestSuite) TestUpdateProjectWithCpuLimitLessThanRequest() {
	subSession := pcrlv.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Create a project (with valid container default resource limit) and a namespace in the project.")
	cpuLimit := "100m"
	cpuReservation := "50m"
	memoryLimit := "64Mi"
	memoryReservation := "32Mi"

	standardUser, err := users.CreateUserWithRole(pcrlv.client, users.UserConfig(), projects.StandardUser)
	require.NoError(pcrlv.T(), err, "Failed to create standard user")
	standardUserClient, err := pcrlv.client.AsUser(standardUser)
	require.NoError(pcrlv.T(), err)
	err = users.AddClusterRoleToUser(pcrlv.client, pcrlv.cluster, standardUser, clusterOwner, nil)
	require.NoError(pcrlv.T(), err, "Failed to add the user as a cluster owner to the downstream cluster")

	createdProject, createdNamespace, err := createProjectAndNamespace(standardUserClient, pcrlv.cluster.ID, "", "", cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)

	log.Info("Verify that the container default resource limit in the Project spec is accurate.")
	projectSpec := createdProject.Spec.ContainerDefaultResourceLimit
	require.Equal(pcrlv.T(), cpuLimit, projectSpec.LimitsCPU, "CPU limit mismatch")
	require.Equal(pcrlv.T(), cpuReservation, projectSpec.RequestsCPU, "CPU reservation mismatch")
	require.Equal(pcrlv.T(), memoryLimit, projectSpec.LimitsMemory, "Memory limit mismatch")
	require.Equal(pcrlv.T(), memoryReservation, projectSpec.RequestsMemory, "Memory reservation mismatch")

	log.Info("Verify that the limit range object is created for the namespace and the resource limit in the limit range is accurate.")
	err = checkLimitRange(standardUserClient, pcrlv.cluster.ID, createdNamespace.Name, cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)

	log.Info("Update the project with CPU request set greater than the CPU limit. Verify that the webhook rejects the request.")
	cpuLimit = "100m"
	cpuReservation = "200m"
	memoryLimit = ""
	memoryReservation = ""
	_, err = updateProjectContainerResourceLimit(standardUserClient, createdProject, cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.Error(pcrlv.T(), err)
	pattern := fmt.Sprintf(`admission webhook "rancher.cattle.io.projects.management.cattle.io" denied the request: project.spec.containerDefaultResourceLimit: Invalid value: v3.ContainerResourceLimit{RequestsCPU:"%s", RequestsMemory:"%s", LimitsCPU:"%s", LimitsMemory:"%s"}: requested CPU %s is greater than limit %s`, cpuReservation, memoryReservation, cpuLimit, memoryLimit, cpuReservation, cpuLimit)
	require.Regexp(pcrlv.T(), regexp.MustCompile(pattern), err.Error())
}

func (pcrlv *ProjectsContainerResourceLimitValidationTestSuite) TestUpdateProjectWithMemoryLimitLessThanRequest() {
	subSession := pcrlv.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Create a project (with valid container default resource limit) and a namespace in the project.")
	cpuLimit := "100m"
	cpuReservation := "50m"
	memoryLimit := "64Mi"
	memoryReservation := "32Mi"

	standardUser, err := users.CreateUserWithRole(pcrlv.client, users.UserConfig(), projects.StandardUser)
	require.NoError(pcrlv.T(), err, "Failed to create standard user")
	standardUserClient, err := pcrlv.client.AsUser(standardUser)
	require.NoError(pcrlv.T(), err)
	err = users.AddClusterRoleToUser(pcrlv.client, pcrlv.cluster, standardUser, clusterOwner, nil)
	require.NoError(pcrlv.T(), err, "Failed to add the user as a cluster owner to the downstream cluster")

	createdProject, createdNamespace, err := createProjectAndNamespace(standardUserClient, pcrlv.cluster.ID, "", "", cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)

	log.Info("Verify that the container default resource limit in the Project spec is accurate.")
	projectSpec := createdProject.Spec.ContainerDefaultResourceLimit
	require.Equal(pcrlv.T(), cpuLimit, projectSpec.LimitsCPU, "CPU limit mismatch")
	require.Equal(pcrlv.T(), cpuReservation, projectSpec.RequestsCPU, "CPU reservation mismatch")
	require.Equal(pcrlv.T(), memoryLimit, projectSpec.LimitsMemory, "Memory limit mismatch")
	require.Equal(pcrlv.T(), memoryReservation, projectSpec.RequestsMemory, "Memory reservation mismatch")

	log.Info("Verify that the limit range object is created for the namespace and the resource limit in the limit range is accurate.")
	err = checkLimitRange(standardUserClient, pcrlv.cluster.ID, createdNamespace.Name, cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.NoError(pcrlv.T(), err)

	log.Info("Update the project with Memory request set greater than Memory limit. Verify that the webhook rejects the request.")
	cpuLimit = ""
	cpuReservation = ""
	memoryLimit = "32Mi"
	memoryReservation = "64Mi"
	_, err = updateProjectContainerResourceLimit(standardUserClient, createdProject, cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	require.Error(pcrlv.T(), err)
	pattern := fmt.Sprintf(`admission webhook "rancher.cattle.io.projects.management.cattle.io" denied the request: project.spec.containerDefaultResourceLimit: Invalid value: v3.ContainerResourceLimit{RequestsCPU:"%s", RequestsMemory:"%s", LimitsCPU:"%s", LimitsMemory:"%s"}: requested memory %s is greater than limit %s`, cpuReservation, memoryReservation, cpuLimit, memoryLimit, memoryReservation, memoryLimit)
	require.Regexp(pcrlv.T(), regexp.MustCompile(pattern), err.Error())
}

func TestProjectsContainerResourceLimitValidationTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsContainerResourceLimitValidationTestSuite))
}
