/*
Copyright 2022 Rancher Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by main. DO NOT EDIT.

package v3

import (
	"context"
	"time"

	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/condition"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/kv"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type ClusterHandler func(string, *v3.Cluster) (*v3.Cluster, error)

type ClusterController interface {
	generic.ControllerMeta
	ClusterClient

	OnChange(ctx context.Context, name string, sync ClusterHandler)
	OnRemove(ctx context.Context, name string, sync ClusterHandler)
	Enqueue(name string)
	EnqueueAfter(name string, duration time.Duration)

	Cache() ClusterCache
}

type ClusterClient interface {
	Create(*v3.Cluster) (*v3.Cluster, error)
	Update(*v3.Cluster) (*v3.Cluster, error)
	UpdateStatus(*v3.Cluster) (*v3.Cluster, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Get(name string, options metav1.GetOptions) (*v3.Cluster, error)
	List(opts metav1.ListOptions) (*v3.ClusterList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.Cluster, err error)
}

type ClusterCache interface {
	Get(name string) (*v3.Cluster, error)
	List(selector labels.Selector) ([]*v3.Cluster, error)

	AddIndexer(indexName string, indexer ClusterIndexer)
	GetByIndex(indexName, key string) ([]*v3.Cluster, error)
}

type ClusterIndexer func(obj *v3.Cluster) ([]string, error)

type clusterController struct {
	controller    controller.SharedController
	client        *client.Client
	gvk           schema.GroupVersionKind
	groupResource schema.GroupResource
}

func NewClusterController(gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) ClusterController {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &clusterController{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func FromClusterHandlerToHandler(sync ClusterHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v3.Cluster
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v3.Cluster))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *clusterController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v3.Cluster))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdateClusterDeepCopyOnChange(client ClusterClient, obj *v3.Cluster, handler func(obj *v3.Cluster) (*v3.Cluster, error)) (*v3.Cluster, error) {
	if obj == nil {
		return obj, nil
	}

	copyObj := obj.DeepCopy()
	newObj, err := handler(copyObj)
	if newObj != nil {
		copyObj = newObj
	}
	if obj.ResourceVersion == copyObj.ResourceVersion && !equality.Semantic.DeepEqual(obj, copyObj) {
		return client.Update(copyObj)
	}

	return copyObj, err
}

func (c *clusterController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *clusterController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), handler))
}

func (c *clusterController) OnChange(ctx context.Context, name string, sync ClusterHandler) {
	c.AddGenericHandler(ctx, name, FromClusterHandlerToHandler(sync))
}

func (c *clusterController) OnRemove(ctx context.Context, name string, sync ClusterHandler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), FromClusterHandlerToHandler(sync)))
}

func (c *clusterController) Enqueue(name string) {
	c.controller.Enqueue("", name)
}

func (c *clusterController) EnqueueAfter(name string, duration time.Duration) {
	c.controller.EnqueueAfter("", name, duration)
}

func (c *clusterController) Informer() cache.SharedIndexInformer {
	return c.controller.Informer()
}

func (c *clusterController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *clusterController) Cache() ClusterCache {
	return &clusterCache{
		indexer:  c.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}

func (c *clusterController) Create(obj *v3.Cluster) (*v3.Cluster, error) {
	result := &v3.Cluster{}
	return result, c.client.Create(context.TODO(), "", obj, result, metav1.CreateOptions{})
}

func (c *clusterController) Update(obj *v3.Cluster) (*v3.Cluster, error) {
	result := &v3.Cluster{}
	return result, c.client.Update(context.TODO(), "", obj, result, metav1.UpdateOptions{})
}

func (c *clusterController) UpdateStatus(obj *v3.Cluster) (*v3.Cluster, error) {
	result := &v3.Cluster{}
	return result, c.client.UpdateStatus(context.TODO(), "", obj, result, metav1.UpdateOptions{})
}

func (c *clusterController) Delete(name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), "", name, *options)
}

func (c *clusterController) Get(name string, options metav1.GetOptions) (*v3.Cluster, error) {
	result := &v3.Cluster{}
	return result, c.client.Get(context.TODO(), "", name, result, options)
}

func (c *clusterController) List(opts metav1.ListOptions) (*v3.ClusterList, error) {
	result := &v3.ClusterList{}
	return result, c.client.List(context.TODO(), "", result, opts)
}

func (c *clusterController) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), "", opts)
}

func (c *clusterController) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*v3.Cluster, error) {
	result := &v3.Cluster{}
	return result, c.client.Patch(context.TODO(), "", name, pt, data, result, metav1.PatchOptions{}, subresources...)
}

type clusterCache struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *clusterCache) Get(name string) (*v3.Cluster, error) {
	obj, exists, err := c.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(*v3.Cluster), nil
}

func (c *clusterCache) List(selector labels.Selector) (ret []*v3.Cluster, err error) {

	err = cache.ListAll(c.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.Cluster))
	})

	return ret, err
}

func (c *clusterCache) AddIndexer(indexName string, indexer ClusterIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v3.Cluster))
		},
	}))
}

func (c *clusterCache) GetByIndex(indexName, key string) (result []*v3.Cluster, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*v3.Cluster, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*v3.Cluster))
	}
	return result, nil
}

type ClusterStatusHandler func(obj *v3.Cluster, status v3.ClusterStatus) (v3.ClusterStatus, error)

type ClusterGeneratingHandler func(obj *v3.Cluster, status v3.ClusterStatus) ([]runtime.Object, v3.ClusterStatus, error)

func RegisterClusterStatusHandler(ctx context.Context, controller ClusterController, condition condition.Cond, name string, handler ClusterStatusHandler) {
	statusHandler := &clusterStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, FromClusterHandlerToHandler(statusHandler.sync))
}

func RegisterClusterGeneratingHandler(ctx context.Context, controller ClusterController, apply apply.Apply,
	condition condition.Cond, name string, handler ClusterGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &clusterGeneratingHandler{
		ClusterGeneratingHandler: handler,
		apply:                    apply,
		name:                     name,
		gvk:                      controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterClusterStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type clusterStatusHandler struct {
	client    ClusterClient
	condition condition.Cond
	handler   ClusterStatusHandler
}

func (a *clusterStatusHandler) sync(key string, obj *v3.Cluster) (*v3.Cluster, error) {
	if obj == nil {
		return obj, nil
	}

	origStatus := obj.Status.DeepCopy()
	obj = obj.DeepCopy()
	newStatus, err := a.handler(obj, obj.Status)
	if err != nil {
		// Revert to old status on error
		newStatus = *origStatus.DeepCopy()
	}

	if a.condition != "" {
		if errors.IsConflict(err) {
			a.condition.SetError(&newStatus, "", nil)
		} else {
			a.condition.SetError(&newStatus, "", err)
		}
	}
	if !equality.Semantic.DeepEqual(origStatus, &newStatus) {
		if a.condition != "" {
			// Since status has changed, update the lastUpdatedTime
			a.condition.LastUpdated(&newStatus, time.Now().UTC().Format(time.RFC3339))
		}

		var newErr error
		obj.Status = newStatus
		newObj, newErr := a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
		if newErr == nil {
			obj = newObj
		}
	}
	return obj, err
}

type clusterGeneratingHandler struct {
	ClusterGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
}

func (a *clusterGeneratingHandler) Remove(key string, obj *v3.Cluster) (*v3.Cluster, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v3.Cluster{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *clusterGeneratingHandler) Handle(obj *v3.Cluster, status v3.ClusterStatus) (v3.ClusterStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.ClusterGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}

	return newStatus, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
}
