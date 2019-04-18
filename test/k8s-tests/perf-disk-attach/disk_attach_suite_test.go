package perf_disk_attach_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestPerfVMAttachApplications(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shoot VM attach and detach Performance Test Suite")
}
