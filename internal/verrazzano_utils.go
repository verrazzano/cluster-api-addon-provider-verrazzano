// Copyright (c) 2023, Oracle and/or its affiliates.

package internal

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"github.com/pkg/errors"
	addonsv1alpha1 "github.com/verrazzano/cluster-api-addon-provider-verrazzano/api/v1alpha1"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/models"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/constants"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/k8sutils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"path"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"text/template"
)

var (
	//go:embed vpoImageMeta.tmpl
	vpoValuesTemplate string

	defaultTemplateFuncMap = template.FuncMap{
		"Indent": templateYAMLIndent,
	}

	GetCoreV1Func = k8sutils.GetCoreV1Client
)

func templateYAMLIndent(i int, input string) string {
	split := strings.Split(input, "\n")
	ident := "\n" + strings.Repeat(" ", i)
	return strings.Repeat(" ", i) + strings.Join(split, ident)
}

func generate(kind string, tpl string, data interface{}) ([]byte, error) {
	tm := template.New(kind).Funcs(defaultTemplateFuncMap)
	t, err := tm.Parse(tpl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s template", kind)
	}
	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return nil, errors.Wrapf(err, "failed to generate %s template", kind)
	}
	return out.Bytes(), nil
}

// GetDefaultVPOImageFromHelmChart returns the default VPO image found in the VPO helm charts value.yaml
func GetDefaultVPOImageFromHelmChart() (string, error) {
	data, err := os.ReadFile(filepath.Join(constants.VerrazzanoPlatformOperatorChartPath, "values.yaml"))
	if err != nil {
		return "", err
	}

	var values map[string]interface{}
	err = yaml.Unmarshal(data, &values)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", values["image"]), nil
}

// ParseDefaultVPOImage parses the default VPO image and returns the parts of the VPO image
func ParseDefaultVPOImage(vpoImage string) (registry string, repo string, image string, tag string) {
	splitTag := strings.Split(vpoImage, ":")
	tag = splitTag[1]
	splitImage := strings.Split(splitTag[0], "/")
	image = splitImage[len(splitImage)-1]
	regRepo := strings.TrimSuffix(splitTag[0], "/"+image)
	splitRegistry := strings.Split(regRepo, "/")
	registry = splitRegistry[0]
	repo = strings.TrimPrefix(regRepo, registry+"/")
	return registry, repo, image, tag
}

func generateDataValuesForVerrazzanoPlatformOperator(ctx context.Context, fleetSpec *addonsv1alpha1.VerrazzanoFleetBinding) ([]byte, error) {
	log := ctrl.LoggerFrom(ctx)

	var helmMeta VPOHelmValuesTemplate

	vpoImage, err := GetDefaultVPOImageFromHelmChart()
	if err != nil {
		log.Error(err, "failed to get verrazzano-platform-operator image from helm chart")
		return nil, err
	}

	var registry string
	var repo string
	var image string
	var tag string

	// Parse the default VPO image and return various parts of the image
	registry, repo, image, tag = ParseDefaultVPOImage(vpoImage)

	spec := fleetSpec.Spec
	// Setting default values for image
	if spec.Image != nil {
		// Set defaults or honor overrides
		if spec.Image.Repository == "" {
			helmMeta.Image = fmt.Sprintf("%s/%s/%s", registry, repo, image)
		} else {
			imageList := strings.Split(strings.Trim(strings.TrimSpace(spec.Image.Repository), "/"), "/")
			if imageList[len(imageList)-1] == image {
				helmMeta.Image = spec.Image.Repository
			} else {
				helmMeta.Image = fmt.Sprintf("%s/%s", strings.Join(imageList[0:len(imageList)], "/"), image)
			}
		}

		if spec.Image.Tag == "" {
			helmMeta.Image = fmt.Sprintf("%s:%s", helmMeta.Image, tag)
		} else {
			helmMeta.Image = fmt.Sprintf("%s:%s", helmMeta.Image, strings.TrimSpace(spec.Image.Tag))
		}

		if spec.Image.PullPolicy == "" {
			helmMeta.PullPolicy = constants.DefaultImagePullPolicy
		} else {
			helmMeta.PullPolicy = strings.TrimSpace(spec.Image.PullPolicy)
		}

		// Parse the override image and return various parts of the image
		registry, repo, image, tag = ParseDefaultVPOImage(helmMeta.Image)
	} else {
		// If nothing has been specified for the image in the API
		helmMeta = VPOHelmValuesTemplate{
			PullPolicy: constants.DefaultImagePullPolicy,
		}

	}

	if spec.PrivateRegistry != nil {
		if spec.PrivateRegistry.Enabled {
			helmMeta.PrivateRegistry = true
			helmMeta.Registry = registry
			helmMeta.Repository = strings.TrimSuffix(repo, "/verrazzano")
		}
	}

	// This handles the use case where a developer has built a verrazzano-platform-operator in the non-default
	// registry and private registry is not being used.  In this case, the app operator and cluster operator
	// need to be explicitly set in the helm chart otherwise the wrong registry (ghcr.io) will be used resulting
	// in image pull errors.
	if registry != constants.VerrazzanoPlatformOperatorRepo {
		if spec.PrivateRegistry != nil {
			if !spec.PrivateRegistry.Enabled {
				helmMeta.AppOperatorImage = fmt.Sprintf("%s/%s/%s:%s", registry, repo, strings.ReplaceAll(image, "verrazzano-platform-operator", "verrazzano-application-operator"), tag)
				helmMeta.ClusterOperatorImage = fmt.Sprintf("%s/%s/%s:%s", registry, repo, strings.ReplaceAll(image, "verrazzano-platform-operator", "verrazzano-cluster-operator"), tag)
			}
		}
	}

	if spec.ImagePullSecrets != nil {
		helmMeta.ImagePullSecrets = spec.ImagePullSecrets
	} else {
		helmMeta.ImagePullSecrets = []addonsv1alpha1.SecretName{}
	}

	return generate("HelmValues", vpoValuesTemplate, helmMeta)
}

