package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	mongo_client "stockbackend/clients/mongo"
	"stockbackend/utils/helpers"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/extrame/xls"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/wailsapp/mimetype"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gopkg.in/mgo.v2/bson"
)

func assert(b bool, mess string) {
	green := "\033[32m"
	reset := "\033[0m"
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

func UpdateFunds() {
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
		// fmt.Println("Skipping the regular scheduler as debug mode is enabled.")
		// fmt.Println("Creating a scheduler that will run every 1 minute.")
		// jobID, err := scheduler.AddFunc("* * * * *", performUploadTask)
		performUploadTask()
		// Need this here for proper Next time calculation
		// scheduler.Start()
		// if err != nil {
		// 	// fmt.Println("An error occurred: the scheduler could not be added.")
		// } else {
		// 	// fmt.Println("Next run time for Debug Scheduler:", scheduler.Entry(jobID).Next)
		// }
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return
	}

	mfDatas := extractPortfolioLinks(string(body))

	for _, mfData := range mfDatas {
		uploadToCloudinary("https://mf.nipponindiaim.com/", mfData)
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
func normalizeWhitespace(s string) string {
	var b strings.Builder
	prevIsSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevIsSpace {
				b.WriteRune(' ')
				prevIsSpace = true
			}
		} else {
			b.WriteRune(r)
			prevIsSpace = false
		}
	}
	return b.String()
}

func removeZeroWidthChars(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\u200B', '\u200C', '\u200D', '\uFEFF':
			// Exclude zero-width characters
			return -1
		default:
			// Include other characters
			return r
		}
	}, s)
}

func cleanHTMLContent(s string) string {
	s = removeZeroWidthChars(s)
	s = normalizeWhitespace(s)
	return s
}

type MFCOLLECTION struct {
	month string
	year  string
	link  string
}

func extractPortfolioLinks(htmlContent string) []MFCOLLECTION {
	// Updated regex pattern to handle various formats
	re := regexp.MustCompile(`(?i)Monthly[\s\p{Zs}]+portfolio[\s\p{Zs}]+for[\s\p{Zs}]+the[\s\p{Zs}]+month(?:[\s\p{Zs}]+(?:of|end))?[\s\p{Zs}]*(?:(\d{1,2})(?:st|nd|rd|th)?[\s\p{Zs}]+)?(\w+)[\s\p{Zs}]*(\d{4})?.*?<a[^>]+href="([^"]+)"`)
	htmlContent = cleanHTMLContent(htmlContent)

	matches := re.FindAllStringSubmatch(htmlContent, -1)
	// fmt.Println("Total Matches Found:", len(matches)) // Debugging: Show total matches found

	var mfDetails []MFCOLLECTION
	for _, match := range matches {
		if len(match) > 4 {
			// entireText := match[0] // Entire matched text

			// Extract day, month, year, and link
			month := match[2] // Month
			year := match[3]  // Optional year
			link := match[4]  // Extracted link

			// If year is missing in match[3], try to extract it from the following content
			if year == "" {
				// Attempt to find a 4-digit year after the month
				yearRe := regexp.MustCompile(`\b(\d{4})\b`)
				yearMatch := yearRe.FindStringSubmatch(htmlContent)
				if len(yearMatch) > 1 {
					year = yearMatch[1]
				}
			}

			// Append the link
			mfDetails = append(mfDetails, MFCOLLECTION{
				month: month,
				year:  year,
				link:  link,
			})
			// // fmt.Println("Entire matched text:", entireText)
			// // fmt.Println("Month:", month) // Print extracted month
			// // fmt.Println("Year:", year)   // Print extracted year
			// // fmt.Println("Link:", link)   // Print the link
		}
	}
	return mfDetails
}

