package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"stockbackend/services"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Body struct {
	Data string `json:"data"`
}

type Part struct {
	PartID   string `json:"partId"`
	MimeType string `json:"mimeType"`
	Body     Body   `json:"body"`
	Parts    []Part `json:"parts"`
	Filename string `json:"filename"`
}

type Payload struct {
	PartID   string   `json:"partId"`
	MimeType string   `json:"mimeType"`
	Body     Body     `json:"body"`
	Headers  []Header `json:"headers"`
	Parts    []Part   `json:"parts"`
}

type EmailDetails struct {
	Payload Payload `json:"payload"`
}

type Message struct {
	ID string `json:"id"`
}

type EmailList struct {
	Messages []Message `json:"messages"`
}

type GmailControllerI interface {
	GetEmails(ctx *gin.Context)
}

type gmailController struct{}

var GmailController GmailControllerI = &gmailController{}

func (g *gmailController) GetEmails(ctx *gin.Context) {
	defer sentry.Recover()
	transaction := sentry.TransactionFromContext(ctx)
	if transaction != nil {
		transaction.Name = "GetEmails"
	}

	sentrySpan := sentry.StartSpan(context.TODO(), "GetEmails")
	defer sentrySpan.Finish()

	accessToken := ctx.PostForm("token")
	sixMonthsAgo := time.Now().AddDate(0, -6, 0).Format("2006-01-02")

	url := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages?q=after:%s+portfolio+disclosure", sixMonthsAgo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating request: %v", err)})
		return
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error sending request: %v", err)})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error reading response body: %v", err)})
		return
	}

	if resp.StatusCode != http.StatusOK {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("API returned status code: %d, body: %s", resp.StatusCode, string(body))})
		return
	}

	var emailList EmailList
	if err := json.Unmarshal(body, &emailList); err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error parsing JSON: %v", err)})
		return
	}

	fileList := make(chan string)
	var wg sync.WaitGroup

	// Track all goroutines, including fetchEmailDetails and downloadFile
	for _, msg := range emailList.Messages {
		wg.Add(1)
		go func(messageID string) {
			defer wg.Done()
			fetchEmailDetails(accessToken, messageID, fileList, &wg, sentrySpan)
		}(msg.ID)
	}

	// Close the fileList channel once all goroutines have finished
	go func() {
		wg.Wait()
		close(fileList)
	}()

	// Process XLSX files
	err = services.FileService.ParseXLSXFile(ctx, fileList)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "Files processed successfully"})
}

func fetchEmailDetails(accessToken, emailID string, fileList chan<- string, wg *sync.WaitGroup, sentrySpan *sentry.Span) {
	url := "https://gmail.googleapis.com/gmail/v1/users/me/messages/" + emailID

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error creating request: %v", zap.Error(err))
		return
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		zap.L().Error("Error sending request: %v", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error reading response body: %v", zap.Error(err))
		return
	}
	var emailDetails EmailDetails
	if err := json.Unmarshal(body, &emailDetails); err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error parsing JSON: %v", zap.Error(err))
		return
	}

	emailBody := extractEmailBody(emailDetails.Payload, sentrySpan)

	if emailBody != "" {
		findRelevantLinks(emailBody, fileList, wg, sentrySpan)
	} else {
		zap.L().Info("No valid email body found.")
	}

}

func extractEmailBody(payload Payload, sentrySpan *sentry.Span) string {
	if payload.MimeType == "text/plain" || payload.MimeType == "text/html" {
		if payload.Body.Data != "" {
			decodedBody := decodeBase64URL(payload.Body.Data, sentrySpan)
			return decodedBody
		}
	}

	return extractFromParts(payload.Parts, sentrySpan)
}

func extractFromParts(parts []Part, sentrySpan *sentry.Span) string {
	for _, part := range parts {
		if (part.MimeType == "text/plain" || part.MimeType == "text/html") && part.Body.Data != "" {
			decodedBody := decodeBase64URL(part.Body.Data, sentrySpan)
			return decodedBody
		}
		if len(part.Parts) > 0 {
			nestedBody := extractFromParts(part.Parts, sentrySpan)
			if nestedBody != "" {
				return nestedBody
			}
		}
	}
	return ""
}

// Helper function to decode Base64URL encoded email content
func decodeBase64URL(data string, sentrySpan *sentry.Span) string {
	data = strings.ReplaceAll(data, "-", "+")
	data = strings.ReplaceAll(data, "_", "/")
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error decoding Base64URL data: %v", zap.Error(err))
		return ""
	}
	return string(decoded)
}
func findRelevantLinks(emailBody string, fileList chan<- string, wg *sync.WaitGroup, sentrySpan *sentry.Span) {
	kfintechLinkRegex := regexp.MustCompile(`https?://scdelivery\.kfintech\.com[^\s"<>]*`)
	kfintechLinks := kfintechLinkRegex.FindAllString(emailBody, -1)

	camsLinkRegex := regexp.MustCompile(`https?://delivery\.camsonline\.com[^\s"<>]*`)
	camsLinks := camsLinkRegex.FindAllString(emailBody, -1)

	for _, link := range kfintechLinks {
		wg.Add(1)
		go func(downloadURL string) {
			defer wg.Done()
			downloadFile(downloadURL, fileList, sentrySpan)
		}(link)
	}

	// Download CamsOnline files
	for _, link := range camsLinks {
		wg.Add(1)
		go func(downloadURL string) {
			defer wg.Done()
			downloadFile(downloadURL, fileList, sentrySpan)
		}(link)
	}
}

// Function to download the file from a link
func downloadFile(url string, fileList chan<- string, sentrySpan *sentry.Span) {
	resp, err := http.Get(url)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error downloading file", zap.String("url", url), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		zap.L().Error("Non ok-http status", zap.String("status", resp.Status), zap.Error(err))
		return
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error reading file", zap.String("url", url), zap.Error(err))
		return
	}

	// Generate a unique filename
	filename := uuid.New().String() + ".xlsx"
	err = os.WriteFile(filename, body, 0644)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error saving file", zap.String("filename", filename), zap.Error(err))
		return
	}
	// Send the filename to the fileList channel
	fileList <- filename
}
