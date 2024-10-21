package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	mongo_client "stockbackend/clients/mongo"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"gopkg.in/mgo.v2/bson"
)

func assert(b bool, mess string) {
	red := "\033[31m"
	green := "\033[32m"
	reset := "\033[0m"
	if b {
		panic(red + "Assert FAILED: " + mess + reset)
	}
	if os.Getenv("DEBUG") == "true" {
		fmt.Println(green+"Assert PASSED: ", mess+reset)
	}
}

func setupCheck() {
	if len(os.Getenv("CLOUDINARY_URL")) < 5 {
		panic("Please provied a CLOUDINARY_URL. Run `export CLOUDINARY_URL=your@url` before in your shell for linux and MacOS")
	}
	if os.Getenv("DEBUG") != "true" {
		log.Printf("IGNORE THIS ONLY FOR DEV: To run the script in debug mode use `export DEBUG=\"true\"`")
	}
}

func main() {
	setupCheck()
	scheduler := cron.New()

	debug := os.Getenv("DEBUG") == "true"

	if !debug {
		// cron dose not support the L flag, for last day of the month!
		// For April, June, September, November
		_, err := scheduler.AddFunc("0 0 30 4,6,9,11 *", performUploadTask)
		// For January, March, May, July, August, October, December
		_, err2 := scheduler.AddFunc("0 0 31 1,3,5,7,8,10,12 *", performUploadTask)
		// For February
		_, err3 := scheduler.AddFunc("0 0 28 2 *", performUploadTask)
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
	} else {
		fmt.Println("Skipping the regular scheduler as debug mode is enabled.")
		fmt.Println("Creating a scheduler that will run every 1 minute.")
		jobID, err := scheduler.AddFunc("* * * * *", performUploadTask)

		// Need this here for proper Next time calculation
		scheduler.Start()
		if err != nil {
			fmt.Println("An error occurred: the scheduler could not be added.")
		} else {
			fmt.Println("Next run time for Debug Scheduler:", scheduler.Entry(jobID).Next)
		}
	}

	log.Println("Scheduler started")

	select {}
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
	assert(len(htmlContent) == 0, "extractPortfolioLinks len(htmlContent) == 0")

	re := regexp.MustCompile(`Monthly portfolio for the month end.*?<a[^>]+href="([^"]+)"`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)

	assert(len(matches) == 0, "extractPortfolioLinks len(matches) == 0")

	var links []string
	for _, match := range matches {
		if len(match) > 1 {
			links = append(links, match[1])
		}
	}

	assert(len(links) == 0, "extractPortfolioLinks len(links) == 0")
	return links
}

func uploadToCloudinary(fileURL string) {
	assert(len(fileURL) == 0, "uploadToCloudinary len(fileURL) == 0")

	cld, err := cloudinary.NewFromURL(os.Getenv("CLOUDINARY_URL"))
	if err != nil {
		log.Println("Error creating Cloudinary instance:", err)
		return
	}

	publicID := extractFileName(fileURL)

	resp, err := cld.Upload.Upload(context.Background(), fileURL, uploader.UploadParams{
		PublicID: publicID,
	})
	if err != nil {
		log.Println("Error uploading to Cloudinary:", err)
		return
	}

	log.Printf("File uploaded successfully: %s\n", resp.SecureURL)

	month := extractMonth(publicID)
	fileUUID := uuid.New().String()
	document := bson.M{
		"month":          month,
		"completeName":   publicID,
		"cloudinaryLink": resp.SecureURL,
		"id":             fileUUID,
	}

	collection := mongo_client.Client.Database(os.Getenv("DATABASE")).Collection(os.Getenv("COLLECTION"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = collection.InsertOne(ctx, document)
	if err != nil {
		log.Println("Error inserting document into MongoDB:", err)
		return
	}
	log.Printf("Document inserted successfully into MongoDB. UUID: %s\n", fileUUID)
}

func extractMonth(fileName string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\d{2}\.\d{2}\.\d{4}`),                                     // dd.mm.yyyy
		regexp.MustCompile(`\d{2}-\d{2}-\d{4}`),                                       // dd-mm-yyyy
		regexp.MustCompile(`\d{2}-\d{2}-\d{2}`),                                       // dd-mm-yy
		regexp.MustCompile(`\d{2}-\d{4}`),                                             // mm-yyyy
		regexp.MustCompile(`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)-\d{2}`), // Abbreviated month-year (e.g., Mar-23)
		regexp.MustCompile(`(January|February|March|April|May|June|July|August|September|October|November|December)-\d{4}`), // Full month-year (e.g., March-2021)
	}
	for _, pattern := range patterns {
		match := pattern.FindString(fileName)
		if match != "" {
			parsedDate := parseDate(match)
			if parsedDate != "" {
				return parsedDate
			}
		}
	}
	return ""
}

func parseDate(dateStr string) string {
	if strings.Contains(dateStr, ".") {
		t, err := time.Parse("02.01.2006", dateStr)
		if err == nil {
			return t.Format("2006-01-02")
		}
	}
	layouts := []string{
		"02-01-2006",   // dd-mm-yyyy
		"02-01-06",     // dd-mm-yy
		"01-2006",      // mm-yyyy
		"Jan-06",       // Abbreviated month-year
		"January-2006", // Full month-year
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, dateStr)
		if err == nil {
			// Format the date in YYYY-MM-DD format
			return t.Format("2006-01-02")
		}
	}

	// Handle month-year patterns (e.g., Mar-23, January-2021)
	if len(dateStr) == 7 || len(dateStr) == 10 {
		monthAbbrevToFull := map[string]string{
			"Jan": "January", "Feb": "February", "Mar": "March", "Apr": "April",
			"May": "May", "Jun": "June", "Jul": "July", "Aug": "August",
			"Sep": "September", "Oct": "October", "Nov": "November", "Dec": "December",
		}
		for abbr, full := range monthAbbrevToFull {
			if strings.Contains(dateStr, abbr) {
				dateStr = strings.Replace(dateStr, abbr, full, 1)
				t, err := time.Parse("January-06", dateStr)
				if err == nil {
					return t.Format("2006-01")
				}
			}
		}
	}
	return ""
}

func checkFileExistence(cld *cloudinary.Cloudinary, publicID string) (bool, error) {
	assert(cld == nil, "checkFileExistence cld == null")
	assert(publicID == "", "checkFileExistence publicId == \"\"")
	_, err := cld.Admin.Asset(context.Background(), admin.AssetParams{
		PublicID: publicID,
	})
	return !strings.Contains(err.Error(), "not found"), nil
}

func extractFileName(fileURL string) string {
	assert(fileURL == "", "extractFileName fileURL == \"\"")

	fileName := path.Base(fileURL)
	return strings.TrimSuffix(fileName, path.Ext(fileName))
}
