// Copyright (c) 2023, Oracle and/or its affiliates.

package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/constants"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"math/big"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

// WaitRandom generates a random number between min and max
func WaitRandom(ctx context.Context, message, timeout string) (int, error) {
	log := ctrl.LoggerFrom(ctx)
	randomBig, err := rand.Int(rand.Reader, big.NewInt(constants.Max))
	if err != nil {
		return 0, fmt.Errorf("Unable to generate random number %v", zap.Error(err))
	}
	randomInt := int(randomBig.Int64())
	if randomInt < constants.Min {
		randomInt = (constants.Min + constants.Max) / 2
	}
	timeParse, err := time.ParseDuration(timeout)
	if err != nil {
		return 0, fmt.Errorf("Unable to parse time duration %v", zap.Error(err))
	}
	// handle timeouts lesser that generated min!
	if float64(randomInt) > timeParse.Seconds() {
		randomInt = int(timeParse.Seconds())
	}
	log.V(1).Info(fmt.Sprintf("%v . Wait for '%v' seconds ...", message, randomInt))
	time.Sleep(time.Second * time.Duration(randomInt))
	return randomInt, nil
}

// ConvertRawExtensionToUnstructured converts a runtime.RawExtension to unstructured.Unstructured.
func ConvertRawExtensionToUnstructured(rawExtension *runtime.RawExtension) (*unstructured.Unstructured, error) {
	var obj runtime.Object
	var scope conversion.Scope
	if err := runtime.Convert_runtime_RawExtension_To_runtime_Object(rawExtension, &obj, scope); err != nil {
		return nil, err
	}

	innerObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: innerObj}, nil
}

func GetEnvValueWithDefault(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return defaultValue
}
