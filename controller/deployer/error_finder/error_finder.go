package error_finder

import "strings"

const TRUST_STORE_ERROR_STRING = "Creating TrustStore with container certificates\nFAILED"

type ErrorFinder struct {
}

func (e *ErrorFinder) FindError(responseString string) error {
	if strings.Contains(responseString, TRUST_STORE_ERROR_STRING) {
		return TrustStoreError{}
	}

	return nil
}
