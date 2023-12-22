// SPDX-License-Identifier: GPL-2.0-or-later

package clients

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/Netflix/go-expect"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/utils"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

const (
	startTimeout    = 5 * time.Second
	deletionTimeout = 10 * time.Minute
)

type Command struct {
	Shell string
	Stdin string
	regex *regexp.Regexp
}

type ExecContext interface {
	ExecCommand(*Command) (string, string, error)
	ExecCommandStdIn(*Command) (string, string, error)
}

var NewSPDYExecutor = remotecommand.NewSPDYExecutor

// ContainerExecContext encapsulates the context in which a command is run; the namespace, pod, and container.
type ContainerExecContext struct {
	clientset     *Clientset
	namespace     string
	podName       string
	containerName string
	podNamePrefix string
}

func (c *ContainerExecContext) refresh() error {
	newPodname, err := c.clientset.FindPodNameFromPrefix(c.namespace, c.podNamePrefix)
	if err != nil {
		return err
	}
	c.podName = newPodname
	return nil
}

func NewContainerContext(
	clientset *Clientset,
	namespace, podNamePrefix, containerName string,
) (*ContainerExecContext, error) {
	podName, err := clientset.FindPodNameFromPrefix(namespace, podNamePrefix)
	if err != nil {
		return &ContainerExecContext{}, err
	}
	ctx := ContainerExecContext{
		namespace:     namespace,
		podName:       podName,
		containerName: containerName,
		podNamePrefix: podNamePrefix,
		clientset:     clientset,
	}
	return &ctx, nil
}

func (c *ContainerExecContext) GetNamespace() string {
	return c.namespace
}

func (c *ContainerExecContext) GetPodName() string {
	return c.podName
}

func (c *ContainerExecContext) GetContainerName() string {
	return c.containerName
}

//nolint:lll,funlen // allow slightly long function definition and function length
func (c *ContainerExecContext) execCommand(cmd *Command) (stdout, stderr string, err error) {
	commandStr := cmd.Shell
	var buffOut bytes.Buffer
	var buffErr bytes.Buffer

	useBuffIn := cmd.Stdin != ""

	logrus.Debugf(
		"execute command on ns=%s, pod=%s container=%s, cmd: %s",
		c.GetNamespace(),
		c.GetPodName(),
		c.GetContainerName(),
		commandStr,
	)
	req := c.clientset.K8sRestClient.Post().
		Namespace(c.GetNamespace()).
		Resource("pods").
		Name(c.GetPodName()).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: c.GetContainerName(),
			Command:   []string{shellCommand},
			Stdin:     useBuffIn,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := NewSPDYExecutor(c.clientset.RestConfig, "POST", req.URL())
	if err != nil {
		logrus.Debug(err)
		return stdout, stderr, fmt.Errorf("error setting up remote command: %w", err)
	}

	var streamOptions remotecommand.StreamOptions
	var bf bytes.Buffer
	bf.WriteString(cmd.Stdin)

	if useBuffIn {
		streamOptions = remotecommand.StreamOptions{
			Stdin:  &bf,
			Stdout: &buffOut,
			Stderr: &buffErr,
		}
	} else {
		streamOptions = remotecommand.StreamOptions{
			Stdout: &buffOut,
			Stderr: &buffErr,
		}
	}

	err = exec.StreamWithContext(context.TODO(), streamOptions)
	stdout, stderr = buffOut.String(), buffErr.String()
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			logrus.Debugf("Pod %s was not found, likely restarted so refreshing context", c.GetPodName())
			refreshErr := c.refresh()
			if refreshErr != nil {
				logrus.Debug("Failed to refresh container context", refreshErr)
			}
		}

		logrus.Debug(err)
		logrus.Debug(req.URL())
		logrus.Debug("command: ", cmd.Shell)
		if useBuffIn {
			logrus.Debug("stdin: ", cmd.Stdin)
		}
		logrus.Debug("stderr: ", stderr)
		logrus.Debug("stdout: ", stdout)
		return stdout, stderr, fmt.Errorf("error running remote command: %w", err)
	}
	return stdout, stderr, nil
}

// ExecCommand runs command in a container and returns output buffers
//
//nolint:lll,funlen // allow slightly long function definition and allow a slightly long function
func (c *ContainerExecContext) ExecCommand(cmd *Command) (stdout, stderr string, err error) {
	return c.execCommand(cmd)
}

//nolint:lll // allow slightly long function definition
func (c *ContainerExecContext) ExecCommandStdIn(cmd *Command) (stdout, stderr string, err error) {
	return c.execCommand(cmd)
}

// ContainerExecContext encapsulates the context in which a command is run; the namespace, pod, and container.
type ContainerCreationExecContext struct {
	*ContainerExecContext
	labels                   map[string]string
	pod                      *corev1.Pod
	containerSecurityContext *corev1.SecurityContext
	containerImage           string
	command                  []string
	volumes                  []*Volume
	hostNetwork              bool
}

type Volume struct {
	VolumeSource corev1.VolumeSource
	Name         string
	MountPath    string
}

