package interfaces

type ErrorFinder interface {
	FindError(responseString string) error
}