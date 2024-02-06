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

type FreeIpaProviderHandler func(string, *v3.FreeIpaProvider) (*v3.FreeIpaProvider, error)

type FreeIpaProviderController interface {
	generic.ControllerMeta
	FreeIpaProviderClient

	OnChange(ctx context.Context, name string, sync FreeIpaProviderHandler)
	OnRemove(ctx context.Context, name string, sync FreeIpaProviderHandler)
	Enqueue(name string)
	EnqueueAfter(name string, duration time.Duration)

	Cache() FreeIpaProviderCache
}

type FreeIpaProviderClient interface {
	Create(*v3.FreeIpaProvider) (*v3.FreeIpaProvider, error)
	Update(*v3.FreeIpaProvider) (*v3.FreeIpaProvider, error)

	Delete(name string, options *metav1.DeleteOptions) error
	Get(name string, options metav1.GetOptions) (*v3.FreeIpaProvider, error)
	List(opts metav1.ListOptions) (*v3.FreeIpaProviderList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.FreeIpaProvider, err error)
}

type FreeIpaProviderCache interface {
	Get(name string) (*v3.FreeIpaProvider, error)
	List(selector labels.Selector) ([]*v3.FreeIpaProvider, error)

	AddIndexer(indexName string, indexer FreeIpaProviderIndexer)
	GetByIndex(indexName, key string) ([]*v3.FreeIpaProvider, error)
}

type FreeIpaProviderIndexer func(obj *v3.FreeIpaProvider) ([]string, error)

type freeIpaProviderController struct {
	controller    controller.SharedController
	client        *client.Client
	gvk           schema.GroupVersionKind
	groupResource schema.GroupResource
}

func NewFreeIpaProviderController(gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) FreeIpaProviderController {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &freeIpaProviderController{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func FromFreeIpaProviderHandlerToHandler(sync FreeIpaProviderHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v3.FreeIpaProvider
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v3.FreeIpaProvider))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *freeIpaProviderController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v3.FreeIpaProvider))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdateFreeIpaProviderDeepCopyOnChange(client FreeIpaProviderClient, obj *v3.FreeIpaProvider, handler func(obj *v3.FreeIpaProvider) (*v3.FreeIpaProvider, error)) (*v3.FreeIpaProvider, error) {
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

func (c *freeIpaProviderController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *freeIpaProviderController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), handler))
}

func (c *freeIpaProviderController) OnChange(ctx context.Context, name string, sync FreeIpaProviderHandler) {
	c.AddGenericHandler(ctx, name, FromFreeIpaProviderHandlerToHandler(sync))
}

func (c *freeIpaProviderController) OnRemove(ctx context.Context, name string, sync FreeIpaProviderHandler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), FromFreeIpaProviderHandlerToHandler(sync)))
}

func (c *freeIpaProviderController) Enqueue(name string) {
	c.controller.Enqueue("", name)
}

func (c *freeIpaProviderController) EnqueueAfter(name string, duration time.Duration) {
	c.controller.EnqueueAfter("", name, duration)
}

func (c *freeIpaProviderController) Informer() cache.SharedIndexInformer {
	return c.controller.Informer()
}

func (c *freeIpaProviderController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *freeIpaProviderController) Cache() FreeIpaProviderCache {
	return &freeIpaProviderCache{
		indexer:  c.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}

func (c *freeIpaProviderController) Create(obj *v3.FreeIpaProvider) (*v3.FreeIpaProvider, error) {
	result := &v3.FreeIpaProvider{}
	return result, c.client.Create(context.TODO(), "", obj, result, metav1.CreateOptions{})
}

func (c *freeIpaProviderController) Update(obj *v3.FreeIpaProvider) (*v3.FreeIpaProvider, error) {
	result := &v3.FreeIpaProvider{}
	return result, c.client.Update(context.TODO(), "", obj, result, metav1.UpdateOptions{})
}

func (c *freeIpaProviderController) Delete(name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), "", name, *options)
}

func (c *freeIpaProviderController) Get(name string, options metav1.GetOptions) (*v3.FreeIpaProvider, error) {
	result := &v3.FreeIpaProvider{}
	return result, c.client.Get(context.TODO(), "", name, result, options)
}

func (c *freeIpaProviderController) List(opts metav1.ListOptions) (*v3.FreeIpaProviderList, error) {
	result := &v3.FreeIpaProviderList{}
	return result, c.client.List(context.TODO(), "", result, opts)
}

func (c *freeIpaProviderController) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), "", opts)
}

func (c *freeIpaProviderController) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*v3.FreeIpaProvider, error) {
	result := &v3.FreeIpaProvider{}
	return result, c.client.Patch(context.TODO(), "", name, pt, data, result, metav1.PatchOptions{}, subresources...)
}

type freeIpaProviderCache struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *freeIpaProviderCache) Get(name string) (*v3.FreeIpaProvider, error) {
	obj, exists, err := c.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(*v3.FreeIpaProvider), nil
}

func (c *freeIpaProviderCache) List(selector labels.Selector) (ret []*v3.FreeIpaProvider, err error) {

	err = cache.ListAll(c.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.FreeIpaProvider))
	})

	return ret, err
}

func (c *freeIpaProviderCache) AddIndexer(indexName string, indexer FreeIpaProviderIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v3.FreeIpaProvider))
		},
	}))
}

func (c *freeIpaProviderCache) GetByIndex(indexName, key string) (result []*v3.FreeIpaProvider, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*v3.FreeIpaProvider, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*v3.FreeIpaProvider))
	}
	return result, nil
}
