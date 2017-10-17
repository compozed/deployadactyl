package error_finder

type TrustStoreError struct{}

func (t TrustStoreError) Error() string {
	return "TrustStore error detected"
}
