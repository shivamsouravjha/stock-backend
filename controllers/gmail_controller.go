package controllers

import (
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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Part struct {
	MimeType string `json:"mimeType"`
	Body     struct {
		Data string `json:"data"`
	} `json:"body"`
	Filename string `json:"filename"`
}

type Payload struct {
	Headers []Header `json:"headers"`
	Parts   []Part   `json:"parts"`
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

func (g *gmailController) GetEmails(c *gin.Context) {
	accessToken := c.PostForm("token")
	url := "https://gmail.googleapis.com/gmail/v1/users/me/messages?q=portfolio+disclosure"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating request: %v", err)})
		return
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error sending request: %v", err)})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error reading response body: %v", err)})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("API returned status code: %d, body: %s", resp.StatusCode, string(body))})
		return
	}

	var emailList EmailList
	if err := json.Unmarshal(body, &emailList); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error parsing JSON: %v", err)})
		return
	}

	fileList := make(chan string)
	var wg sync.WaitGroup

	// Track all goroutines, including fetchEmailDetails and downloadFile
	for _, msg := range emailList.Messages {
		wg.Add(1)
		go func(messageID string) {
			defer wg.Done()
			fetchEmailDetails(accessToken, messageID, fileList, &wg)
		}(msg.ID)
	}

	// Close the fileList channel once all goroutines have finished
	go func() {
		wg.Wait()
		close(fileList)
	}()

	// Process XLSX files
	err = services.FileService.ParseXLSXFile(c, fileList)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Files processed successfully"})
}

func fetchEmailDetails(accessToken, emailID string, fileList chan<- string, wg *sync.WaitGroup) {
	url := "https://gmail.googleapis.com/gmail/v1/users/me/messages/" + emailID

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
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
		zap.L().Error("Error reading response body: %v", zap.Error(err))
		return
	}

	var emailDetails EmailDetails
	if err := json.Unmarshal(body, &emailDetails); err != nil {
		zap.L().Error("Error parsing JSON: %v", zap.Error(err))
		return
	}

	for _, header := range emailDetails.Payload.Headers {
		if header.Name == "From" {
			break
		}
	}

	for _, part := range emailDetails.Payload.Parts {
		if part.MimeType == "text/plain" || part.MimeType == "text/html" {
			emailBody := decodeBase64URL(part.Body.Data)
			findXLSXLinks(emailBody, fileList, wg)
		}
	}
}

// Helper function to decode Base64URL encoded email content
func decodeBase64URL(data string) string {
	data = strings.ReplaceAll(data, "-", "+")
	data = strings.ReplaceAll(data, "_", "/")
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		zap.L().Error("Error decoding Base64URL data: %v", zap.Error(err))
		return ""
	}
	return string(decoded)
}

func findXLSXLinks(emailBody string, fileList chan<- string, wg *sync.WaitGroup) {
	// Regular expression to find .xlsx and CamsOnline links
	xlsxLinkRegex := regexp.MustCompile(`https?://[^\s]+\.xlsx`)
	camsLinkRegex := regexp.MustCompile(`https?://delivery\.camsonline\.com[^\s]*`)

	// Find .xlsx links
	xlsxLinks := xlsxLinkRegex.FindAllString(emailBody, -1)
	camsLinks := camsLinkRegex.FindAllString(emailBody, -1)

	// Download .xlsx files
	for _, link := range xlsxLinks {
		wg.Add(1)
		go func(downloadURL string) {
			defer wg.Done()
			downloadFile(downloadURL, fileList)
		}(link)
	}

	// Download CamsOnline files
	for _, link := range camsLinks {
		wg.Add(1)
		go func(downloadURL string) {
			defer wg.Done()
			downloadFile(downloadURL, fileList)
		}(link)
	}
}

// Function to download the file from a link
func downloadFile(url string, fileList chan<- string) {
	resp, err := http.Get(url)
	if err != nil {
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
		zap.L().Error("Error reading file", zap.String("url", url), zap.Error(err))
		return
	}

	// Generate a unique filename
	filename := uuid.New().String() + ".xlsx"
	err = os.WriteFile(filename, body, 0644)
	if err != nil {
		zap.L().Error("Error saving file", zap.String("filename", filename), zap.Error(err))
		return
	}
	// Send the filename to the fileList channel
	fileList <- filename
}
