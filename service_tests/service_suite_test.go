package service_test

import (
	"os"

	"github.com/compozed/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var (
	username string
	password string
)

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Suite")
}

var _ = BeforeSuite(func() {
	gin.SetMode(gin.TestMode)
	username = os.Getenv("CF_USERNAME")
	password = os.Getenv("CF_PASSWORD")
})

var _ = AfterSuite(func() {
	os.Setenv("CF_USERNAME", username)
	os.Setenv("CF_PASSWORD", password)
})