func (c *ContainerCreationExecContext) createPod() error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.podName,
			Namespace: c.namespace,
			Labels:    c.labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            c.containerName,
					Image:           c.containerImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
				},
			},
			HostNetwork: c.hostNetwork,
		},
	}
	if len(c.command) > 0 {
		pod.Spec.Containers[0].Command = c.command
	}
	if c.containerSecurityContext != nil {
		pod.Spec.Containers[0].SecurityContext = c.containerSecurityContext
	}
	if len(c.volumes) > 0 {
		volumes := make([]corev1.Volume, 0)
		volumeMounts := make([]corev1.VolumeMount, 0)

		for _, v := range c.volumes {
			volumes = append(volumes, corev1.Volume{Name: v.Name, VolumeSource: v.VolumeSource})
			pod.Spec.Volumes = volumes
			volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: v.Name, MountPath: v.MountPath})
			pod.Spec.Containers[0].VolumeMounts = volumeMounts
		}
	}

	pod, err := c.clientset.K8sClient.CoreV1().Pods(pod.Namespace).Create(
		context.TODO(),
		pod,
		metav1.CreateOptions{},
	)
	c.pod = pod
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}
	return nil
}

func (c *ContainerCreationExecContext) listPods(options *metav1.ListOptions) (*corev1.PodList, error) {
	pods, err := c.clientset.K8sClient.CoreV1().Pods(c.pod.Namespace).List(
		context.TODO(),
		*options,
	)
	if err != nil {
		return pods, fmt.Errorf("failed to find pods: %s", err.Error())
	}
	return pods, nil
}

func (c *ContainerCreationExecContext) refeshPod() error {
	pods, err := c.listPods(&metav1.ListOptions{
		FieldSelector:   fields.OneTermEqualSelector("metadata.name", c.podName).String(),
		ResourceVersion: c.pod.ResourceVersion,
	})
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("failed to find pod: %s", c.podName)
	}
	c.pod = &pods.Items[0]

	return nil
}

func (c *ContainerCreationExecContext) isPodRunning() (bool, error) {
	err := c.refeshPod()
	if err != nil {
		return false, err
	}
	if c.pod.Status.Phase == corev1.PodRunning {
		return true, nil
	}
	return false, nil
}

func (c *ContainerCreationExecContext) waitForPodToStart() error {
	start := time.Now()
	for time.Since(start) <= startTimeout {
		running, err := c.isPodRunning()
		if err != nil {
			return err
		}
		if running {
			return nil
		}
		time.Sleep(time.Microsecond)
	}
	return errors.New("timed out waiting for pod to start")
}

func (c *ContainerCreationExecContext) CreatePodAndWait() error {
	var err error
	running := false
	if c.pod != nil {
		running, err = c.isPodRunning()
		if err != nil {
			return err
		}
	}
	if !running {
		err := c.createPod()
		if err != nil {
			return err
		}
	}
	return c.waitForPodToStart()
}

func (c *ContainerCreationExecContext) deletePod() error {
	deletePolicy := metav1.DeletePropagationForeground
	err := c.clientset.K8sClient.CoreV1().Pods(c.pod.Namespace).Delete(
		context.TODO(),
		c.pod.Name,
		metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		})
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}
	return nil
}

func (c *ContainerCreationExecContext) waitForPodToDelete() error {
	start := time.Now()
	for time.Since(start) <= deletionTimeout {
		pods, err := c.listPods(&metav1.ListOptions{})
		if err != nil {
			return err
		}
		found := false
		for _, pod := range pods.Items { //nolint:gocritic // This isn't my object I can't use a pointer
			if pod.Name == c.podName {
				found = true
			}
		}
		if !found {
			return nil
		}
		time.Sleep(time.Microsecond)
	}
	return errors.New("pod has not terminated within the timeout")
}

func (c *ContainerCreationExecContext) DeletePodAndWait() error {
	err := c.deletePod()
	if err != nil {
		return err
	}
	return c.waitForPodToDelete()
}

func NewContainerCreationExecContext(
	clientset *Clientset,
	namespace, podName, containerName, containerImage string,
	labels map[string]string,
	command []string,
	containerSecurityContext *corev1.SecurityContext,
	hostNetwork bool,
	volumes []*Volume,
) *ContainerCreationExecContext {
	ctx := ContainerExecContext{
		namespace:     namespace,
		podNamePrefix: podName,
		podName:       podName,
		containerName: containerName,
		clientset:     clientset,
	}

	return &ContainerCreationExecContext{
		ContainerExecContext:     &ctx,
		containerImage:           containerImage,
		labels:                   labels,
		command:                  command,
		containerSecurityContext: containerSecurityContext,
		hostNetwork:              hostNetwork,
		volumes:                  volumes,
	}
}

var clockClassRE = regexp.MustCompile(`\sclockClass\s+(\d+)`)
var promptRE = regexp.MustCompile(`(sh-\d.\d#\s*)`)

const shellCommand = "/usr/bin/sh"

type result struct {
	stdout string
	stderr string
	err    error
}

type command struct {
	*Command
	result chan *result
}
type Shell struct {
	expecter *expect.Console
	errBuff  bytes.Buffer
}

