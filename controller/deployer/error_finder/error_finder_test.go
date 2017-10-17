package error_finder_test

import (
	. "github.com/compozed/deployadactyl/controller/deployer/error_finder"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"reflect"
)

var _ = Describe("ErrorFinder", func() {
	It("returns a TrustStoreError when the response to the user shows a trust store error", func(){
		response := "Creating TrustStore with container certificates\nFAILED"
		errorFinder := ErrorFinder{}
		err := errorFinder.FindError(response)
		Expect(reflect.TypeOf(err).String()).To(Equal("error_finder.TrustStoreError"))
	})

	It("returns a nil when it cannot detect the error", func(){
		errorFinder := ErrorFinder{}
		err := errorFinder.FindError("Good luck catching this error")
		Expect(err).To(BeNil())
	})
})
