package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RecoveryMiddleware catches panics and prevents the server from crashing
func RecoveryMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic and stack trace
				zap.L().Error("Panic recovered", zap.Any("panic", r), zap.String("stack", string(debug.Stack())))
				// Respond with a 500 Internal Server Error
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error. Please try again later.",
				})
				ctx.Abort()
			}
		}()
		// Continue to the next handler
		ctx.Next()
	}
}