type ReusedConnectionContext struct {
	*ContainerExecContext
	shell          Shell
	commandChannel chan *command
	commandQuit    chan os.Signal
	wg             *utils.WaitGroupCount
}

//nolint:lll // allow slightly long function definition
func (c *ReusedConnectionContext) OpenShell(tty *os.File) error {
	logrus.Debugf(
		"execute command on ns=%s, pod=%s container=%s, cmd: %s",
		c.GetNamespace(),
		c.GetPodName(),
		c.GetContainerName(),
		shellCommand,
	)
	req := c.clientset.K8sRestClient.Post().
		Namespace(c.GetNamespace()).
		Resource("pods").
		Name(c.GetPodName()).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: c.GetContainerName(),
			Command:   []string{shellCommand},
			Stdin:     true,
			Stdout:    true,
			Stderr:    false,
			TTY:       true,
		}, scheme.ParameterCodec)

	// quit := make(chan os.Signal)
	exec, err := NewSPDYExecutor(c.clientset.RestConfig, "POST", req.URL())
	if err != nil {
		logrus.Debug(err)
		return fmt.Errorf("error setting up remote command: %w", err)
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
			Stdin:  tty,
			Stdout: tty,
			Stderr: nil,
			Tty:    true,
		})
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.commandQuit:
				logrus.Info("Quitting epector")
				c.shell.expecter.SendLine("quit")
				for len(c.commandChannel) > 0 {
					cmd := <-c.commandChannel
					c.handleCommand(cmd)
				}
				for x := c.wg.GetCount(); x > 1; x = c.wg.GetCount() {
					logrus.Infof("waiting for %d to finish", x)
					time.Sleep(time.Microsecond)
				}
				return
			case cmd := <-c.commandChannel:
				c.handleCommand(cmd)
			}
		}
	}()

	return nil
}

func (c ReusedConnectionContext) handleCommand(cmd *command) {
	logrus.Info("recived command")
	n, err := c.shell.expecter.SendLine(cmd.Stdin)
	logrus.Info("write command to stdin")
	if err != nil && n > 0 {
		e := fmt.Errorf("sent incomplete line: '%s', encountered error '%s'", cmd.Stdin[:n], err)
		logrus.Error(e)
	} else if err != nil {
		e := fmt.Errorf("could not run command: '%s', encountered error '%s'", cmd.Stdin, err)
		logrus.Error(e)
	}
	stdout, err := c.shell.expecter.Expect(expect.Regexp(cmd.regex))
	c.shell.expecter.Expect(expect.Regexp(promptRE))
	logrus.Info("waiting for prompt")
	stderr := c.shell.errBuff.String()
	logrus.Info("getting stderr")
	c.shell.errBuff.Reset()
	logrus.Info("clearing butter")
	cmd.result <- &result{stdout: stdout, stderr: stderr, err: err}
	logrus.Info("sent result")
}

func (c ReusedConnectionContext) execCommand(cmd *Command) (stdout, stderr string, err error) {
	logrus.Infof("attempting to run %s", cmd.Stdin)
	resChan := make(chan *result, 1)
	logrus.Info("made resp channel")
	c.commandChannel <- &command{Command: cmd, result: resChan}
	logrus.Info("sent command")
	resp := <-resChan
	logrus.Infof("resp %v", resp)
	return resp.stdout, resp.stderr, resp.err
}

//nolint:lll,funlen // allow slightly long function definition and allow a slightly long function
func (c ReusedConnectionContext) ExecCommand(cmd *Command) (stdout, stderr string, err error) {
	return c.execCommand(cmd)
}

//nolint:lll // allow slightly long function definition
func (c ReusedConnectionContext) ExecCommandStdIn(cmd *Command) (stdout, stderr string, err error) {
	return c.execCommand(cmd)
}

func (c *ReusedConnectionContext) CloseShell() {
	c.commandQuit <- os.Kill
	c.wg.Wait()
}

func NewReusedConnectionContext(
	clientset *Clientset,
	namespace, podNamePrefix, containerName string,
) (ReusedConnectionContext, error) {
	podName, err := clientset.FindPodNameFromPrefix(namespace, podNamePrefix)
	if err != nil {
		return ReusedConnectionContext{}, err
	}

	containerCtx, err := NewContainerContext(clientset, namespace, podName, containerName)
	if err != nil {
		return ReusedConnectionContext{}, err
	}

	w := logrus.StandardLogger().Writer()
	l := log.New(w, "", 0)

	expecter, err := expect.NewConsole(
		expect.WithDefaultTimeout(30*time.Second),
		expect.WithLogger(l),
	)
	if err != nil {
		return ReusedConnectionContext{}, err
	}

	ctx := ReusedConnectionContext{
		ContainerExecContext: containerCtx,
		shell: Shell{
			expecter: expecter,
		},
		commandChannel: make(chan *command, 10),
		commandQuit:    make(chan os.Signal, 1),
		wg:             &utils.WaitGroupCount{},
	}
	err = ctx.OpenShell(expecter.Tty())
	if err != nil {
		return ReusedConnectionContext{}, err
	}

	return ctx, nil
}
