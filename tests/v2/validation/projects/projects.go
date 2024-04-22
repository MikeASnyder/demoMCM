package projects

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/defaults"
	"github.com/rancher/shepherd/extensions/kubeapi/limitranges"
	"github.com/rancher/shepherd/extensions/kubeapi/namespaces"
	"github.com/rancher/shepherd/extensions/kubeapi/projects"
	"github.com/rancher/shepherd/extensions/kubeapi/rbac"
	"github.com/rancher/shepherd/extensions/kubeapi/workloads/deployments"
	"github.com/rancher/shepherd/extensions/kubeapi/workloads/pods"
	"github.com/rancher/shepherd/extensions/kubeconfig"
	"github.com/rancher/shepherd/extensions/workloads"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kwait "k8s.io/apimachinery/pkg/util/wait"
)

const (
	dummyFinalizer = "dummy"
	timeFormat     = "2006/01/02 15:04:05"
	projectOwner   = "project-owner"
	clusterOwner   = "cluster-owner"
	containerName  = "nginx"
	imageName      = "nginx"
)

var resourceQuotaLimit = v3.ResourceQuotaLimit{
	Pods:                   "",
	Services:               "",
	ReplicationControllers: "",
	Secrets:                "",
	ConfigMaps:             "",
	PersistentVolumeClaims: "",
	ServicesNodePorts:      "",
	ServicesLoadBalancers:  "",
	RequestsCPU:            "",
	RequestsMemory:         "",
	RequestsStorage:        "",
	LimitsCPU:              "",
	LimitsMemory:           "",
}

var resourceQuota = &v3.ProjectResourceQuota{
	Limit: resourceQuotaLimit,
}

var namespaceResourceQuota = v3.NamespaceResourceQuota{
	Limit: resourceQuotaLimit,
}

var containerResourceLimit = v3.ContainerResourceLimit{
	RequestsCPU:    "",
	RequestsMemory: "",
	LimitsCPU:      "",
	LimitsMemory:   "",
}

var project = v3.Project{
	ObjectMeta: metav1.ObjectMeta{
		Name:       "",
		Namespace:  "",
		Finalizers: []string{},
	},
	Spec: v3.ProjectSpec{
		ClusterName:                   "",
		ResourceQuota:                 resourceQuota,
		NamespaceDefaultResourceQuota: &namespaceResourceQuota,
		ContainerDefaultResourceLimit: &containerResourceLimit,
	},
}

var prtb = v3.ProjectRoleTemplateBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "",
		Namespace: "",
	},
	ProjectName:       "",
	RoleTemplateName:  "",
	UserPrincipalName: "",
}

func createProject(client *rancher.Client, clusterID string, namespacePodLimit, projectPodLimit, cpuLimit, cpuReservation, memoryLimit, memoryReservation string) (*v3.Project, error) {
	project.Name = namegen.AppendRandomString("testproject")
	project.Namespace = clusterID
	project.Spec.ClusterName = clusterID
	project.Spec.NamespaceDefaultResourceQuota.Limit.Pods = namespacePodLimit
	project.Spec.ResourceQuota.Limit.Pods = projectPodLimit
	project.Spec.ContainerDefaultResourceLimit.LimitsCPU = cpuLimit
	project.Spec.ContainerDefaultResourceLimit.RequestsCPU = cpuReservation
	project.Spec.ContainerDefaultResourceLimit.LimitsMemory = memoryLimit
	project.Spec.ContainerDefaultResourceLimit.RequestsMemory = memoryReservation

	createdProject, err := projects.CreateProject(client, &project)
	if err != nil {
		return nil, err
	}

	return createdProject, nil
}

