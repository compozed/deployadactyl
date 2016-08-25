package interfaces

import "github.com/compozed/gin"

// Endpoints interface.
type Endpoints interface {
	Deploy(c *gin.Context)
}
