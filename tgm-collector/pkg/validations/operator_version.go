// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/clients"
	validationsBase "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/validations"
)

const (
	PTPOperatorVersionID          = TGMEnvVerPath + "/openshift/ptp-operator/"
	ptpOperatorVersionDescription = "PTP Operator Version is valid"
	MinOperatorVersion            = "4.14.0-0" // trailing -0 is required to allow preGA version
	ptpOperatorDiplayName         = "PTP Operator"
)

type CSV struct {
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
}

func getOperatorVersion(
	group,
	version,
	resource,
	namespace string,
	client *clients.Clientset,
) (string, error) {
	dynamicClient, err := dynamic.NewForConfig(client.RestConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create dynamic client: %w", err)
	}

	resourceID := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	list, err := dynamicClient.Resource(resourceID).Namespace(namespace).
		List(context.Background(), metav1.ListOptions{})

	if err != nil {
		return "", fmt.Errorf("failed to fetch operator version %w", err)
	}

	for _, item := range list.Items {
		value := item.Object["spec"]
		crd := &CSV{}
		marsh, err := json.Marshal(value)
		if err != nil {
			log.Debug("failed to marshal cluster service version spec", err)
			continue
		}
		err = json.Unmarshal(marsh, crd)
		if err != nil {
			log.Debug("failed to marshal cluster service version spec", err)
			continue
		}
		if crd.DisplayName == ptpOperatorDiplayName {
			return crd.Version, nil
		}
	}
	return "", errors.New("failed to find PTP Operator CSV")
}

func NewOperatorVersion(args map[string]any) (validationsBase.Validation, error) {
	rawClient, ok := args["clientset"]
	if !ok {
		return nil, fmt.Errorf("clientset not in args")
	}
	client, ok := rawClient.(*clients.Clientset)
	if !ok {
		return nil, fmt.Errorf("clientset not in args")
	}
	version, err := getOperatorVersion(
		"operators.coreos.com",
		"v1alpha1",
		"clusterserviceversions",
		"openshift-ptp",
		client,
	)
	v := VersionWithErrorCheck{
		VersionCheck: VersionCheck{
			id:           PTPOperatorVersionID,
			Version:      version,
			checkVersion: version,
			MinVersion:   MinOperatorVersion,
			description:  ptpOperatorVersionDescription,
			order:        ptpOperatorVersionOrdering,
		},
		Error: err,
	}
	return &v, nil
}

func init() {
	validationsBase.RegisterValidation(PTPOperatorVersionID, NewOperatorVersion, []string{})
}
