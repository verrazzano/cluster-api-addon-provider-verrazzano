/*
Copyright The Kubernetes Authors.

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

// This file from the cluster-api community (https://github.com/kubernetes-sigs/cluster-api) has been modified by Oracle.
// Code generated by MockGen. DO NOT EDIT.
// Source: ../kubeconfig.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	internal "github.com/verrazzano/cluster-api-addon-provider-verrazzano/internal"
	gomock "go.uber.org/mock/gomock"
	dynamic "k8s.io/client-go/dynamic"
	kubernetes "k8s.io/client-go/kubernetes"
	v1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// MockGetter is a mock of Getter interface.
type MockGetter struct {
	ctrl     *gomock.Controller
	recorder *MockGetterMockRecorder
}

// MockGetterMockRecorder is the mock recorder for MockGetter.
type MockGetterMockRecorder struct {
	mock *MockGetter
}

// NewMockGetter creates a new mock instance.
func NewMockGetter(ctrl *gomock.Controller) *MockGetter {
	mock := &MockGetter{ctrl: ctrl}
	mock.recorder = &MockGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGetter) EXPECT() *MockGetterMockRecorder {
	return m.recorder
}

// CreateOrUpdateVerrazzano mocks base method.
func (m *MockGetter) CreateOrUpdateVerrazzano(ctx context.Context, fleetBindingName, kubeconfig, clusterName, vzspec string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOrUpdateVerrazzano", ctx, fleetBindingName, kubeconfig, clusterName, vzspec)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateOrUpdateVerrazzano indicates an expected call of CreateOrUpdateVerrazzano.
func (mr *MockGetterMockRecorder) CreateOrUpdateVerrazzano(ctx, fleetBindingName, kubeconfig, clusterName, vzspec interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOrUpdateVerrazzano", reflect.TypeOf((*MockGetter)(nil).CreateOrUpdateVerrazzano), ctx, fleetBindingName, kubeconfig, clusterName, vzspec)
}

// DeleteVerrazzano mocks base method.
func (m *MockGetter) DeleteVerrazzano(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVerrazzano", ctx, fleetBindingName, kubeconfig, clusterName)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVerrazzano indicates an expected call of DeleteVerrazzano.
func (mr *MockGetterMockRecorder) DeleteVerrazzano(ctx, fleetBindingName, kubeconfig, clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVerrazzano", reflect.TypeOf((*MockGetter)(nil).DeleteVerrazzano), ctx, fleetBindingName, kubeconfig, clusterName)
}

// GetClusterKubeconfig mocks base method.
func (m *MockGetter) GetClusterKubeconfig(ctx context.Context, cluster *v1beta1.Cluster) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClusterKubeconfig", ctx, cluster)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClusterKubeconfig indicates an expected call of GetClusterKubeconfig.
func (mr *MockGetterMockRecorder) GetClusterKubeconfig(ctx, cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClusterKubeconfig", reflect.TypeOf((*MockGetter)(nil).GetClusterKubeconfig), ctx, cluster)
}

// GetVerrazzano mocks base method.
func (m *MockGetter) GetVerrazzano(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) (*internal.Verrazzano, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVerrazzano", ctx, fleetBindingName, kubeconfig, clusterName)
	ret0, _ := ret[0].(*internal.Verrazzano)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVerrazzano indicates an expected call of GetVerrazzano.
func (mr *MockGetterMockRecorder) GetVerrazzano(ctx, fleetBindingName, kubeconfig, clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVerrazzano", reflect.TypeOf((*MockGetter)(nil).GetVerrazzano), ctx, fleetBindingName, kubeconfig, clusterName)
}

// GetWorkloadClusterDynamicK8sClient mocks base method.
func (m *MockGetter) GetWorkloadClusterDynamicK8sClient(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) (dynamic.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorkloadClusterDynamicK8sClient", ctx, fleetBindingName, kubeconfig, clusterName)
	ret0, _ := ret[0].(dynamic.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWorkloadClusterDynamicK8sClient indicates an expected call of GetWorkloadClusterDynamicK8sClient.
func (mr *MockGetterMockRecorder) GetWorkloadClusterDynamicK8sClient(ctx, fleetBindingName, kubeconfig, clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorkloadClusterDynamicK8sClient", reflect.TypeOf((*MockGetter)(nil).GetWorkloadClusterDynamicK8sClient), ctx, fleetBindingName, kubeconfig, clusterName)
}

// GetWorkloadClusterK8sClient mocks base method.
func (m *MockGetter) GetWorkloadClusterK8sClient(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) (*kubernetes.Clientset, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorkloadClusterK8sClient", ctx, fleetBindingName, kubeconfig, clusterName)
	ret0, _ := ret[0].(*kubernetes.Clientset)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWorkloadClusterK8sClient indicates an expected call of GetWorkloadClusterK8sClient.
func (mr *MockGetterMockRecorder) GetWorkloadClusterK8sClient(ctx, fleetBindingName, kubeconfig, clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorkloadClusterK8sClient", reflect.TypeOf((*MockGetter)(nil).GetWorkloadClusterK8sClient), ctx, fleetBindingName, kubeconfig, clusterName)
}

// WaitForVerrazzanoUninstallCompletion mocks base method.
func (m *MockGetter) WaitForVerrazzanoUninstallCompletion(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForVerrazzanoUninstallCompletion", ctx, fleetBindingName, kubeconfig, clusterName)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitForVerrazzanoUninstallCompletion indicates an expected call of WaitForVerrazzanoUninstallCompletion.
func (mr *MockGetterMockRecorder) WaitForVerrazzanoUninstallCompletion(ctx, fleetBindingName, kubeconfig, clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForVerrazzanoUninstallCompletion", reflect.TypeOf((*MockGetter)(nil).WaitForVerrazzanoUninstallCompletion), ctx, fleetBindingName, kubeconfig, clusterName)
}
