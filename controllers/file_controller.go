package controllers

import (
	"io"
	"os"
	"path/filepath"
	"stockbackend/services"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

type FileControllerI interface {
	ParseXLSXFile(ctx *gin.Context)
}

type fileController struct{}

var FileController FileControllerI = &fileController{}

func (f *fileController) ParseXLSXFile(ctx *gin.Context) {
	defer sentry.Recover()
	span := sentry.StartSpan(ctx.Request.Context(), "[GIN] ParseXLSXFile", sentry.WithTransactionName("ParseXLSXFile"))
	defer span.Finish()

	// Parse the form and retrieve the uploaded files
	form, err := ctx.MultipartForm()
	if err != nil {
		span.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		ctx.JSON(400, gin.H{"error": "Error parsing form data"})
		return
	}

	// Retrieve the files from the form
	files := form.File["files"]
	if len(files) == 0 {
		ctx.JSON(400, gin.H{"error": "No files found"})
		return
	}

	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		span.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		ctx.JSON(500, gin.H{"error": "Error creating upload directory"})
		return
	}
	var savedFilePaths = make(chan string, len(files))
	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			span.Status = sentry.SpanStatusFailedPrecondition
			sentry.CaptureException(err)
			ctx.JSON(500, gin.H{"error": "Error opening file"})
			return
		}
		defer src.Close()

		filename := filepath.Base(file.Filename)
		savePath := filepath.Join(uploadDir, filename)

		dst, err := os.Create(savePath)
		if err != nil {
			span.Status = sentry.SpanStatusFailedPrecondition
			sentry.CaptureException(err)
			ctx.JSON(500, gin.H{"error": "Error creating file on server"})
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			span.Status = sentry.SpanStatusFailedPrecondition
			sentry.CaptureException(err)
			ctx.JSON(500, gin.H{"error": "Error saving file"})
			return
		}

		savedFilePaths <- savePath
	}
	close(savedFilePaths)

	// Set headers for chunked transfer (if needed)
	ctx.Writer.Header().Set("Content-Type", "text/plain")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")

	err = services.FileService.ParseXLSXFile(ctx, savedFilePaths, span.Context())
	if err != nil {
		span.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	span.Status = sentry.SpanStatusOK
	if _, err := ctx.Writer.Write([]byte("\nStream complete.\n")); err != nil {
		ctx.JSON(500, gin.H{"error": "Error streaming"})
		return
	}

	ctx.Writer.Flush() // Ensure the final response is sent
}
