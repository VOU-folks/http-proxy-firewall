package interfaces

import (
	"github.com/gin-gonic/gin"
)

type FilterInterface interface {
	Handler(c *gin.Context) FilterResult
}

type FilterResult struct {
	Error        error
	AbortHandler func(c *gin.Context)
	Passed       bool
	BreakLoop    bool
}
