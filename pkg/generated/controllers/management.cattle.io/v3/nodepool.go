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

	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/wrangler/v2/pkg/apply"
	"github.com/rancher/wrangler/v2/pkg/condition"
	"github.com/rancher/wrangler/v2/pkg/generic"
	"github.com/rancher/wrangler/v2/pkg/kv"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NodePoolController interface for managing NodePool resources.
type NodePoolController interface {
	generic.ControllerInterface[*v3.NodePool, *v3.NodePoolList]
}

// NodePoolClient interface for managing NodePool resources in Kubernetes.
type NodePoolClient interface {
	generic.ClientInterface[*v3.NodePool, *v3.NodePoolList]
}

// NodePoolCache interface for retrieving NodePool resources in memory.
type NodePoolCache interface {
	generic.CacheInterface[*v3.NodePool]
}

type NodePoolStatusHandler func(obj *v3.NodePool, status v3.NodePoolStatus) (v3.NodePoolStatus, error)

type NodePoolGeneratingHandler func(obj *v3.NodePool, status v3.NodePoolStatus) ([]runtime.Object, v3.NodePoolStatus, error)

func RegisterNodePoolStatusHandler(ctx context.Context, controller NodePoolController, condition condition.Cond, name string, handler NodePoolStatusHandler) {
	statusHandler := &nodePoolStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, generic.FromObjectHandlerToHandler(statusHandler.sync))
}

func RegisterNodePoolGeneratingHandler(ctx context.Context, controller NodePoolController, apply apply.Apply,
	condition condition.Cond, name string, handler NodePoolGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &nodePoolGeneratingHandler{
		NodePoolGeneratingHandler: handler,
		apply:                     apply,
		name:                      name,
		gvk:                       controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterNodePoolStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type nodePoolStatusHandler struct {
	client    NodePoolClient
	condition condition.Cond
	handler   NodePoolStatusHandler
}

func (a *nodePoolStatusHandler) sync(key string, obj *v3.NodePool) (*v3.NodePool, error) {
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

type nodePoolGeneratingHandler struct {
	NodePoolGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
}

func (a *nodePoolGeneratingHandler) Remove(key string, obj *v3.NodePool) (*v3.NodePool, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v3.NodePool{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *nodePoolGeneratingHandler) Handle(obj *v3.NodePool, status v3.NodePoolStatus) (v3.NodePoolStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.NodePoolGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}

	return newStatus, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
}
