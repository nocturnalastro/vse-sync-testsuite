// SPDX-License-Identifier: GPL-2.0-or-later

package callbacks_test

import (
	"bytes"
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/callbacks"
	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/clients"
)

func NewTestFile() *testFile {
	return &testFile{Buffer: *bytes.NewBuffer([]byte("")), open: true}
}

type testFile struct {
	bytes.Buffer
	open bool
}

func (t *testFile) Close() error {
	if t.open {
		t.open = false
		return nil
	}
	return errors.New("File is already closed") // TODO look up actual errors
}

var _ = Describe("Client", func() {
	var mockedFile *testFile
	var callback *callbacks.FileCallBack

	BeforeEach(func() {
		clients.ClearClientSet()
		mockedFile = NewTestFile()
		callback = &callbacks.FileCallBack{FileHandle: mockedFile}
	})

	When("A FileCallback is called", func() {
		It("should write to the file", func() {

			err := callback.Call("Test", "Nothing", "This is a test line")
			Expect(err).NotTo(HaveOccurred())
			Expect(mockedFile.ReadString('\n')).To(ContainSubstring("This is a test line"))
		})
	})
	When("A FileCallback is clened up", func() {
		It("should close the file", func() {
			err := callback.CleanUp()
			Expect(err).NotTo(HaveOccurred())
			Expect(mockedFile.open).To(BeFalse())
		})
	})

})

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Clients Suite")
}
