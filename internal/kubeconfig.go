/*
Copyright 2022 The Kubernetes Authors.

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

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"os"
	"path/filepath"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	ctrl "sigs.k8s.io/controller-runtime"
	configclient "sigs.k8s.io/controller-runtime/pkg/client/config"
	"time"
)

type Getter interface {
	GetClusterKubeconfig(ctx context.Context, cluster *clusterv1.Cluster) (string, error)
	GetWorkloadClusterK8sClient(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) (*kubernetes.Clientset, error)
	GetWorkloadClusterDynamicK8sClient(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) (dynamic.Interface, error)
	CreateOrUpdateVerrazzano(ctx context.Context, fleetBindingName, kubeconfig, clusterName, vzspec string) error
	GetVerrazzano(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) (*Verrazzano, error)
	DeleteVerrazzano(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) error
	WaitForVerrazzanoUninstallCompletion(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) error
}

type KubeconfigGetter struct{}

// GetClusterKubeconfig returns the kubeconfig for a selected Cluster as a string.
func (k *KubeconfigGetter) GetClusterKubeconfig(ctx context.Context, cluster *clusterv1.Cluster) (string, error) {
	log := ctrl.LoggerFrom(ctx)

	log.V(2).Info("Initializing management cluster kubeconfig")
	managementKubeconfig, err := initInClusterKubeconfig(ctx)
	if err != nil {
		return "", errors.Wrapf(err, "failed to initialize management cluster kubeconfig")
	}

	c, err := client.New("")
	if err != nil {
		return "", err
	}

	options := client.GetKubeconfigOptions{
		Kubeconfig:          client.Kubeconfig(*managementKubeconfig),
		WorkloadClusterName: cluster.Name,
		Namespace:           cluster.Namespace,
	}

	log.V(4).Info("Getting kubeconfig for cluster", "cluster", cluster.Name)
	kubeconfig, err := c.GetKubeconfig(options)
	if err != nil {
		return "", err
	}

	return kubeconfig, nil
}

// initInClusterKubeconfig generates a kubeconfig file for the management cluster.
// Note: The k8s.io/client-go/tools/clientcmd/api package and associated tools require a path to a kubeconfig file rather than the data stored in an object.
func initInClusterKubeconfig(ctx context.Context) (*cluster.Kubeconfig, error) {
	log := ctrl.LoggerFrom(ctx)

	log.V(2).Info("Generating kubeconfig file")
	restConfig := configclient.GetConfigOrDie()

	apiConfig, err := constructInClusterKubeconfig(ctx, restConfig, "")
	if err != nil {
		log.Error(err, "error constructing in-cluster kubeconfig")
		return nil, err
	}
	filePath := "tmp/management.kubeconfig"
	if err = writeInClusterKubeconfigToFile(ctx, filePath, *apiConfig); err != nil {
		log.Error(err, "error writing kubeconfig to file")
		return nil, err
	}
	kubeconfigPath := filePath
	kubeContext := apiConfig.CurrentContext

	return &cluster.Kubeconfig{Path: kubeconfigPath, Context: kubeContext}, nil
}

// GetClusterKubeconfig generates a kubeconfig file for the management cluster using a rest.Config. This is a bit of a workaround
// since the k8s.io/client-go/tools/clientcmd/api expects to be run from a CLI context, but within a pod we don't have that.
// As a result, we have to manually fill in the fields that would normally be present in ~/.kube/config. This seems to work for now.
func constructInClusterKubeconfig(ctx context.Context, restConfig *rest.Config, namespace string) (*clientcmdapi.Config, error) {
	log := ctrl.LoggerFrom(ctx)

	log.V(2).Info("Constructing kubeconfig file from rest.Config")

	clusterName := "management-cluster"
	userName := "default-user"
	contextName := "default-context"
	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters[clusterName] = &clientcmdapi.Cluster{
		Server: restConfig.Host,
		// Used in regular kubeconfigs.
		CertificateAuthorityData: restConfig.CAData,
		// Used in in-cluster configs.
		CertificateAuthority: restConfig.CAFile,
	}

	contexts := make(map[string]*clientcmdapi.Context)
	contexts[contextName] = &clientcmdapi.Context{
		Cluster:   clusterName,
		Namespace: namespace,
		AuthInfo:  userName,
	}

	authInfos := make(map[string]*clientcmdapi.AuthInfo)
	authInfos[userName] = &clientcmdapi.AuthInfo{
		Token:                 restConfig.BearerToken,
		ClientCertificateData: restConfig.TLSClientConfig.CertData,
		ClientKeyData:         restConfig.TLSClientConfig.KeyData,
	}

	return &clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		Contexts:       contexts,
		CurrentContext: contextName,
		AuthInfos:      authInfos,
	}, nil
}

// writeInClusterKubeconfigToFile writes the clientcmdapi.Config to a kubeconfig file.
func writeInClusterKubeconfigToFile(ctx context.Context, filePath string, clientConfig clientcmdapi.Config) error {
	log := ctrl.LoggerFrom(ctx)

	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			return errors.Wrapf(err, "failed to create directory %s", dir)
		}
	}

	log.V(2).Info("Writing kubeconfig to location", "location", filePath)
	if err := clientcmd.WriteToFile(clientConfig, filePath); err != nil {
		return err
	}

	return nil
}

// GetWorkloadClusterK8sClient returns the K8s client of an OCNE cluster if it exists.
func (k *KubeconfigGetter) GetWorkloadClusterK8sClient(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) (*kubernetes.Clientset, error) {
	log := ctrl.LoggerFrom(ctx)

	k8sRestConfig, err := BuildWorkloadClusterRESTKubeConfig(ctx, fleetBindingName, kubeconfig, clusterName)
	if err != nil {
		log.Error(err, "failed to get k8s rest config")
		return nil, errors.Wrap(err, "failed to get k8s rest config")
	}
	return GetKubernetesClientsetWithConfig(k8sRestConfig)
}

// GetWorkloadClusterDynamicK8sClient returns the Dynamic K8s client of an OCNE cluster if it exists.
func (k *KubeconfigGetter) GetWorkloadClusterDynamicK8sClient(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) (dynamic.Interface, error) {
	log := ctrl.LoggerFrom(ctx)
	k8sRestConfig, err := BuildWorkloadClusterRESTKubeConfig(ctx, fleetBindingName, kubeconfig, clusterName)
	if err != nil {
		log.Error(err, "failed to get k8s rest config")
		return nil, errors.Wrap(err, "failed to get k8s rest config")
	}
	return dynamic.NewForConfig(k8sRestConfig)
}

// GetVerrazzanoFromRemoteCluster fetches the Verrazzano object from a remote cluster.
func (k *KubeconfigGetter) GetVerrazzanoFromRemoteCluster(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) (*Verrazzano, error) {
	log := ctrl.LoggerFrom(ctx)
	dclient, err := k.GetWorkloadClusterDynamicK8sClient(ctx, fleetBindingName, kubeconfig, clusterName)
	if err != nil {
		log.Error(err, "unable to get workload kubeconfig ")
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    APIGroup,
		Version:  APIVersionBeta1,
		Resource: VerrazzanoResource,
	}

	var vzinstalled Verrazzano
	vzos, err := dclient.Resource(gvr).Namespace(VerrazzanoInstallNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error(err, "Unable to fetch vz install")
		return nil, err
	}
	if len(vzos.Items) == 0 {
		log.Info("verrazzano installation not found")
		return nil, nil
	}
	for _, vz := range vzos.Items {
		modBinaryData, err := json.Marshal(vz.Object)
		if err != nil {
			log.Error(err, "json marshalling error")
			return nil, err
		}
		err = json.Unmarshal(modBinaryData, &vzinstalled)
		if err != nil {
			log.Error(err, "json unmarshalling error")
			return nil, err
		}
	}
	return &vzinstalled, nil

}

// DeleteVerrazzanoFromRemoteCluster triggers the Verrazzano deletion on the remote cluster.
func (k *KubeconfigGetter) DeleteVerrazzanoFromRemoteCluster(ctx context.Context, vz *Verrazzano, fleetBindingName, kubeconfig, clusterName string) error {
	log := ctrl.LoggerFrom(ctx)

	dclient, err := k.GetWorkloadClusterDynamicK8sClient(ctx, fleetBindingName, kubeconfig, clusterName)
	if err != nil {
		log.Error(err, "unable to get workload kubeconfig ")
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    "install.verrazzano.io",
		Version:  "v1beta1",
		Resource: "verrazzanos",
	}
	return dclient.Resource(gvr).Namespace(vz.Metadata.Namespace).Delete(context.TODO(), vz.Metadata.Name, metav1.DeleteOptions{})
}

// WaitForVerrazzanoUninstallCompletion waits for verrazzano uninstall process to complete within a defined timeout
func (k *KubeconfigGetter) WaitForVerrazzanoUninstallCompletion(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) error {
	log := ctrl.LoggerFrom(ctx)
	done := false
	var timeSeconds float64

	timeParse, err := time.ParseDuration(VERRAZZANO_UNINSTALL_TIMEOUT_MINUTES)
	if err != nil {
		log.Error(err, "Unable to parse time duration")
		return err
	}
	totalSeconds := timeParse.Seconds()

	for !done {
		vz, err := k.GetVerrazzanoFromRemoteCluster(ctx, fleetBindingName, kubeconfig, clusterName)
		if err != nil {
			log.Error(err, "unable to fetch verrazzano install from workload cluster")
			return err
		}
		if vz != nil {
			if timeSeconds < totalSeconds {
				message := fmt.Sprintf("Verrazzano detected on cluster '%s' ,state: '%s', component health: '%s'", clusterName, vz.Status.State, vz.Status.Available)
				duration, err := WaitRandom(ctx, message, VERRAZZANO_UNINSTALL_TIMEOUT_MINUTES)
				if err != nil {
					return err
				}
				timeSeconds = timeSeconds + float64(duration)
			} else {
				log.Error(err, "verrazzano deleetion timeout '%s' exceeded.", VERRAZZANO_UNINSTALL_TIMEOUT_MINUTES)
				return err
			}
		} else {
			done = true
		}
	}
	return nil
}

// CreateOrUpdateVerrazzano starts verrazzano deployment
func (k *KubeconfigGetter) CreateOrUpdateVerrazzano(ctx context.Context, fleetBindingName, kubeconfig, clusterName string, vzSpecRawExtension *runtime.RawExtension) error {
	log := ctrl.LoggerFrom(ctx)
	vzSpecObject, err := ConvertRawExtensionToUnstructured(vzSpecRawExtension)
	if err != nil {
		log.Error(err, "Failed to convert raw extension to unstructured data")
		return err
	}
	return PatchVerrazzano(ctx, fleetBindingName, kubeconfig, clusterName, vzSpecObject)
}
