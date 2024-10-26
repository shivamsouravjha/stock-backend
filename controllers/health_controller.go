package controllers

import "github.com/gin-gonic/gin"

type HealthControllerI interface {
	IsRunning(ctx *gin.Context)
}

type healthController struct{}

var HealthController HealthControllerI = &healthController{}

func (h *healthController) IsRunning(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Server is running"})
}
