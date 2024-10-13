package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/robfig/cron/v3"
)

func main() {
	scheduler := cron.New()

	// cron dose not support the L flag, for last day of the month!
	// For April, June, September, November
	_, err := scheduler.AddFunc("0 0 30 4,6,9,11 *", executeMonthlyTask)
	// For January, March, May, July, August, October, December
	_, err2 := scheduler.AddFunc("0 0 31 1,3,5,7,8,10,12 *", executeMonthlyTask)
	// For February
	_, err3 := scheduler.AddFunc("0 0 28 2 *", executeMonthlyTask)
	if err != nil {
		log.Fatal("Error scheduling task:", err)
	}
	if err2 != nil {
		log.Fatal("Error scheduling task:", err)
	}
	if err3 != nil {
		log.Fatal("Error scheduling task:", err)
	}

	scheduler.Start()

	log.Println("Scheduler started. Waiting for month-end...")

	select {}
}

func executeMonthlyTask() {
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)

	if now.Month() != tomorrow.Month() {
		log.Println("Month-end detected. Executing upload task...")
		performUploadTask()
	} else {
		log.Println("Not month-end. Skipping upload task.")
	}
}

func performUploadTask() {
	log.Println("Starting monthly upload task...")

	req, err := http.NewRequest("GET", "https://mf.nipponindiaim.com/investor-service/downloads/factsheet-portfolio-and-other-disclosures", nil)
	if err != nil {
		log.Println("Error creating request:", err)
		return
	}

	setRequestHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return
	}

	portfolioLinks := extractPortfolioLinks(string(body))

	for _, link := range portfolioLinks {
		uploadToCloudinary("https://mf.nipponindiaim.com/" + link)
	}

	log.Println("Monthly upload task completed.")
}

func setRequestHeaders(req *http.Request) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Sec-Ch-Ua", "\"Google Chrome\";v=\"129\", \"Not=A?Brand\";v=\"8\", \"Chromium\";v=\"129\"")
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", "\"macOS\"")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

func extractPortfolioLinks(htmlContent string) []string {
	re := regexp.MustCompile(`Monthly portfolio for the month end.*?]+href="([^"]+)"`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)

	var links []string
	for _, match := range matches {
		if len(match) > 1 {
			links = append(links, match[1])
		}
	}
	return links
}

func uploadToCloudinary(fileURL string) {
	cld, err := cloudinary.NewFromURL(os.Getenv("CLOUDINARY_URL"))
	if err != nil {
		log.Println("Error creating Cloudinary instance:", err)
		return
	}

	publicID := extractFileName(fileURL)

	exists, err := checkFileExistence(cld, publicID)
	if err != nil {
		log.Println("Error checking file existence:", err)
		return
	}

	if exists {
		log.Printf("File already exists on Cloudinary: %s\n", publicID)
		return
	}

	resp, err := cld.Upload.Upload(context.Background(), fileURL, uploader.UploadParams{
		PublicID: publicID,
	})
	if err != nil {
		log.Println("Error uploading to Cloudinary:", err)
		return
	}

	log.Printf("File uploaded successfully: %s\n", resp.SecureURL)
}

func checkFileExistence(cld *cloudinary.Cloudinary, publicID string) (bool, error) {
	_, err := cld.Admin.Asset(context.Background(), admin.AssetParams{
		PublicID: publicID,
	})
	return !strings.Contains(err.Error(), "not found"), nil
}

func extractFileName(fileURL string) string {
	fileName := path.Base(fileURL)
	return strings.TrimSuffix(fileName, path.Ext(fileName))
}
