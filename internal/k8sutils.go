// Copyright (c) 2023, Oracle and/or its affiliates.

package internal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	gerrors "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sYaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
)

var decUnstructured = k8sYaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

func setConfigQPSBurst(config *rest.Config) {
	config.Burst = APIServerBurst
	config.QPS = APIServerQPS
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
	if testKubeConfig := os.Getenv(EnvVarTestKubeConfig); len(testKubeConfig) > 0 {
		return testKubeConfig, nil
	}

	if kubeConfig := os.Getenv(EnvVarKubeConfig); len(kubeConfig) > 0 {
		return kubeConfig, nil
	}

	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config"), nil
	}

	return "", errors.New("unable to find kubeconfig")

}

// GetKubeConfigGivenPathAndContext returns a rest.config given the kubeconfigPath and context
func GetKubeConfigGivenPathAndContext(kubeConfigPath string, kubeContext string) (*rest.Config, error) {
	// If no values passed, call default GetKubeConfig
	if len(kubeConfigPath) == 0 && len(kubeContext) == 0 {
		return GetKubeConfig()
	}

	// Default the value of kubeConfigLoc?
	var err error
	if len(kubeConfigPath) == 0 {
		kubeConfigPath, err = GetKubeConfigLocation()
		if err != nil {
			return nil, err
		}
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&clientcmd.ConfigOverrides{CurrentContext: kubeContext}).ClientConfig()
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
func BuildWorkloadClusterRESTKubeConfig(ctx context.Context, fleetBindingName, kubeconfig, clusterName string) (*rest.Config, error) {
	log := ctrl.LoggerFrom(ctx)
	tmpFile, err := os.CreateTemp(os.TempDir(), fleetBindingName)
	if err != nil {
		log.Error(err, "failed to create temporary file")
		return nil, gerrors.Wrap(err, "failed to create temporary file")
	}
	defer os.RemoveAll(tmpFile.Name())

	if err := os.WriteFile(tmpFile.Name(), []byte(kubeconfig), 0600); err != nil {
		log.Error(err, "failed to write to destination file")
		return nil, gerrors.Wrap(err, "failed to write to destination file")
	}
	clusterContext := fmt.Sprintf("%s-admin@%s", clusterName, clusterName)
	clusterContext = getEnvValueWithDefault("DEV_CLUSTER_CONTEXT", clusterContext)

	return GetKubeConfigGivenPathAndContext(tmpFile.Name(), clusterContext)
}

// PatchVerrazzano helps apply the verrazzano config on the remote cluster
func PatchVerrazzano(ctx context.Context, fleetBindingName, kubeconfig, clusterName string, obj *unstructured.Unstructured) error {
	log := ctrl.LoggerFrom(ctx)

	k := KubeconfigGetter{}
	dclient, err := k.GetWorkloadClusterDynamicK8sClient(ctx, fleetBindingName, kubeconfig, clusterName)
	if err != nil {
		log.Error(err, "unable to get dynamic client ")
		return err
	}

	newObj, err := processVerrazzanoSpec(ctx, obj)
	if err != nil {
		log.Error(err, "unable to process verrazzano spec input")
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    APIGroup,
		Version:  APIVersionBeta1,
		Resource: VerrazzanoResource,
	}

	data, err := yaml.Marshal(newObj.Object)
	if err != nil {
		log.Error(err, "failed in yaml marshall")
		return err
	}
	log.V(5).Info("verrazzano spec applied", string(data))

	//Apply the Yaml
	_, err = dclient.Resource(gvr).Namespace(newObj.GetNamespace()).Patch(ctx, newObj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "verrazzano-platform-controller",
	})
	if err != nil {
		log.Error(err, fmt.Sprintf("failed to apply verrazzano spec"))
		return err
	}
	return nil
}

func processVerrazzanoSpec(ctx context.Context, inputObj *unstructured.Unstructured) (unstructured.Unstructured, error) {
	/*
		apiVersion: install.verrazzano.io/v1beta1
		kind: Verrazzano
		metadata:
		  name: verrazzano
		  namespace: default
	*/
	// overwrite above contents even if specified in input spec

	log := ctrl.LoggerFrom(ctx)
	var newObj unstructured.Unstructured
	if newObj.Object == nil {
		newObj.Object = make(map[string]interface{})
	}
	if err := unstructured.SetNestedField(newObj.Object, inputObj.Object, Spec); err != nil {
		log.Error(err, "unable to set nested field")
		return newObj, err
	}
	newObj.SetAPIVersion(fmt.Sprintf("%s/%s", APIGroup, APIVersionBeta1))
	newObj.SetKind(VerrazzanoDomainKind)
	newObj.SetName(VerrazzanoInstallName)
	newObj.SetNamespace(VerrazzanoInstallNamespace)

	return newObj, nil

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
