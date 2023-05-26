// SPDX-License-Identifier: GPL-2.0-or-later

package clients_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeK8s "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-testsuite/testutils"
	"github.com/redhat-partner-solutions/vse-sync-testsuite/testutils/mocks"
)

var notATestPod = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:        "NotATestPod-3989",
		Namespace:   "TestNamespace",
		Annotations: map[string]string{},
	},
}
var testPod = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:        "TestPod-8292",
		Namespace:   "TestNamespace",
		Annotations: map[string]string{},
	},
}

var _ = Describe("NewContainerContext", func() {
	var clientset *clients.Clientset
	BeforeEach(func() {
		clients.ClearClientSet()
		clientset = clients.GetClientset(kubeconfigPath)
	})

	When("A ContainerContext is requested for a pod which DOES NOT exist", func() {
		It("should return an error ", func() {
			fakeK8sClient := fakeK8s.NewSimpleClientset(notATestPod)
			clientset.K8sClient = fakeK8sClient

			_, err := clients.NewContainerContext(clientset, "TestNamespace", "Test", "TestContainer")
			Expect(err).To(HaveOccurred())
		})
	})
	When("A ContainerContext is requested for a pod which DOES exist", func() {
		It("should return the context for that pod", func() {
			fakeK8sClient := fakeK8s.NewSimpleClientset(notATestPod, testPod)
			clientset.K8sClient = fakeK8sClient

			ctx, err := clients.NewContainerContext(clientset, "TestNamespace", "Test", "TestContainer")
			Expect(err).NotTo(HaveOccurred())
			Expect(ctx.GetNamespace()).To(Equal("TestNamespace"))
			Expect(ctx.GetContainerName()).To(Equal("TestContainer"))
			Expect(ctx.GetPodName()).To(Equal("TestPod-8292"))
		})
	})

})

var _ = Describe("ExecCommandContainer", func() {
	var clientset *clients.Clientset
	BeforeEach(func() {
		clientset = testutils.GetMockedClientSet(testPod)
	})

	When("given a pod", func() {
		It("should exec the command and return the std buffers", func() {
			expectedStdOut := "my test command stdout"
			expectedStdErr := "my test command stderr"
			// responder := func(method string, url *url.URL) ([]byte, []byte, error) {
			// 	return []byte(expectedStdOut), []byte(expectedStdErr), nil
			// }
			mockCtrl := gomock.NewController(GinkgoT())
			defer mockCtrl.Finish()
			mockExec := mocks.NewMockExecutor(mockCtrl)
			mockExec.
				EXPECT().
				StreamWithContext(gomock.Any(), gomock.Any()).
				Return(nil).
				Do(func(ctx context.Context, options remotecommand.StreamOptions) {
					_, _ = options.Stdout.Write([]byte(expectedStdOut))
					_, _ = options.Stderr.Write([]byte(expectedStdErr))
				}).Times(1)

			clients.NewSPDYExecutor = testutils.NewMockedNewSPDYExecutor(mockCtrl, mockExec, nil)
			ctx, _ := clients.NewContainerContext(clientset, "TestNamespace", "Test", "TestContainer")
			cmd := []string{"my", "test", "command"}
			stdout, stderr, err := clientset.ExecCommandContainer(ctx, cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(Equal(expectedStdOut))
			Expect(stderr).To(Equal(expectedStdErr))
		})
	})

	//nolint:dupl //it is incorrectly saying that this is a duplicate despite the aguments being in a different order
	When("NewSPDYExecutor fails", func() {
		It("should return an error", func() {
			expectedStdOut := ""
			expectedStdErr := ""
			expectedErr := errors.New("Something went horribly wrong when creating the executor")
			mockCtrl := gomock.NewController(GinkgoT())
			defer mockCtrl.Finish()
			mockExec := mocks.NewMockExecutor(mockCtrl)
			clients.NewSPDYExecutor = testutils.NewMockedNewSPDYExecutor(mockCtrl, mockExec, expectedErr)
			ctx, _ := clients.NewContainerContext(clientset, "TestNamespace", "Test", "TestContainer")
			cmd := []string{"my", "test", "command"}
			stdout, stderr, err := clientset.ExecCommandContainer(ctx, cmd)
			Expect(err).To(HaveOccurred())
			Expect(expectedErr.Error()).To(ContainSubstring(expectedErr.Error()))
			Expect(stdout).To(Equal(expectedStdOut))
			Expect(stderr).To(Equal(expectedStdErr))
		})
	})
	//nolint:dupl //it is incorrectly saying that this is a duplicate despite the aguments being in a different order
	When("SteamWithContext fails", func() {
		It("should return an error", func() {
			expectedStdOut := ""
			expectedStdErr := ""
			expectedErr := errors.New("Something went horribly wrong with the stream")
			mockCtrl := gomock.NewController(GinkgoT())
			defer mockCtrl.Finish()
			mockExec := mocks.NewMockExecutor(mockCtrl)
			mockExec.
				EXPECT().
				StreamWithContext(gomock.Any(), gomock.Any()).
				Return(expectedErr)
			clients.NewSPDYExecutor = testutils.NewMockedNewSPDYExecutor(mockCtrl, mockExec, nil)
			ctx, _ := clients.NewContainerContext(clientset, "TestNamespace", "Test", "TestContainer")
			cmd := []string{"my", "test", "command"}
			stdout, stderr, err := clientset.ExecCommandContainer(ctx, cmd)
			Expect(err).To(HaveOccurred())
			Expect(expectedErr.Error()).To(ContainSubstring(expectedErr.Error()))
			Expect(stdout).To(Equal(expectedStdOut))
			Expect(stderr).To(Equal(expectedStdErr))
		})
	})
})
