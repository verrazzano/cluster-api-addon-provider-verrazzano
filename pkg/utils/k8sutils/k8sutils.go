// Copyright (c) 2023, Oracle and/or its affiliates.

package k8sutils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/models"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	"os"
	"path/filepath"

	gerrors "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sYaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime"
)

var decUnstructured = k8sYaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

func setConfigQPSBurst(config *rest.Config) {
	config.Burst = constants.APIServerBurst
	config.QPS = constants.APIServerQPS
}

// GetKubeConfig Returns kubeconfig from KUBECONFIG env var if set
// Else from default location ~/.kube/config
func GetKubeConfig() (*rest.Config, error) {
	var config *rest.Config
	kubeConfigLoc, err := GetKubeConfigLocation()
	if err != nil {
		return nil, err
	}
	config, err = clientcmd.BuildConfigFromFlags("", kubeConfigLoc)
	if err != nil {
		return nil, err
	}
	setConfigQPSBurst(config)
	return config, nil
}

// GetKubeConfigLocation Helper function to obtain the default kubeConfig location
func GetKubeConfigLocation() (string, error) {
	if testKubeConfig := os.Getenv(constants.EnvVarTestKubeConfig); len(testKubeConfig) > 0 {
		return testKubeConfig, nil
	}

	if kubeConfig := os.Getenv(constants.EnvVarKubeConfig); len(kubeConfig) > 0 {
		return kubeConfig, nil
	}

	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config"), nil
	}

	return "", errors.New("unable to find kubeconfig")

}

func GetRESTConfigGivenString(kubeconfig string) (*rest.Config, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}
	setConfigQPSBurst(config)
	return config, nil
}

// GetKubernetesClientsetWithConfig returns the Kubernetes clientset for the given configuration
func GetKubernetesClientsetWithConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	var clientset *kubernetes.Clientset
	clientset, err := kubernetes.NewForConfig(config)
	return clientset, err
}

// BuildWorkloadClusterRESTKubeConfig writes the kubeconfig to a temporary file and then returns the rest.config
func BuildWorkloadClusterRESTKubeConfig(kubeconfig string) (*rest.Config, error) {
	return GetRESTConfigGivenString(kubeconfig)
}

// IsPodReady checks if POD is in ready state
func IsPodReady(ctx context.Context, pod *v1.Pod) bool {
	log := ctrl.LoggerFrom(ctx)
	for _, condition := range pod.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == "True" {
			log.V(1).Info(fmt.Sprintf("Pod '%s' in namespace '%s' is in '%s' state", pod.Name, pod.Namespace, condition.Type))
			return true
		}
	}
	log.V(1).Info(fmt.Sprintf("Pod '%s' in namespace '%s' is still not Ready", pod.Name, pod.Namespace))
	return false
}

func GetCoreV1Client() (v12.CoreV1Interface, error) {
	restConfig := ctrl.GetConfigOrDie()
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return kubeClient.CoreV1(), nil
}

func GetVerrazzanoVersionOfAdminCluster() (string, error) {
	k8sRestConfig, err := rest.InClusterConfig()
	if err != nil {
		return "", gerrors.Wrap(err, "failed to get k8s rest config")
	}
	dynClient, err := dynamic.NewForConfig(k8sRestConfig)

	if err != nil {
		return "", gerrors.Wrap(err, "failed to get dynamic client")
	}
	gvr := schema.GroupVersionResource{
		Group:    constants.APIGroup,
		Version:  constants.APIVersionBeta1,
		Resource: constants.VerrazzanoResource,
	}

	var vzinstalled models.Verrazzano
	vzos, err := dynClient.Resource(gvr).Namespace(constants.VerrazzanoInstallNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", gerrors.Wrap(err, "Unable to fetch vz install")
	}
	if len(vzos.Items) == 0 {
		return "", nil
	}
	for _, vz := range vzos.Items {
		modBinaryData, err := json.Marshal(vz.Object)
		if err != nil {
			return "", gerrors.Wrap(err, "json marshalling error")
		}
		err = json.Unmarshal(modBinaryData, &vzinstalled)
		if err != nil {
			return "", gerrors.Wrap(err, "json unmarshalling error")
		}
	}
	return vzinstalled.Status.Version, nil
}