func createProjectAndNamespace(client *rancher.Client, clusterID string, namespacePodLimit, projectPodLimit, cpuLimit, cpuReservation, memoryLimit, memoryReservation string) (*v3.Project, *corev1.Namespace, error) {
	createdProject, err := createProject(client, clusterID, namespacePodLimit, projectPodLimit, cpuLimit, cpuReservation, memoryLimit, memoryReservation)
	if err != nil {
		return nil, nil, err
	}

	namespaceName := namegen.AppendRandomString("testns-")
	createdNamespace, err := namespaces.CreateNamespace(client, clusterID, createdProject.Name, namespaceName, "", map[string]string{}, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	return createdProject, createdNamespace, nil
}

func createProjectRoleTemplateBinding(client *rancher.Client, user *management.User, project *v3.Project, role string) (*v3.ProjectRoleTemplateBinding, error) {
	projectName := fmt.Sprintf("%s:%s", project.Namespace, project.Name)
	prtb.Name = namegen.AppendRandomString("prtb-")
	prtb.Namespace = project.Name
	prtb.ProjectName = projectName
	prtb.RoleTemplateName = role
	prtb.UserPrincipalName = user.PrincipalIDs[0]
	createdProjectRoleTemplateBinding, err := rbac.CreateProjectRoleTemplateBinding(client, &prtb)
	if err != nil {
		return nil, err
	}

	return createdProjectRoleTemplateBinding, nil
}

func createDeployment(client *rancher.Client, clusterID string, namespace string, replicaCount int) (*appv1.Deployment, error) {
	deploymentName := namegen.AppendRandomString("testdeployment")
	containerTemplate := workloads.NewContainer(containerName, imageName, corev1.PullAlways, []corev1.VolumeMount{}, []corev1.EnvFromSource{}, nil, nil, nil)
	podTemplate := workloads.NewPodTemplate([]corev1.Container{containerTemplate}, []corev1.Volume{}, []corev1.LocalObjectReference{}, nil)
	replicas := int32(replicaCount)

	deploymentObj, err := deployments.CreateDeployment(client, clusterID, deploymentName, namespace, podTemplate, replicas)
	if err != nil {
		return nil, err
	}

	return deploymentObj, nil
}

func updateProjectNamespaceFinalizer(client *rancher.Client, existingProject *v3.Project, finalizer []string) (*v3.Project, error) {
	updatedProject := existingProject.DeepCopy()
	updatedProject.ObjectMeta.Finalizers = finalizer

	updatedProject, err := projects.UpdateProject(client, existingProject, updatedProject)
	if err != nil {
		return nil, err
	}

	return updatedProject, nil
}

func updateProjectContainerResourceLimit(client *rancher.Client, existingProject *v3.Project, cpuLimit, cpuReservation, memoryLimit, memoryReservation string) (*v3.Project, error) {
	updatedProject := existingProject.DeepCopy()
	updatedProject.Spec.ContainerDefaultResourceLimit.LimitsCPU = cpuLimit
	updatedProject.Spec.ContainerDefaultResourceLimit.RequestsCPU = cpuReservation
	updatedProject.Spec.ContainerDefaultResourceLimit.LimitsMemory = memoryLimit
	updatedProject.Spec.ContainerDefaultResourceLimit.RequestsMemory = memoryReservation

	updatedProject, err := projects.UpdateProject(client, existingProject, updatedProject)
	if err != nil {
		return nil, err
	}

	return updatedProject, nil
}

func waitForFinalizerToUpdate(client *rancher.Client, projectName string, projectNamespace string, finalizerCount int) error {
	err := kwait.Poll(defaults.FiveHundredMillisecondTimeout, defaults.TenSecondTimeout, func() (done bool, pollErr error) {
		project, pollErr := projects.ListProjects(client, project.Namespace, metav1.ListOptions{
			FieldSelector: "metadata.name=" + project.Name,
		})
		if pollErr != nil {
			return false, pollErr
		}

		if len(project.Items[0].Finalizers) == finalizerCount {
			return true, nil
		}
		return false, pollErr
	})

	if err != nil {
		return err
	}

	return nil
}

func getPodByName(client *rancher.Client, clusterID, namespace, podName string) (*corev1.Pod, error) {
	updatedPodList, err := pods.ListPods(client, clusterID, namespace, metav1.ListOptions{
		FieldSelector: "metadata.name=" + podName,
	})
	if err != nil {
		return nil, err
	}

	if len(updatedPodList.Items) == 0 {
		return nil, fmt.Errorf("deployment %s not found", podName)
	}
	updatedPod := updatedPodList.Items[0]

	return &updatedPod, nil
}

func getPodNamesFromDeployment(client *rancher.Client, clusterID, namespaceName string, deploymentName string) ([]string, error) {
	deploymentList, err := deployments.ListDeployments(client, clusterID, namespaceName, metav1.ListOptions{
		FieldSelector: "metadata.name=" + deploymentName,
	})
	if err != nil {
		return nil, err
	}

	if len(deploymentList.Items) == 0 {
		return nil, fmt.Errorf("deployment %s not found", deploymentName)
	}
	deployment := deploymentList.Items[0]
	selector := deployment.Spec.Selector
	labelSelector, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, err
	}

	var podNames []string

	pods, err := pods.ListPods(client, clusterID, namespaceName, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return nil, err
	}

	for _, pod := range pods.Items {
		podNames = append(podNames, pod.Name)
	}

	return podNames, nil
}

