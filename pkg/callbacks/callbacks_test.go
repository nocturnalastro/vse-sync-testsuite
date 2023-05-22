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
