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
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

func main() {
	req, err := http.NewRequest("GET", "https://mf.nipponindiaim.com/investor-service/downloads/factsheet-portfolio-and-other-disclosures", nil)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// Use regex to find all href links for "Monthly portfolio for the month end"
	re := regexp.MustCompile(`Monthly portfolio for the month end.*?<a[^>]+href="([^"]+)"`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	var links []string
	for _, match := range matches {
		if len(match) > 1 {
			links = append(links, match[1])
		}
	}

	// upload the downloaded file to cloudinary
	for _, link := range links {
		uploadFileToCloudinary("https://mf.nipponindiaim.com/" + link)
	}
}

func uploadFileToCloudinary(fileURL string) {
	// Initialize Cloudinary
	cloudinaryURL := os.Getenv("CLOUDINARY_URL")
	cld, err := cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		fmt.Println("Error creating Cloudinary instance:", err)
		return
	}

	// Extract the file name from the file URL
	publicID := getFileNameFromURL(fileURL)

	// Check if the file already exists
	exists, err := checkIfFileExists(cld, publicID)
	if err != nil {
		fmt.Println("Error checking file existence:", err)
		return
	}

	if exists {
		fmt.Printf("File already exists on Cloudinary: %s\n", publicID)
		return
	}

	// Upload the file if it doesn't exist
	resp, err := cld.Upload.Upload(context.Background(), fileURL, uploader.UploadParams{
		PublicID: publicID,
	})
	if err != nil {
		fmt.Println("Error uploading to Cloudinary:", err)
		return
	}
	fmt.Printf("File uploaded successfully: %s\n", resp.SecureURL)
}

func checkIfFileExists(cld *cloudinary.Cloudinary, publicID string) (bool, error) {
	e, _ := cld.Admin.Asset(context.Background(), admin.AssetParams{
		PublicID: publicID,
	})
	log.Println(e.Error.Message)
	if strings.Contains(e.Error.Message, "not found") {
		return false, nil
	}
	return true, nil
}

func getFileNameFromURL(fileURL string) string {
	fileName := path.Base(fileURL)
	fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
	return fileName
}