func checkContainerResources(client *rancher.Client, clusterID, namespaceName, deploymentName, cpuLimit, cpuReservation, memoryLimit, memoryReservation string) error {
	var errs []string

	podNames, err := getPodNamesFromDeployment(client, clusterID, namespaceName, deploymentName)
	if err != nil {
		return fmt.Errorf("error fetching pod by deployment name: %w", err)
	}
	if len(podNames) < 1 {
		return errors.New("expected at least one pod, but got " + strconv.Itoa(len(podNames)))
	}

	pod, err := getPodByName(client, clusterID, namespaceName, podNames[0])
	if err != nil {
		return err
	}
	if len(pod.Spec.Containers) == 0 {
		return errors.New("no containers found in the pod")
	}

	normalizeString := func(s string) string {
		if s == "" {
			return "0"
		}
		return s
	}

	cpuLimit = normalizeString(cpuLimit)
	cpuReservation = normalizeString(cpuReservation)
	memoryLimit = normalizeString(memoryLimit)
	memoryReservation = normalizeString(memoryReservation)

	containerResources := pod.Spec.Containers[0].Resources
	containerCPULimit := containerResources.Limits[corev1.ResourceCPU]
	containerCPURequest := containerResources.Requests[corev1.ResourceCPU]
	containerMemoryLimit := containerResources.Limits[corev1.ResourceMemory]
	containerMemoryRequest := containerResources.Requests[corev1.ResourceMemory]

	if cpuLimit != containerCPULimit.String() {
		errs = append(errs, "CPU limit mismatch")
	}
	if cpuReservation != containerCPURequest.String() {
		errs = append(errs, "CPU reservation mismatch")
	}
	if memoryLimit != containerMemoryLimit.String() {
		errs = append(errs, "Memory limit mismatch")
	}
	if memoryReservation != containerMemoryRequest.String() {
		errs = append(errs, "Memory reservation mismatch")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func checkNamespaceLabelsAndAnnotations(clusterID string, projectName string, namespace *corev1.Namespace) error {
	var errorMessages []string
	expectedLabels := map[string]string{
		projects.ProjectIDAnnotation: projectName,
	}

	expectedAnnotations := map[string]string{
		projects.ProjectIDAnnotation: clusterID + ":" + projectName,
	}

	for key, value := range expectedLabels {
		if _, ok := namespace.Labels[key]; !ok {
			errorMessages = append(errorMessages, fmt.Sprintf("expected label %s not present in namespace labels", key))
		} else if namespace.Labels[key] != value {
			errorMessages = append(errorMessages, fmt.Sprintf("label value mismatch for %s: expected %s, got %s", key, value, namespace.Labels[key]))
		}
	}

	for key, value := range expectedAnnotations {
		if _, ok := namespace.Annotations[key]; !ok {
			errorMessages = append(errorMessages, fmt.Sprintf("expected annotation %s not present in namespace annotations", key))
		} else if namespace.Annotations[key] != value {
			errorMessages = append(errorMessages, fmt.Sprintf("annotation value mismatch for %s: expected %s, got %s", key, value, namespace.Annotations[key]))
		}
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf(strings.Join(errorMessages, "\n"))
	}

	return nil
}

func checkPodLogsForErrors(client *rancher.Client, clusterID string, podName string, namespace string, errorPattern string, startTime time.Time) error {
	startTimeUTC := startTime.UTC()

	errorRegex := regexp.MustCompile(errorPattern)
	timeRegex := regexp.MustCompile(`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`)

	var errorMessage string

	kwait.Poll(defaults.TenSecondTimeout, defaults.TwoMinuteTimeout, func() (bool, error) {
		podLogs, err := kubeconfig.GetPodLogs(client, clusterID, podName, namespace, "")
		if err != nil {
			return false, err
		}

		segments := strings.Split(podLogs, "\n")
		for _, segment := range segments {
			timeMatches := timeRegex.FindStringSubmatch(segment)
			if len(timeMatches) > 0 {
				segmentTime, err := time.Parse(timeFormat, timeMatches[0])
				if err != nil {
					continue
				}

				segmentTimeUTC := segmentTime.UTC()
				if segmentTimeUTC.After(startTimeUTC) {
					if matches := errorRegex.FindStringSubmatch(segment); len(matches) > 0 {
						errorMessage = "error logs found in rancher: " + segment
						return true, nil
					}
				}
			}
		}
		return false, nil
	})

	if errorMessage != "" {
		return errors.New(errorMessage)
	}

	return nil
}

func checkLimitRange(client *rancher.Client, clusterID, namespaceName string, expectedCPULimit, expectedCPURequest, expectedMemoryLimit, expectedMemoryRequest string) error {
	limitRanges, err := limitranges.ListLimitRange(client, clusterID, namespaceName, metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(limitRanges.Items) != 1 {
		return fmt.Errorf("expected limit range count is 1, but got %d", len(limitRanges.Items))
	}
	limitRangeList := limitRanges.Items[0].Spec

	actualCPULimit, ok := limitRangeList.Limits[0].Default["cpu"]
	if !ok {
		return fmt.Errorf("cpu limit not found in the limit range")
	}
	cpuLimit := actualCPULimit.String()
	if cpuLimit != expectedCPULimit {
		return fmt.Errorf("cpu limit in the limit range: %s does not match the expected value: %s", cpuLimit, expectedCPULimit)
	}

	actualMemoryLimit, ok := limitRangeList.Limits[0].Default["memory"]
	if !ok {
		return fmt.Errorf("memory limit not found in the limit range")
	}
	memoryLimit := actualMemoryLimit.String()
	if memoryLimit != expectedMemoryLimit {
		return fmt.Errorf("memory limit in the limit range: %s does not match the expected value: %s", memoryLimit, expectedMemoryLimit)
	}

	actualCPURequest, ok := limitRangeList.Limits[0].DefaultRequest["cpu"]
	if !ok {
		return fmt.Errorf("cpu request not found in the limit range")
	}
	cpuRequest := actualCPURequest.String()
	if cpuRequest != expectedCPURequest {
		return fmt.Errorf("cpu request in the limit range: %s does not match the expected value: %s", cpuRequest, expectedCPURequest)
	}

	actualMemoryRequest, ok := limitRangeList.Limits[0].DefaultRequest["memory"]
	if !ok {
		return fmt.Errorf("memory request not found in the limit range")
	}
	memoryRequest := actualMemoryRequest.String()
	if memoryRequest != expectedMemoryRequest {
		return fmt.Errorf("memory request in the limit range: %s does not match the expected value: %s", memoryRequest, expectedMemoryRequest)
	}

	return nil
}
