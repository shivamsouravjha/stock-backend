package routes

import (
	"stockbackend/controllers"

	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {

	v1 := r.Group("/api")

	{
		v1.POST("/uploadXlsx", controllers.FileController.ParseXLSXFile)
		v1.GET("/keepServerRunning", controllers.HealthController.IsRunning)
		v1.POST("/fetchGmail", controllers.GmailController.GetEmails)
		v1.GET("/fetchStocks", controllers.StockController.GetStocks)
	}
}
