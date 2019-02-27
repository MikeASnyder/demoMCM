package client

import (
	"github.com/rancher/norman/types"
)

const (
	PodType                               = "pod"
	PodFieldActiveDeadlineSeconds         = "activeDeadlineSeconds"
	PodFieldAnnotations                   = "annotations"
	PodFieldAutomountServiceAccountToken  = "automountServiceAccountToken"
	PodFieldContainers                    = "containers"
	PodFieldCreated                       = "created"
	PodFieldCreatorID                     = "creatorId"
	PodFieldDNSConfig                     = "dnsConfig"
	PodFieldDNSPolicy                     = "dnsPolicy"
	PodFieldDescription                   = "description"
	PodFieldFsgid                         = "fsgid"
	PodFieldGids                          = "gids"
	PodFieldHostAliases                   = "hostAliases"
	PodFieldHostIPC                       = "hostIPC"
	PodFieldHostNetwork                   = "hostNetwork"
	PodFieldHostPID                       = "hostPID"
	PodFieldHostname                      = "hostname"
	PodFieldImagePullSecrets              = "imagePullSecrets"
	PodFieldLabels                        = "labels"
	PodFieldName                          = "name"
	PodFieldNamespaceId                   = "namespaceId"
	PodFieldNodeID                        = "nodeId"
	PodFieldOwnerReferences               = "ownerReferences"
	PodFieldPriority                      = "priority"
	PodFieldPriorityClassName             = "priorityClassName"
	PodFieldProjectID                     = "projectId"
	PodFieldPublicEndpoints               = "publicEndpoints"
	PodFieldReadinessGates                = "readinessGates"
	PodFieldRemoved                       = "removed"
	PodFieldRestartPolicy                 = "restartPolicy"
	PodFieldRunAsGroup                    = "runAsGroup"
	PodFieldRunAsNonRoot                  = "runAsNonRoot"
	PodFieldRuntimeClassName              = "runtimeClassName"
	PodFieldSchedulerName                 = "schedulerName"
	PodFieldScheduling                    = "scheduling"
	PodFieldServiceAccountName            = "serviceAccountName"
	PodFieldShareProcessNamespace         = "shareProcessNamespace"
	PodFieldState                         = "state"
	PodFieldStatus                        = "status"
	PodFieldSubdomain                     = "subdomain"
	PodFieldSysctls                       = "sysctls"
	PodFieldTerminationGracePeriodSeconds = "terminationGracePeriodSeconds"
	PodFieldTransitioning                 = "transitioning"
	PodFieldTransitioningMessage          = "transitioningMessage"
	PodFieldUUID                          = "uuid"
	PodFieldUid                           = "uid"
	PodFieldVolumes                       = "volumes"
	PodFieldWorkloadID                    = "workloadId"
)

