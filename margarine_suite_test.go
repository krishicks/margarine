package margarine_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMargarine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Margarine Suite")
}