// GetVerrazzanoPlatformOperatorAddons returns the needed info to install the verrazzano-platform-operator helm chart.
func GetVerrazzanoPlatformOperatorAddons(ctx context.Context, fleetSpec *addonsv1alpha1.VerrazzanoFleetBinding) (*models.HelmModuleAddons, error) {
	log := ctrl.LoggerFrom(ctx)

	client, err := GetCoreV1Func()
	if err != nil {
		return nil, err
	}

	// Get the config map containing the verrazzano-platform-operator helm chart.
	cm, err := client.ConfigMaps(constants.VerrazzanoPlatformOperatorNameSpace).Get(ctx, constants.VerrazzanoPlatformOperatorHelmChartConfigMapName, metav1.GetOptions{})
	if err != nil {
		log.Error(err, "Installing the verrazzano-platform-operator helm chart in an OCNE cluster requires a Verrazzano installation")
		return nil, err
	}

	// Cleanup verrazzano-platform-operator helm chart from a previous installation.
	err = os.RemoveAll(constants.VerrazzanoPlatformOperatorChartPath)
	if err != nil {
		log.Error(err, "Unable to cleanup chart directory for verrazzano platform operator")
		return nil, err
	}

	// Create the needed directories if they don't exist.
	err = os.MkdirAll(filepath.Join(constants.VerrazzanoPlatformOperatorChartPath, "crds"), 0755)
	if err != nil {
		log.Error(err, "Unable to create crds chart directory for verrazzano platform operator")
		return nil, err
	}

	err = os.MkdirAll(filepath.Join(constants.VerrazzanoPlatformOperatorChartPath, "templates"), 0755)
	if err != nil {
		log.Error(err, "Unable to create templates chart directory for verrazzano platform operator")
		return nil, err
	}

	// Iterate through the config map and create all the verrazzano-platform-operator helm chart files.
	for k, v := range cm.Data {
		fileName := strings.ReplaceAll(k, "...", "/")
		fp, fileErr := os.Create(path.Join(constants.VerrazzanoPlatformOperatorChartPath, fileName))
		if fileErr != nil {
			log.Error(fileErr, "Unable to create file")
			return nil, fileErr
		}
		defer fp.Close()
		if _, fileErr = fp.Write([]byte(v)); err != nil {
			log.Error(fileErr, "Unable to write to file")
			return nil, fileErr
		}
	}

	// Get the values to pass to the verrazzano-platform-operator helm chart
	out, err := generateDataValuesForVerrazzanoPlatformOperator(ctx, fleetSpec)
	if err != nil {
		log.Error(err, "failed to generate data")
		return nil, err
	}

	return &models.HelmModuleAddons{
		ChartName:        constants.VerrazzanoPlatformOperatorChartName,
		ReleaseName:      constants.VerrazzanoPlatformOperatorChartName,
		ReleaseNamespace: constants.VerrazzanoPlatformOperatorNameSpace,
		RepoURL:          constants.VerrazzanoPlatformOperatorChartPath,
		Local:            true,
		ValuesTemplate:   string(out),
	}, nil
}

type VPOHelmValuesTemplate struct {
	Image                string                      `json:"image,omitempty"`
	PrivateRegistry      bool                        `json:"privateRegistry"`
	Repository           string                      `json:"repository,omitempty"`
	Registry             string                      `json:"registry,omitempty"`
	PullPolicy           string                      `json:"pullPolicy,omitempty"`
	ImagePullSecrets     []addonsv1alpha1.SecretName `json:"imagePullSecrets,omitempty"`
	AppOperatorImage     string                      `json:"appOperatorImage,omitempty"`
	ClusterOperatorImage string                      `json:"clusterOperatorImage,omitempty"`
}