type Pod struct {
	types.Resource
	ActiveDeadlineSeconds         *int64                 `json:"activeDeadlineSeconds,omitempty" yaml:"activeDeadlineSeconds,omitempty"`
	Annotations                   map[string]string      `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	AutomountServiceAccountToken  *bool                  `json:"automountServiceAccountToken,omitempty" yaml:"automountServiceAccountToken,omitempty"`
	Containers                    []Container            `json:"containers,omitempty" yaml:"containers,omitempty"`
	Created                       string                 `json:"created,omitempty" yaml:"created,omitempty"`
	CreatorID                     string                 `json:"creatorId,omitempty" yaml:"creatorId,omitempty"`
	DNSConfig                     *PodDNSConfig          `json:"dnsConfig,omitempty" yaml:"dnsConfig,omitempty"`
	DNSPolicy                     string                 `json:"dnsPolicy,omitempty" yaml:"dnsPolicy,omitempty"`
	Description                   string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Fsgid                         *int64                 `json:"fsgid,omitempty" yaml:"fsgid,omitempty"`
	Gids                          []int64                `json:"gids,omitempty" yaml:"gids,omitempty"`
	HostAliases                   []HostAlias            `json:"hostAliases,omitempty" yaml:"hostAliases,omitempty"`
	HostIPC                       bool                   `json:"hostIPC,omitempty" yaml:"hostIPC,omitempty"`
	HostNetwork                   bool                   `json:"hostNetwork,omitempty" yaml:"hostNetwork,omitempty"`
	HostPID                       bool                   `json:"hostPID,omitempty" yaml:"hostPID,omitempty"`
	Hostname                      string                 `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	ImagePullSecrets              []LocalObjectReference `json:"imagePullSecrets,omitempty" yaml:"imagePullSecrets,omitempty"`
	Labels                        map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	Name                          string                 `json:"name,omitempty" yaml:"name,omitempty"`
	NamespaceId                   string                 `json:"namespaceId,omitempty" yaml:"namespaceId,omitempty"`
	NodeID                        string                 `json:"nodeId,omitempty" yaml:"nodeId,omitempty"`
	OwnerReferences               []OwnerReference       `json:"ownerReferences,omitempty" yaml:"ownerReferences,omitempty"`
	Priority                      *int64                 `json:"priority,omitempty" yaml:"priority,omitempty"`
	PriorityClassName             string                 `json:"priorityClassName,omitempty" yaml:"priorityClassName,omitempty"`
	ProjectID                     string                 `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	PublicEndpoints               []PublicEndpoint       `json:"publicEndpoints,omitempty" yaml:"publicEndpoints,omitempty"`
	ReadinessGates                []PodReadinessGate     `json:"readinessGates,omitempty" yaml:"readinessGates,omitempty"`
	Removed                       string                 `json:"removed,omitempty" yaml:"removed,omitempty"`
	RestartPolicy                 string                 `json:"restartPolicy,omitempty" yaml:"restartPolicy,omitempty"`
	RunAsGroup                    *int64                 `json:"runAsGroup,omitempty" yaml:"runAsGroup,omitempty"`
	RunAsNonRoot                  *bool                  `json:"runAsNonRoot,omitempty" yaml:"runAsNonRoot,omitempty"`
	RuntimeClassName              string                 `json:"runtimeClassName,omitempty" yaml:"runtimeClassName,omitempty"`
	SchedulerName                 string                 `json:"schedulerName,omitempty" yaml:"schedulerName,omitempty"`
	Scheduling                    *Scheduling            `json:"scheduling,omitempty" yaml:"scheduling,omitempty"`
	ServiceAccountName            string                 `json:"serviceAccountName,omitempty" yaml:"serviceAccountName,omitempty"`
	ShareProcessNamespace         *bool                  `json:"shareProcessNamespace,omitempty" yaml:"shareProcessNamespace,omitempty"`
	State                         string                 `json:"state,omitempty" yaml:"state,omitempty"`
	Status                        *PodStatus             `json:"status,omitempty" yaml:"status,omitempty"`
	Subdomain                     string                 `json:"subdomain,omitempty" yaml:"subdomain,omitempty"`
	Sysctls                       []Sysctl               `json:"sysctls,omitempty" yaml:"sysctls,omitempty"`
	TerminationGracePeriodSeconds *int64                 `json:"terminationGracePeriodSeconds,omitempty" yaml:"terminationGracePeriodSeconds,omitempty"`
	Transitioning                 string                 `json:"transitioning,omitempty" yaml:"transitioning,omitempty"`
	TransitioningMessage          string                 `json:"transitioningMessage,omitempty" yaml:"transitioningMessage,omitempty"`
	UUID                          string                 `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	Uid                           *int64                 `json:"uid,omitempty" yaml:"uid,omitempty"`
	Volumes                       []Volume               `json:"volumes,omitempty" yaml:"volumes,omitempty"`
	WorkloadID                    string                 `json:"workloadId,omitempty" yaml:"workloadId,omitempty"`
}

type PodCollection struct {
	types.Collection
	Data   []Pod `json:"data,omitempty"`
	client *PodClient
}

type PodClient struct {
	apiClient *Client
}

type PodOperations interface {
	List(opts *types.ListOpts) (*PodCollection, error)
	Create(opts *Pod) (*Pod, error)
	Update(existing *Pod, updates interface{}) (*Pod, error)
	Replace(existing *Pod) (*Pod, error)
	ByID(id string) (*Pod, error)
	Delete(container *Pod) error
}

func newPodClient(apiClient *Client) *PodClient {
	return &PodClient{
		apiClient: apiClient,
	}
}

func (c *PodClient) Create(container *Pod) (*Pod, error) {
	resp := &Pod{}
	err := c.apiClient.Ops.DoCreate(PodType, container, resp)
	return resp, err
}

func (c *PodClient) Update(existing *Pod, updates interface{}) (*Pod, error) {
	resp := &Pod{}
	err := c.apiClient.Ops.DoUpdate(PodType, &existing.Resource, updates, resp)
	return resp, err
}

func (c *PodClient) Replace(obj *Pod) (*Pod, error) {
	resp := &Pod{}
	err := c.apiClient.Ops.DoReplace(PodType, &obj.Resource, obj, resp)
	return resp, err
}

func (c *PodClient) List(opts *types.ListOpts) (*PodCollection, error) {
	resp := &PodCollection{}
	err := c.apiClient.Ops.DoList(PodType, opts, resp)
	resp.client = c
	return resp, err
}

func (cc *PodCollection) Next() (*PodCollection, error) {
	if cc != nil && cc.Pagination != nil && cc.Pagination.Next != "" {
		resp := &PodCollection{}
		err := cc.client.apiClient.Ops.DoNext(cc.Pagination.Next, resp)
		resp.client = cc.client
		return resp, err
	}
	return nil, nil
}

func (c *PodClient) ByID(id string) (*Pod, error) {
	resp := &Pod{}
	err := c.apiClient.Ops.DoByID(PodType, id, resp)
	return resp, err
}

func (c *PodClient) Delete(container *Pod) error {
	return c.apiClient.Ops.DoResourceDelete(PodType, &container.Resource)
}
