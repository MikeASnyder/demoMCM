/*
Copyright 2023 Rancher Labs, Inc.

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
	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/wrangler/pkg/generic"
)

// RkeK8sSystemImageController interface for managing RkeK8sSystemImage resources.
type RkeK8sSystemImageController interface {
	generic.ControllerInterface[*v3.RkeK8sSystemImage, *v3.RkeK8sSystemImageList]
}

// RkeK8sSystemImageClient interface for managing RkeK8sSystemImage resources in Kubernetes.
type RkeK8sSystemImageClient interface {
	generic.ClientInterface[*v3.RkeK8sSystemImage, *v3.RkeK8sSystemImageList]
}

// RkeK8sSystemImageCache interface for retrieving RkeK8sSystemImage resources in memory.
type RkeK8sSystemImageCache interface {
	generic.CacheInterface[*v3.RkeK8sSystemImage]
}
