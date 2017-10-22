package mocks

type ErrorFinder struct {
	FindErrorCall struct {
		Received struct {
			Response string
		}
		Returns struct {
			Error error
		}
	}
}

func (e *ErrorFinder) FindError(responseString string) error {
	e.FindErrorCall.Received.Response = responseString
	return e.FindErrorCall.Returns.Error
}