func uploadToCloudinary(fileURL string, mfData MFCOLLECTION) {
	cld, err := cloudinary.NewFromURL(os.Getenv("CLOUDINARY_URL"))
	if err != nil {
		log.Println("Error creating Cloudinary instance:", err)
		return
	}
	publicID := extractFileName(fileURL + mfData.link)
	asset, err := cld.Admin.Asset(context.Background(), admin.AssetParams{PublicID: publicID})

	secureUrl := asset.SecureURL
	if err == nil && asset.PublicID == "" {
		resp, err := cld.Upload.Upload(context.Background(), fileURL+mfData.link, uploader.UploadParams{
			PublicID: publicID,
		})
		if err != nil {
			log.Println("Error uploading to Cloudinary:", err)
			return
		}
		secureUrl = resp.SecureURL
	} else if err != nil {
		return
	}

	response, err := http.Get(secureUrl)
	if err != nil {
		log.Println("Error downloading xlsx file:", err)
		return
	}
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return
	}

	defer response.Body.Close()

	// Save the downloaded file to a temporary location
	m := mimetype.Detect(bodyBytes)
	// log.Println("Detected MIME type:", m.String())

	// Assign file extension based on MIME type
	var fileExt string
	if m.Is("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet") {
		fileExt = ".xlsx"
	} else if m.Is("application/vnd.ms-excel") {
		fileExt = ".xls"
	} else {
		// log.Println("Downloaded file is not a supported Excel format (.xlsx or .xls).")
		return
	}
	tempInputFile := fmt.Sprintf("%s%s", uuid.New().String(), fileExt)
	err = os.WriteFile(tempInputFile, bodyBytes, 0644)
	if err != nil {
		log.Println("Error saving Excel file:", err)
		return
	}
	defer os.Remove(tempInputFile)
	month := extractMonth(publicID)

	if fileExt == ".xlsx" {
		// Process .xlsx file
		err = processXLSXFile(tempInputFile, month)
		if err != nil {
			// fmt.Println("tempInputFile", tempInputFile)
			log.Println("Error processing .xlsx file:", err)
			return
		}
	} else if fileExt == ".xls" {
		// Process .xls file
		// fmt.Println("tempInputFile", tempInputFile)
		err = processXLSFile(tempInputFile, month)
		if err != nil {
			log.Println("Error processing .xls file:", err)
			return
		}
	} else {
		log.Println("Unsupported file format:", fileExt)
		return
	}

	fileUUID := uuid.New().String()
	document := bson.M{
		"_id":            fileUUID,
		"month":          month,
		"completeName":   publicID,
		"cloudinaryLink": secureUrl,
		"fund_house":     "nippon",
	}
	collection := mongo_client.Client.Database(os.Getenv("DATABASE")).Collection(os.Getenv("MFCOLLECTION"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = collection.InsertOne(ctx, document)
	if err != nil {
		log.Println("Error inserting document into MongoDB:", err)
		return
	}
	log.Printf("Document inserted successfully into MongoDB. UUID: %s\n", fileUUID)
}

func processXLSXFile(tempInputFile, month string) error {
	// Open the xlsx file using excelize
	xlsxFile, err := excelize.OpenFile(tempInputFile)
	if err != nil {
		// fmt.Println("Error opening xlsx file:", err)
		return fmt.Errorf("error opening xlsx file: %v", err)
	}
	defer xlsxFile.Close()

	// Get all sheet names
	sheetList := xlsxFile.GetSheetList()
	// stock := make([]map[string]interface{}, 0)

	for _, sheet := range sheetList {
		rows, err := xlsxFile.GetRows(sheet)
		if err != nil {
			sentry.CaptureException(err)
			zap.L().Error("Error reading rows from sheet", zap.String("sheet", sheet), zap.Error(err))
			continue
		}
		headerFound := false
		headerMap := make(map[string]int)
		stopExtracting := false
		stockDetail := make([]map[string]interface{}, 0)
		processingStarted := false
		mutualFundName := ""
		if len(rows) > 0 && len(rows[0]) > 1 {
			mutualFundName = rows[0][1]
			// log.Printf("Mutual Fund Name for sheet %s: %s", sheet, mutualFundName)
		}
		for _, row := range rows {
			stock := make(map[string]interface{}, 0)

			if len(row) == 0 {
				continue
			}
			if !headerFound {
				for _, cell := range row {
					if helpers.MatchHeader(cell, []string{`name\s*of\s*(the)?\s*instrument`}) {
						headerFound = true
						// Build the header map
						for i, headerCell := range row {
							normalizedHeader := helpers.NormalizeString(headerCell)
							// Map possible variations to standard keys
							switch {
							case helpers.MatchHeader(normalizedHeader, []string{`name\s*of\s*(the)?\s*instrument`}):
								headerMap["Name of the Instrument"] = i
							case helpers.MatchHeader(normalizedHeader, []string{`isin`}):
								headerMap["ISIN"] = i
							case helpers.MatchHeader(normalizedHeader, []string{`rating\s*/\s*industry`, `industry\s*/\s*rating`}):
								headerMap["Industry/Rating"] = i
							case helpers.MatchHeader(normalizedHeader, []string{`quantity`}):
								headerMap["Quantity"] = i
							case helpers.MatchHeader(normalizedHeader, []string{`market\s*/\s*fair\s*value.*`, `market\s*value.*`}):
								headerMap["Market/Fair Value"] = i
							case helpers.MatchHeader(normalizedHeader, []string{`%.*nav`, `%.*net\s*assets`}):
								headerMap["Percentage of AUM"] = i
							}
						}
						// zap.L().Info("Header found", zap.Any("headerMap", headerMap))
						break
					}
				}
				continue
			}

			joinedRow := strings.Join(row, "")
			if strings.Contains(strings.ToLower(joinedRow), "subtotal") || strings.Contains(strings.ToLower(joinedRow), "total") {
				stopExtracting = true
				break
			}
			if !processingStarted {
				nameOfInstrument := ""
				if idx, exists := headerMap["Name of the Instrument"]; exists && idx < len(row) {
					nameOfInstrument = row[idx]
				}
				if strings.Contains(nameOfInstrument, "Equity & Equity related") {
					processingStarted = true
					continue // Skip the header description row and move to the next row
				}
			}
			if processingStarted && !stopExtracting {
				for key, idx := range headerMap {
					if idx < len(row) {
						// println(stockDetail["Name of the Instrument"].(string), "Name of the Instrument in stockDetail")
						stock[key] = row[idx]
					} else {
						stock[key] = ""
					}
				}
				_, ok := stock["Name of the Instrument"].(string)
				if ok {
					stockDetail = append(stockDetail, stock)
					// println("Stock Detail: isin", stock["ISIN"].(string), "Name of the Instrument", stock["Name of the Instrument"].(string))
				}
			}
		}
		// println(mutualFundName, "Mutual Fund Name")
		hash := sha256.New()
		hashName := sha256.New()
		hashName.Write([]byte(mutualFundName))

		hash.Write([]byte(mutualFundName + month))
		hashedId := hex.EncodeToString(hash.Sum(nil))
		hashedName := hex.EncodeToString(hashName.Sum(nil))
		collection := mongo_client.Client.Database(os.Getenv("DATABASE")).Collection(os.Getenv("MFHOLDING"))

		validStockDetails := []map[string]interface{}{}

		for _, stockDetail := range stockDetail {
			if _, ok := stockDetail["Name of the Instrument"]; !ok {
				// fmt.Println("Skipping entry: Missing Name of the Instrument")
				continue
			}
			if _, ok := stockDetail["ISIN"]; !ok || stockDetail["ISIN"] == "" {
				// fmt.Println("Skipping entry: Missing ISIN")
				continue
			}
			if _, ok := stockDetail["Quantity"]; !ok {
				// // fmt.Println("Skipping entry: Missing Quantity")
				continue
			}
			if _, ok := stockDetail["Market/Fair Value"]; !ok {
				// // fmt.Println("Skipping entry: Missing Market/Fair Value")
				continue
			}
			if _, ok := stockDetail["Percentage of AUM"]; !ok {
				// // fmt.Println("Skipping entry: Missing Percentage of AUM")
				continue
			}
			validStockDetails = append(validStockDetails, stockDetail)
		}
		stockDetail = validStockDetails

		document := bson.M{
			"_id":              hashedId,
			"month":            month,
			"mutual_fund_name": mutualFundName,
			"stock_details":    stockDetail,
			"hash":             hashedName,
			"created_at":       time.Now(),
		}
		if mutualFundName == "" || stockDetail == nil || len(stockDetail) == 0 {
			// fmt.Println("Skipping empty document")
			continue
		}
		var existingDocument bson.M
		err = collection.FindOne(context.TODO(), bson.M{"_id": hashedId}).Decode(&existingDocument)
		if err == mongo.ErrNoDocuments {
			// Document does not exist, so insert it
			_, err := collection.InsertOne(context.TODO(), document)
			if err != nil {
				log.Fatal(err)
			}
			// fmt.Println("Document inserted with ID:", insertResult.InsertedID)
		} else if err != nil {
			log.Fatal(err)
		} else {
			// Document already exists
			// // fmt.Println("Document already exists, skipping insertion.")
		}

	}
	return nil
}

func safeGetRow(sheet *xls.WorkSheet, rowIndex int) (*xls.Row, bool) {
	defer func() {
		if r := recover(); r != nil {
			// log.Printf("Recovered from panic when accessing row %d: %v", rowIndex, r)
		}
	}()

	// Attempt to access the row
	row := sheet.Row(rowIndex)
	if row == nil {
		return nil, false
	}

	return row, true
}

// {
// 	"mutual_fund_name": "text",
// 	"stock_details.ISIN": "text",
// 	"stock_details.Name of the Instrument": "text"
//   }

func processXLSFile(tempInputFile, month string) error {
	xlsFile, err := xls.Open(tempInputFile, "utf-8")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}

	for sheetIndex := 0; sheetIndex < xlsFile.NumSheets(); sheetIndex++ {
		sheet := xlsFile.GetSheet(sheetIndex)
		if sheet == nil {
			// log.Printf("Sheet at index %d is nil. Skipping.", sheetIndex)
			continue
		}

		// log.Printf("Processing sheet: %s with MaxRow: %d", sheet.Name, sheet.MaxRow)

		headerFound := false
		headerMap := make(map[string]int)
		stopExtracting := false
		processingStarted := false
		stockDetail := make([]map[string]interface{}, 0)

		// Extract mutual fund name from the first row, second column
		var mutualFundName string
		if sheet.MaxRow > 0 {
			firstRow, ok := safeGetRow(sheet, 0)
			if !ok {
				continue
			}
			if firstRow != nil && firstRow.LastCol() > 1 {
				mutualFundName = firstRow.Col(1)
				// log.Printf("Mutual Fund Name for sheet %s: %s", sheet.Name, mutualFundName)
			}
		}

		for rowIndex := 0; rowIndex < int(sheet.MaxRow); rowIndex++ {
			if rowIndex >= int(sheet.MaxRow) {
				// log.Printf("Row %d in sheet %s is nil or out of range. Skipping.", rowIndex, sheet.Name)
				continue
			}
			// println("Row Index: ", rowIndex, int(sheet.MaxRow))
			row, ok := safeGetRow(sheet, rowIndex)
			if !ok || row.LastCol() == 0 {
				continue
			}

			// Detect headers if not already found
			if !headerFound {
				for colIndex := 0; colIndex < min(row.LastCol(), 10); colIndex++ { // Limit columns for header detection
					cellValue := strings.TrimSpace(strings.ToLower(row.Col(colIndex)))
					if strings.Contains(cellValue, "name of the instrument") {
						headerFound = true
						// Build the header map
						for i := 0; i < min(row.LastCol(), 10); i++ { // Limit column count to avoid extra empty columns
							header := strings.TrimSpace(strings.ToLower(row.Col(i)))
							switch {
							case strings.Contains(header, "name of the instrument"):
								headerMap["Name of the Instrument"] = i
							case strings.Contains(header, "isin"):
								headerMap["ISIN"] = i
							case strings.Contains(header, "rating/industry") || strings.Contains(header, "industry/rating"):
								headerMap["Industry/Rating"] = i
							case strings.Contains(header, "quantity"):
								headerMap["Quantity"] = i
							case strings.Contains(header, "market/fair value") || strings.Contains(header, "market value"):
								headerMap["Market/Fair Value"] = i
							case strings.Contains(header, "% nav") || strings.Contains(header, "% to nav") || strings.Contains(header, "% net assets"):
								headerMap["Percentage of AUM"] = i
							}
						}
						// log.Printf("Header found: %v", headerMap)
						break
					}
				}
				continue
			}

			// Stop extraction if "Subtotal" or "Total" is encountered
			var joinedRow string
			for colIndex := 0; colIndex < row.LastCol(); colIndex++ {
				joinedRow += row.Col(colIndex)
			}

			if strings.Contains(strings.ToLower(joinedRow), "subtotal") || strings.Contains(strings.ToLower(joinedRow), "total") {
				stopExtracting = true
				break
			}

			// Start processing only after "Equity & Equity related" is encountered
			if !processingStarted {
				nameOfInstrument := ""
				if idx, exists := headerMap["Name of the Instrument"]; exists && idx < row.LastCol() {
					nameOfInstrument = row.Col(idx)
				}
				if strings.Contains(nameOfInstrument, "Equity & Equity related") {
					processingStarted = true
					continue // Skip the header description row and move to the next row
				}
			}

			// Check if we need to adjust column indices for this row
			adjustColumns := false
			if headerMap["ISIN"] < row.LastCol() {
				isinValue := row.Col(headerMap["ISIN"])
				if !strings.HasPrefix(isinValue, "INE") { // Assuming valid ISINs start with "INE"
					// log.Printf("Adjusting row %d columns by 1 due to misalignment", rowIndex)
					adjustColumns = true
				}
			}

			// Data extraction based on header map with potential adjustment
			if processingStarted && !stopExtracting {
				stock := make(map[string]interface{})
				// log.Printf("Header map: %v", headerMap)
				for key, colIndex := range headerMap {
					adjustedColIndex := colIndex
					if adjustColumns {
						adjustedColIndex = colIndex + 1
					}

					if adjustedColIndex < row.LastCol() {
						cellValue := row.Col(adjustedColIndex)
						// log.Printf("Row %d, Key: %s, Expected Column: %d, Value: %s", rowIndex, key, adjustedColIndex, cellValue)
						cleanedValue := strings.ReplaceAll(cellValue, ",", "") // Remove commas

						// Handle percentage values
						if strings.HasSuffix(cleanedValue, "%") {
							stock[key] = cleanedValue
						} else if key == "Market/Fair Value" || key == "Quantity" {
							// Force parsing Market/Fair Value and Quantity as floats
							if parsedValue, err := strconv.ParseFloat(cleanedValue, 64); err == nil {
								stock[key] = parsedValue
								// log.Printf("Forced float parsing for key: %s, value: %v", key, parsedValue)
							} else {
								stock[key] = cellValue // Fallback to original if parsing fails
								// log.Printf("Failed to parse as float for key: %s, value: %s", key, cellValue)
							}
						} else {
							stock[key] = cellValue
						}
					} else {
						stock[key] = ""
					}
				}
				// println("*********************")
				// Skip rows without meaningful data in "Name of the Instrument"
				if stock["Name of the Instrument"] == "" {
					continue
				}

				// Append the stock detail to the list
				stockDetail = append(stockDetail, stock)
			}
		}
		// println(mutualFundName, "Mutual Fund Name")
		hash := sha256.New()
		hashName := sha256.New()
		hashName.Write([]byte(mutualFundName))

		hash.Write([]byte(mutualFundName + month))
		hashedId := hex.EncodeToString(hash.Sum(nil))
		hashedName := hex.EncodeToString(hashName.Sum(nil))
		collection := mongo_client.Client.Database(os.Getenv("DATABASE")).Collection(os.Getenv("MFHOLDING"))

		validStockDetails := []map[string]interface{}{}

		for _, stockDetail := range stockDetail {
			if _, ok := stockDetail["Name of the Instrument"]; !ok {
				// fmt.Println("Skipping entry: Missing Name of the Instrument")
				continue
			}
			if _, ok := stockDetail["ISIN"]; !ok || stockDetail["ISIN"] == "" {
				// fmt.Println("Skipping entry: Missing ISIN")
				continue
			}
			if _, ok := stockDetail["Quantity"]; !ok {
				// // fmt.Println("Skipping entry: Missing Quantity")
				continue
			}
			if _, ok := stockDetail["Market/Fair Value"]; !ok {
				// // fmt.Println("Skipping entry: Missing Market/Fair Value")
				continue
			}
			if _, ok := stockDetail["Percentage of AUM"]; !ok {
				// // fmt.Println("Skipping entry: Missing Percentage of AUM")
				continue
			}
			validStockDetails = append(validStockDetails, stockDetail)
		}
		stockDetail = validStockDetails

		document := bson.M{
			"_id":              hashedId,
			"month":            month,
			"mutual_fund_name": mutualFundName,
			"stock_details":    stockDetail,
			"hash":             hashedName,
			"created_at":       time.Now(),
		}
		if mutualFundName == "" || stockDetail == nil || len(stockDetail) == 0 {
			// fmt.Println("Skipping empty document")
			continue
		}
		var existingDocument bson.M
		err = collection.FindOne(context.TODO(), bson.M{"_id": hashedId}).Decode(&existingDocument)
		if err == mongo.ErrNoDocuments {
			// Document does not exist, so insert it
			_, err := collection.InsertOne(context.TODO(), document)
			if err != nil {
				log.Fatal(err)
			}
			// // fmt.Println("Document inserted with ID:", insertResult.InsertedID)
		} else if err != nil {
			log.Fatal(err)
		} else {
			// Document already exists
			// // fmt.Println("Document already exists, skipping insertion.")
		}

	}
	return nil
}

// Helper function to limit column count to avoid misalignment issues
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	fileName := path.Base(fileURL)
	return strings.TrimSuffix(fileName, path.Ext(fileName))
}
