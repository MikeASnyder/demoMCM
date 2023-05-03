// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rancher/rancher/pkg/generated/controllers/provisioning.cattle.io/v1 (interfaces: ClusterCache)

// Package mockranchercontrollers is a generated GoMock package.
package mockranchercontrollers

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	v10 "github.com/rancher/rancher/pkg/generated/controllers/provisioning.cattle.io/v1"
	labels "k8s.io/apimachinery/pkg/labels"
)

// MockClusterCache is a mock of ClusterCache interface.
type MockClusterCache struct {
	ctrl     *gomock.Controller
	recorder *MockClusterCacheMockRecorder
}

// MockClusterCacheMockRecorder is the mock recorder for MockClusterCache.
type MockClusterCacheMockRecorder struct {
	mock *MockClusterCache
}

// NewMockClusterCache creates a new mock instance.
func NewMockClusterCache(ctrl *gomock.Controller) *MockClusterCache {
	mock := &MockClusterCache{ctrl: ctrl}
	mock.recorder = &MockClusterCacheMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClusterCache) EXPECT() *MockClusterCacheMockRecorder {
	return m.recorder
}

// AddIndexer mocks base method.
func (m *MockClusterCache) AddIndexer(arg0 string, arg1 v10.ClusterIndexer) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddIndexer", arg0, arg1)
}

// AddIndexer indicates an expected call of AddIndexer.
func (mr *MockClusterCacheMockRecorder) AddIndexer(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddIndexer", reflect.TypeOf((*MockClusterCache)(nil).AddIndexer), arg0, arg1)
}

// Get mocks base method.
func (m *MockClusterCache) Get(arg0, arg1 string) (*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockClusterCacheMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockClusterCache)(nil).Get), arg0, arg1)
}

// GetByIndex mocks base method.
func (m *MockClusterCache) GetByIndex(arg0, arg1 string) ([]*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByIndex", arg0, arg1)
	ret0, _ := ret[0].([]*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByIndex indicates an expected call of GetByIndex.
func (mr *MockClusterCacheMockRecorder) GetByIndex(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByIndex", reflect.TypeOf((*MockClusterCache)(nil).GetByIndex), arg0, arg1)
}

// List mocks base method.
func (m *MockClusterCache) List(arg0 string, arg1 labels.Selector) ([]*v1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0, arg1)
	ret0, _ := ret[0].([]*v1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockClusterCacheMockRecorder) List(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockClusterCache)(nil).List), arg0, arg1)
}
