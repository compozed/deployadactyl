package error_finder_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestErrorFinder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ErrorFinder Suite")
}
