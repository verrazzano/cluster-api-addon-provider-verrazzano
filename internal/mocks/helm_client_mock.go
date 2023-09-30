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
// Source: ../helm_client.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	v1alpha1 "github.com/verrazzano/cluster-api-addon-provider-verrazzano/api/v1alpha1"
	models "github.com/verrazzano/cluster-api-addon-provider-verrazzano/models"
	gomock "go.uber.org/mock/gomock"
	release "helm.sh/helm/v3/pkg/release"
	kubernetes "k8s.io/client-go/kubernetes"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// GetHelmRelease mocks base method.
func (m *MockClient) GetHelmRelease(ctx context.Context, kubeconfig string, spec *models.HelmModuleAddons) (*release.Release, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHelmRelease", ctx, kubeconfig, spec)
	ret0, _ := ret[0].(*release.Release)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHelmRelease indicates an expected call of GetHelmRelease.
func (mr *MockClientMockRecorder) GetHelmRelease(ctx, kubeconfig, spec interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHelmRelease", reflect.TypeOf((*MockClient)(nil).GetHelmRelease), ctx, kubeconfig, spec)
}

// GetWorkloadClusterK8sClient mocks base method.
func (m *MockClient) GetWorkloadClusterK8sClient(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) (*kubernetes.Clientset, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorkloadClusterK8sClient", ctx, fleetBindingName, kubeconfig, clusterName)
	ret0, _ := ret[0].(*kubernetes.Clientset)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWorkloadClusterK8sClient indicates an expected call of GetWorkloadClusterK8sClient.
func (mr *MockClientMockRecorder) GetWorkloadClusterK8sClient(ctx, fleetBindingName, kubeconfig, clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorkloadClusterK8sClient", reflect.TypeOf((*MockClient)(nil).GetWorkloadClusterK8sClient), ctx, fleetBindingName, kubeconfig, clusterName)
}

// InstallOrUpgradeHelmRelease mocks base method.
func (m *MockClient) InstallOrUpgradeHelmRelease(ctx context.Context, kubeconfig, values string, spec *models.HelmModuleAddons, verrazzanoFleetBinding *v1alpha1.VerrazzanoFleetBinding) (*release.Release, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstallOrUpgradeHelmRelease", ctx, kubeconfig, values, spec, verrazzanoFleetBinding)
	ret0, _ := ret[0].(*release.Release)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InstallOrUpgradeHelmRelease indicates an expected call of InstallOrUpgradeHelmRelease.
func (mr *MockClientMockRecorder) InstallOrUpgradeHelmRelease(ctx, kubeconfig, values, spec, verrazzanoFleetBinding interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstallOrUpgradeHelmRelease", reflect.TypeOf((*MockClient)(nil).InstallOrUpgradeHelmRelease), ctx, kubeconfig, values, spec, verrazzanoFleetBinding)
}

// UninstallHelmRelease mocks base method.
func (m *MockClient) UninstallHelmRelease(ctx context.Context, kubeconfig string, spec *models.HelmModuleAddons) (*release.UninstallReleaseResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UninstallHelmRelease", ctx, kubeconfig, spec)
	ret0, _ := ret[0].(*release.UninstallReleaseResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UninstallHelmRelease indicates an expected call of UninstallHelmRelease.
func (mr *MockClientMockRecorder) UninstallHelmRelease(ctx, kubeconfig, spec interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UninstallHelmRelease", reflect.TypeOf((*MockClient)(nil).UninstallHelmRelease), ctx, kubeconfig, spec)
}
