package controllers

import (
	"bytes"
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

	sentrySpan := sentry.StartSpan(ctx.Request.Context(), "GetEmails", sentry.WithTransactionName("GetEmails"))
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
	err = services.FileService.ParseXLSXFile(ctx, fileList, sentrySpan.Context())
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
	var subject string
	for _, header := range emailDetails.Payload.Headers {
		if header.Name == "Subject" {
			subject = header.Value
		}
	}

	emailBody := extractEmailBody(emailDetails.Payload, sentrySpan)

	if emailBody != "" {
		if strings.Contains(strings.ToUpper(subject), "SBI") {
			camsLinkRegex := regexp.MustCompile(`ext=([^&]+)`)
			matches := camsLinkRegex.FindStringSubmatch(emailBody)
			if len(matches) < 2 {
				zap.L().Error("No 'ext' parameter found in the URL", zap.String("emailBody", emailBody))
				return
			}
			ext := matches[1]

			downloadLink := getSBIDownloadLink(ext)

			if downloadLink != "" {
				downloadXLSXFile(downloadLink, fileList, sentrySpan)
			}
			return
		}

		findRelevantLinks(emailBody, fileList, wg, sentrySpan)
	} else {
		zap.L().Info("No valid email body found.")
	}

}
func downloadXLSXFile(downloadURL string, fileList chan<- string, sentrySpan *sentry.Span) {
	client := &http.Client{}
	resp, err := client.Get(downloadURL)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error downloading .xlsx file", zap.String("url", downloadURL), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		zap.L().Error("Non-OK HTTP status for .xlsx download", zap.String("status", resp.Status), zap.String("url", downloadURL))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error reading .xlsx file", zap.String("url", downloadURL), zap.Error(err))
		return
	}

	filename := uuid.New().String() + ".xlsx"
	err = os.WriteFile(filename, body, 0644)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error saving .xlsx file", zap.String("filename", filename), zap.Error(err))
		return
	}

	fileList <- filename
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

func parseQueryString(query string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(query, "&")
	for _, part := range parts {
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) == 2 {
			result[keyValue[0]] = keyValue[1]
		}
	}
	return result
}
func getSBIDownloadLink(ext string) string {
	client := &http.Client{}
	decodedExt, _ := base64.StdEncoding.DecodeString(ext)

	queryMap := parseQueryString(string(decodedExt))
	fundID := queryMap["FundID"]
	portfoliotype := queryMap["Portfoliotype"]
	if fundID == "" || portfoliotype == "" {
		zap.L().Error("Missing FundID or Portfoliotype in decoded ext", zap.String("ext", ext))
		return ""
	}

	postData := map[string]string{
		"FundId":      fundID,
		"PSFrequency": portfoliotype,
	}
	jsonData, err := json.Marshal(postData)
	if err != nil {
		zap.L().Error("Error marshaling POST data", zap.Error(err))
		return ""
	}

	req, err := http.NewRequest("POST", "https://www.sbimf.com/ajaxcall/CMS/GetSchemePortfolioSheetsQS", bytes.NewBuffer(jsonData))
	if err != nil {
		zap.L().Error("Error creating POST request", zap.Error(err))
		return ""
	}

	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		zap.L().Error("Error sending POST request", zap.Error(err))
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		zap.L().Error("Error reading POST response", zap.Error(err))
		return ""
	}

	downloadLink := strings.Trim(strings.TrimSpace(string(body)), "\"")
	return downloadLink
}

// Function to download the file from a link
func downloadFile(url string, fileList chan<- string, sentrySpan *sentry.Span) {
	client := &http.Client{
		CheckRedirect: nil,
	}
	resp, err := client.Get(url)

	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error downloading file", zap.String("url", url), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		redirectURL := resp.Header.Get("Location")
		if redirectURL == "" {
			zap.L().Error("Redirect with no location header", zap.String("url", url))
			return
		}

		zap.L().Info("Redirect found", zap.String("redirectURL", redirectURL))

		resp, err = client.Get(redirectURL)
		if err != nil {
			sentry.CaptureException(err)
			zap.L().Error("Error following redirect", zap.String("url", redirectURL), zap.Error(err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			zap.L().Error("Non-OK HTTP status after redirect", zap.String("status", resp.Status), zap.String("url", redirectURL))
			return
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error reading file", zap.String("url", url), zap.Error(err))
		return
	}

	filename := uuid.New().String() + ".xlsx"
	err = os.WriteFile(filename, body, 0644)
	if err != nil {
		sentrySpan.Status = sentry.SpanStatusFailedPrecondition
		sentry.CaptureException(err)
		zap.L().Error("Error saving file", zap.String("filename", filename), zap.Error(err))
		return
	}

	fileList <- filename
}
