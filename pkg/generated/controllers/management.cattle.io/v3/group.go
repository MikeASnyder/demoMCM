/*
Copyright 2024 Rancher Labs, Inc.

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
	"github.com/rancher/wrangler/pkg/generic"
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

type GroupHandler func(string, *v3.Group) (*v3.Group, error)

type GroupController interface {
	generic.ControllerMeta
	GroupClient

	OnChange(ctx context.Context, name string, sync GroupHandler)
	OnRemove(ctx context.Context, name string, sync GroupHandler)
	Enqueue(name string)
	EnqueueAfter(name string, duration time.Duration)

	Cache() GroupCache
}

type GroupClient interface {
	Create(*v3.Group) (*v3.Group, error)
	Update(*v3.Group) (*v3.Group, error)

	Delete(name string, options *metav1.DeleteOptions) error
	Get(name string, options metav1.GetOptions) (*v3.Group, error)
	List(opts metav1.ListOptions) (*v3.GroupList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.Group, err error)
}

type GroupCache interface {
	Get(name string) (*v3.Group, error)
	List(selector labels.Selector) ([]*v3.Group, error)

	AddIndexer(indexName string, indexer GroupIndexer)
	GetByIndex(indexName, key string) ([]*v3.Group, error)
}

type GroupIndexer func(obj *v3.Group) ([]string, error)

type groupController struct {
	controller    controller.SharedController
	client        *client.Client
	gvk           schema.GroupVersionKind
	groupResource schema.GroupResource
}

func NewGroupController(gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) GroupController {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &groupController{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func FromGroupHandlerToHandler(sync GroupHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v3.Group
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v3.Group))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *groupController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v3.Group))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdateGroupDeepCopyOnChange(client GroupClient, obj *v3.Group, handler func(obj *v3.Group) (*v3.Group, error)) (*v3.Group, error) {
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

func (c *groupController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *groupController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), handler))
}

func (c *groupController) OnChange(ctx context.Context, name string, sync GroupHandler) {
	c.AddGenericHandler(ctx, name, FromGroupHandlerToHandler(sync))
}

func (c *groupController) OnRemove(ctx context.Context, name string, sync GroupHandler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), FromGroupHandlerToHandler(sync)))
}

func (c *groupController) Enqueue(name string) {
	c.controller.Enqueue("", name)
}

func (c *groupController) EnqueueAfter(name string, duration time.Duration) {
	c.controller.EnqueueAfter("", name, duration)
}

func (c *groupController) Informer() cache.SharedIndexInformer {
	return c.controller.Informer()
}

func (c *groupController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *groupController) Cache() GroupCache {
	return &groupCache{
		indexer:  c.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}

func (c *groupController) Create(obj *v3.Group) (*v3.Group, error) {
	result := &v3.Group{}
	return result, c.client.Create(context.TODO(), "", obj, result, metav1.CreateOptions{})
}

func (c *groupController) Update(obj *v3.Group) (*v3.Group, error) {
	result := &v3.Group{}
	return result, c.client.Update(context.TODO(), "", obj, result, metav1.UpdateOptions{})
}

func (c *groupController) Delete(name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), "", name, *options)
}

func (c *groupController) Get(name string, options metav1.GetOptions) (*v3.Group, error) {
	result := &v3.Group{}
	return result, c.client.Get(context.TODO(), "", name, result, options)
}

func (c *groupController) List(opts metav1.ListOptions) (*v3.GroupList, error) {
	result := &v3.GroupList{}
	return result, c.client.List(context.TODO(), "", result, opts)
}

func (c *groupController) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), "", opts)
}

func (c *groupController) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*v3.Group, error) {
	result := &v3.Group{}
	return result, c.client.Patch(context.TODO(), "", name, pt, data, result, metav1.PatchOptions{}, subresources...)
}

type groupCache struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *groupCache) Get(name string) (*v3.Group, error) {
	obj, exists, err := c.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(*v3.Group), nil
}

func (c *groupCache) List(selector labels.Selector) (ret []*v3.Group, err error) {

	err = cache.ListAll(c.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.Group))
	})

	return ret, err
}

func (c *groupCache) AddIndexer(indexName string, indexer GroupIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v3.Group))
		},
	}))
}

func (c *groupCache) GetByIndex(indexName, key string) (result []*v3.Group, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*v3.Group, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*v3.Group))
	}
	return result, nil
}
