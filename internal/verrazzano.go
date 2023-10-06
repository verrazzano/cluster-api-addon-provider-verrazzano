// Copyright (c) 2023, Oracle and/or its affiliates.

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/models"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/constants"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// GetVerrazzanoFromRemoteCluster fetches the Verrazzano object from a remote cluster.
func GetVerrazzanoFromRemoteCluster(ctx context.Context, c Client, fleetBindingName, kubeconfig, clusterName string) (*models.Verrazzano, error) {
	log := controllerruntime.LoggerFrom(ctx)
	dclient, err := c.GetWorkloadClusterDynamicK8sClient(ctx, kubeconfig)
	if err != nil {
		log.Error(err, "unable to get workload kubeconfig ")
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    constants.APIGroup,
		Version:  constants.APIVersionBeta1,
		Resource: constants.VerrazzanoResource,
	}

	var vzinstalled models.Verrazzano
	vzos, err := dclient.Resource(gvr).Namespace(constants.VerrazzanoInstallNamespace).List(context.TODO(), v1.ListOptions{})
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
func DeleteVerrazzanoFromRemoteCluster(ctx context.Context, c Client, vz *models.Verrazzano, fleetBindingName, kubeconfig, clusterName string) error {
	log := controllerruntime.LoggerFrom(ctx)

	dclient, err := c.GetWorkloadClusterDynamicK8sClient(ctx, kubeconfig)
	if err != nil {
		log.Error(err, "unable to get workload kubeconfig ")
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    "install.verrazzano.io",
		Version:  "v1beta1",
		Resource: "verrazzanos",
	}
	return dclient.Resource(gvr).Namespace(vz.Metadata.Namespace).Delete(context.TODO(), vz.Metadata.Name, v1.DeleteOptions{})
}

// WaitForVerrazzanoUninstallCompletion waits for verrazzano uninstall process to complete within a defined timeout
func WaitForVerrazzanoUninstallCompletion(ctx context.Context, c Client, fleetBindingName, kubeconfig, clusterName string) error {
	log := controllerruntime.LoggerFrom(ctx)
	done := false
	var timeSeconds float64

	timeParse, err := time.ParseDuration(constants.VERRAZZANO_UNINSTALL_TIMEOUT_MINUTES)
	if err != nil {
		log.Error(err, "Unable to parse time duration")
		return err
	}
	totalSeconds := timeParse.Seconds()

	for !done {
		vz, err := GetVerrazzanoFromRemoteCluster(ctx, c, fleetBindingName, kubeconfig, clusterName)
		if err != nil {
			log.Error(err, "unable to fetch verrazzano install from workload cluster")
			return err
		}
		if vz != nil {
			if timeSeconds < totalSeconds {
				message := fmt.Sprintf("Verrazzano detected on cluster '%s' ,state: '%s', component health: '%s'", clusterName, vz.Status.State, vz.Status.Available)
				duration, err := utils.WaitRandom(ctx, message, constants.VERRAZZANO_UNINSTALL_TIMEOUT_MINUTES)
				if err != nil {
					return err
				}
				timeSeconds = timeSeconds + float64(duration)
			} else {
				log.Error(err, "verrazzano deletion timeout '%s' exceeded.", constants.VERRAZZANO_UNINSTALL_TIMEOUT_MINUTES)
				return err
			}
		} else {
			done = true
		}
	}
	return nil
}

// CreateOrUpdateVerrazzano starts verrazzano deployment
func (k *KubeconfigGetter) CreateOrUpdateVerrazzano(ctx context.Context, client Client, fleetBindingName, kubeconfig, clusterName string, vzSpecRawExtension *runtime.RawExtension) error {
	log := controllerruntime.LoggerFrom(ctx)
	vzSpecObject, err := utils.ConvertRawExtensionToUnstructured(vzSpecRawExtension)
	if err != nil {
		log.Error(err, "Failed to convert raw extension to unstructured data")
		return err
	}
	return PatchVerrazzano(ctx, client, fleetBindingName, kubeconfig, clusterName, vzSpecObject)
}

// PatchVerrazzano helps apply the verrazzano config on the remote cluster
func PatchVerrazzano(ctx context.Context, client Client, fleetBindingName, kubeconfig, clusterName string, obj *unstructured.Unstructured) error {
	log := controllerruntime.LoggerFrom(ctx)

	dclient, err := client.GetWorkloadClusterDynamicK8sClient(ctx, kubeconfig)
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
		Group:    constants.APIGroup,
		Version:  constants.APIVersionBeta1,
		Resource: constants.VerrazzanoResource,
	}

	data, err := json.Marshal(newObj.Object)
	if err != nil {
		log.Error(err, "failed in json marshall")
		return err
	}
	log.V(5).Info("verrazzano spec applied", string(data))

	//Apply the Yaml
	_, err = dclient.Resource(gvr).Namespace(constants.VerrazzanoInstallNamespace).Patch(ctx, constants.VerrazzanoInstallName, types.StrategicMergePatchType, data, v1.PatchOptions{
		FieldManager: "verrazzano-platform-controller"})
	if err != nil {
		log.Error(err, fmt.Sprintf("failed to apply verrazzano spec"))
		return err
	}
	return nil
}

// processVerrazzanoSpec - wrap the updates in a spec field
func processVerrazzanoSpec(ctx context.Context, inputObj *unstructured.Unstructured) (unstructured.Unstructured, error) {
	log := controllerruntime.LoggerFrom(ctx)
	var newObj unstructured.Unstructured
	if newObj.Object == nil {
		newObj.Object = make(map[string]interface{})
	}
	if err := unstructured.SetNestedField(newObj.Object, inputObj.Object, constants.Spec); err != nil {
		log.Error(err, "unable to set nested field")
		return newObj, err
	}
	return newObj, nil
}
