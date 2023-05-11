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

package clients

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

// ContainerContext encapsulates the context in which a command is run; the namespace, pod, and container.
type ContainerContext struct {
	namespace     string
	podName       string
	containerName string
}

func (clientsholder *Clientset) findPodNameFromPrefix(namespace, prefix string) (string, error) {
	podList, err := clientsholder.K8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to getting pod list: %w", err)
	}
	podNames := make([]string, 0)

	for i := range podList.Items {
		if strings.HasPrefix(podList.Items[i].Name, prefix) {
			podNames = append(podNames, podList.Items[i].Name)
		}
	}

	switch len(podNames) {
	case 0:
		return "", fmt.Errorf("no pod with prefix %v found in namespace %v", prefix, namespace)
	case 1:
		return podNames[0], nil
	default:
		return "", fmt.Errorf("too many (%v) pods with prefix %v found in namespace %v", len(podNames), prefix, namespace)
	}
}

func NewContainerContext(
	clientset *Clientset,
	namespace, podNamePrefix, containerName string,
) (ContainerContext, error) {
	podName, err := clientset.findPodNameFromPrefix(namespace, podNamePrefix)
	if err != nil {
		return ContainerContext{}, err
	}
	ctx := ContainerContext{
		namespace:     namespace,
		podName:       podName,
		containerName: containerName,
	}
	return ctx, nil
}

func (c *ContainerContext) GetNamespace() string {
	return c.namespace
}

func (c *ContainerContext) GetPodName() string {
	return c.podName
}

func (c *ContainerContext) GetContainerName() string {
	return c.containerName
}

// ExecCommand runs command in a container and returns output buffers
//
//nolint:lll // allow slightly long function definition
func (clientsholder *Clientset) ExecCommandContainer(ctx ContainerContext, command []string) (stdout, stderr string, err error) {
	commandStr := command
	var buffOut bytes.Buffer
	var buffErr bytes.Buffer
	log.Debug(fmt.Sprintf(
		"execute command on ns=%s, pod=%s container=%s, cmd: %s",
		ctx.GetNamespace(),
		ctx.GetPodName(),
		ctx.GetContainerName(),
		strings.Join(commandStr, " "),
	))
	req := clientsholder.K8sClient.CoreV1().RESTClient().
		Post().
		Namespace(ctx.GetNamespace()).
		Resource("pods").
		Name(ctx.GetPodName()).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: ctx.GetContainerName(),
			Command:   commandStr,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(clientsholder.RestConfig, "POST", req.URL())
	if err != nil {
		log.Error(err)
		return stdout, stderr, fmt.Errorf("error setting up remote command: %w", err)
	}
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdout: &buffOut,
		Stderr: &buffErr,
	})
	stdout, stderr = buffOut.String(), buffErr.String()
	if err != nil {
		log.Error(err)
		log.Error(req.URL())
		log.Error("command: ", command)
		log.Error("stderr: ", stderr)
		log.Error("stdout: ", stdout)
		return stdout, stderr, fmt.Errorf("error running remote command: %w", err)
	}
	return stdout, stderr, nil
}
