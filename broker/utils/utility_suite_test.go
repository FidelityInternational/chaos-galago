package utils_test

import (
	. "github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/onsi/gomega"
	"testing"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utility test suite")
}
