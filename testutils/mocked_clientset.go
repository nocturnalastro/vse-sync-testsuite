// Copyright 2023 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutils

import (
	"net/url"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakeK8s "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/clients"
)

const kubeconfigPath string = "test_files/kubeconfig"

func GetMockedClientSet(k8APIObjects ...runtime.Object) *clients.Clientset {
	clients.ClearClientSet()
	clientset := clients.GetClientset(kubeconfigPath)
	fakeK8sClient := fakeK8s.NewSimpleClientset(k8APIObjects...)

	config := rest.ClientContentConfig{
		GroupVersion: schema.GroupVersion{Version: "v1"},
	}
	fakeRestClient, err := rest.NewRESTClient(&url.URL{}, "", config, nil, nil)
	if err != nil {
		panic("Failed to create rest client")
	}
	clientset.K8sClient = fakeK8sClient
	clientset.K8sRestClient = fakeRestClient
	return clientset
}
