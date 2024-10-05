package controllers

import (
	"stockbackend/services"

	"github.com/gin-gonic/gin"
)

type FileControllerI interface {
	ParseXLSXFile(ctx *gin.Context)
}

type fileController struct{}

var FileController FileControllerI = &fileController{}

func (f *fileController) ParseXLSXFile(ctx *gin.Context) {
	// Parse the form and retrieve the uploaded files
	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Error parsing form data"})
		return
	}

	// Retrieve the files from the form
	files := form.File["files"]
	if len(files) == 0 {
		ctx.JSON(400, gin.H{"error": "No files found"})
		return
	}

	// Set headers for chunked transfer (if needed)
	ctx.Writer.Header().Set("Content-Type", "text/plain")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")

	err = services.FileService.ParseXLSXFile(ctx, files)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	ctx.Writer.Write([]byte("\nStream complete.\n"))
	ctx.Writer.Flush() // Ensure the final response is sent
}
