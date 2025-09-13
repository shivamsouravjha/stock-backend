package controllers

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	mongo_client "stockbackend/clients/mongo"
	"stockbackend/services"
	"stockbackend/utils/helpers"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"gopkg.in/mgo.v2/bson"
)

type StockControllerI interface {
	GetStocks(ctx *gin.Context)
	UpdateCompanyData(ctx *gin.Context)
	GetInvestmentRecommendation(ctx *gin.Context)
	GetStocksWithRecommendations(ctx *gin.Context)
}

type stockController struct{}

var StockController StockControllerI = &stockController{}

func (s *stockController) GetStocks(ctx *gin.Context) {
	// parse the reuqest for pag number and last fetcehd stock fo r pagination
	pageNumberStr := ctx.DefaultQuery("pageNumber", "1")
	pageNumber, err := strconv.Atoi(pageNumberStr)
	if err != nil || pageNumber < 1 {
		ctx.JSON(400, gin.H{"error": "Invalid page number"})
		return
	}
	// fetch the stocks from the database
	collection := mongo_client.Client.Database(os.Getenv("DATABASE")).Collection(os.Getenv("COLLECTION"))

	//write  a code that does the limit annd the page size for the stocks
	findOptions := options.Find()
	findOptions.SetLimit(10)
	findOptions.SetSkip(int64(10 * (pageNumber - 1)))
	cursor, err := collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		zap.L().Error("Error while fetching documents", zap.Error(err))
		ctx.JSON(500, gin.H{"error": "Error while fetching stocks"})
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var result bson.M
		err := cursor.Decode(&result)
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Error while decoding stocks"})
			return
		}
		stockDetail := make(map[string]interface{})
		stockDetail["name"] = result["name"]
		stockDetail["marketCapValue"] = result["marketCap"]
		stockDetail["url"] = result["url"]
		stockDetail["marketCap"] = helpers.GetMarketCapCategory(fmt.Sprintf("%v", result["marketCap"]))
		stockDetail["stockRate"] = result["rank"]
		fmt.Println(result["fScore"], "fScore")
		stockDetail["fScore"] = result["fScore"]
		stockDataMarshal, err := json.Marshal(stockDetail)
		if err != nil {
			zap.L().Error("Error marshalling data", zap.Error(err))
			continue
		}

		_, err = ctx.Writer.Write(append(stockDataMarshal, '\n')) // Send each stockDetail as JSON with a newline separator

		if err != nil {
			zap.L().Error("Error writing data", zap.Error(err))
			break
		}
		ctx.Writer.Flush() // Flush each chunk immediately to the client
	}
	ctx.JSON(200, gin.H{"message": "Stocks are fetched"})
}

func (s *stockController) UpdateCompanyData(ctx *gin.Context) {
	zap.L().Info("Manual company data update triggered via API")

	// Run the company data update in a goroutine to avoid blocking the request
	go func() {
		services.UpdateCompanyData()
	}()

	ctx.JSON(200, gin.H{
		"message": "Company data update process started",
		"status":  "running",
	})
}

func (s *stockController) GetInvestmentRecommendation(ctx *gin.Context) {
	// Get company name from query parameter
	companyName := ctx.Query("company")
	if companyName == "" {
		ctx.JSON(400, gin.H{"error": "Company name is required"})
		return
	}

	// Fetch company data from database
	collection := mongo_client.Client.Database(os.Getenv("DATABASE")).Collection(os.Getenv("STOCK_COLLECTION"))

	filter := bson.M{"name": bson.M{"$regex": companyName, "$options": "i"}}
	var result bson.M
	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		zap.L().Error("Error finding company", zap.Error(err))
		ctx.JSON(404, gin.H{"error": "Company not found"})
		return
	}

	print(result, "result")
	// Extract URL and ID for updating
	url, urlExists := result["url"].(string)
	companyID := result["_id"]
	companyNameFromDB := result["name"].(string)

	if !urlExists || url == "" {
		ctx.JSON(400, gin.H{"error": "Company URL not found"})
		return
	}

	zap.L().Info("Fetching fresh data for company",
		zap.String("name", companyNameFromDB),
		zap.String("url", url),
		zap.Any("_id", companyID))

	// Fetch fresh company data from external source
	freshCompanyData, err := helpers.FetchCompanyData(url)
	if err != nil {
		zap.L().Error("Error fetching fresh company data",
			zap.String("company", companyNameFromDB),
			zap.String("url", url),
			zap.Error(err))
		ctx.JSON(500, gin.H{"error": "Failed to fetch fresh company data"})
		return
	}

	// Calculate target price and recommendation using fresh data
	valuation := services.CalculateTargetPrice(freshCompanyData)

	// Prepare update data with fresh information and new valuation
	updateData := bson.M{
		"$set": bson.M{
			"marketCap":           freshCompanyData["Market Cap"],
			"currentPrice":        freshCompanyData["Current Price"],
			"highLow":             freshCompanyData["High / Low"],
			"stockPE":             freshCompanyData["Stock P/E"],
			"bookValue":           freshCompanyData["Book Value"],
			"dividendYield":       freshCompanyData["Dividend Yield"],
			"roce":                freshCompanyData["ROCE"],
			"roe":                 freshCompanyData["ROE"],
			"faceValue":           freshCompanyData["Face Value"],
			"pros":                freshCompanyData["pros"],
			"cons":                freshCompanyData["cons"],
			"quarterlyResults":    freshCompanyData["quarterlyResults"],
			"profitLoss":          freshCompanyData["profitLoss"],
			"balanceSheet":        freshCompanyData["balanceSheet"],
			"cashFlows":           freshCompanyData["cashFlows"],
			"ratios":              freshCompanyData["ratios"],
			"shareholdingPattern": freshCompanyData["shareholdingPattern"],
			"peers":               freshCompanyData["peers"],
			// Updated valuation fields
			"targetPrice":    valuation.TargetPrice,
			"recommendation": valuation.Recommendation,
			"upsideDownside": valuation.UpsideDownside,
			"dcfValue":       valuation.DCFValue,
			"relativeValue":  valuation.RelativeValue,
			"scenarioValue":  valuation.ScenarioValue,
			"valuationData":  valuation,
			"lastUpdated":    time.Now(),
			"update_at":      time.Now(),
		},
	}

	// Update the document in database
	updateFilter := bson.M{"_id": companyID}
	updateOptions := options.Update().SetUpsert(false)

	updateResult, err := collection.UpdateOne(ctx, updateFilter, updateData, updateOptions)
	if err != nil {
		zap.L().Error("Error updating company data",
			zap.String("company", companyNameFromDB),
			zap.Any("_id", companyID),
			zap.Error(err))
		ctx.JSON(500, gin.H{"error": "Failed to update company data"})
		return
	}

	if updateResult.ModifiedCount == 0 {
		zap.L().Warn("No changes made to company data",
			zap.String("company", companyNameFromDB),
			zap.Any("_id", companyID))
	}

	zap.L().Info("Successfully updated company data with fresh information",
		zap.String("company", companyNameFromDB),
		zap.Any("_id", companyID),
		zap.Float64("targetPrice", valuation.TargetPrice),
		zap.String("recommendation", valuation.Recommendation),
		zap.Float64("upsideDownside", valuation.UpsideDownside))

	// Return the recommendation with updated data
	ctx.JSON(200, gin.H{
		"company":        companyNameFromDB,
		"currentPrice":   freshCompanyData["Current Price"],
		"targetPrice":    valuation.TargetPrice,
		"recommendation": valuation.Recommendation,
		"upsideDownside": valuation.UpsideDownside,
		"dcfValue":       valuation.DCFValue,
		"relativeValue":  valuation.RelativeValue,
		"scenarioValue":  valuation.ScenarioValue,
		"updated":        true,
		"message": fmt.Sprintf("Investment recommendation: %s with %.2f%% %s",
			valuation.Recommendation,
			math.Abs(valuation.UpsideDownside),
			func() string {
				if valuation.UpsideDownside >= 0 {
					return "upside"
				}
				return "downside"
			}()),
	})
}

func (s *stockController) GetStocksWithRecommendations(ctx *gin.Context) {
	// Parse the request for page number and last fetched stock for pagination
	pageNumberStr := ctx.DefaultQuery("pageNumber", "1")
	pageNumber, err := strconv.Atoi(pageNumberStr)
	if err != nil || pageNumber < 1 {
		ctx.JSON(400, gin.H{"error": "Invalid page number"})
		return
	}

	// Parse recommendation filter (optional)
	recommendationFilter := ctx.Query("recommendation") // BUY, SELL, HOLD

	// Fetch the stocks from the database
	collection := mongo_client.Client.Database(os.Getenv("DATABASE")).Collection(os.Getenv("STOCK_COLLECTION"))

	// Build filter based on recommendation if provided
	filter := bson.M{}
	if recommendationFilter != "" && (recommendationFilter == "BUY" || recommendationFilter == "SELL" || recommendationFilter == "HOLD") {
		filter["recommendation"] = recommendationFilter
	}

	// Write a code that does the limit and the page size for the stocks
	findOptions := options.Find()
	findOptions.SetLimit(10)
	findOptions.SetSkip(int64(10 * (pageNumber - 1)))

	// Sort by name for consistent pagination
	findOptions.SetSort(bson.M{"name": 1})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		zap.L().Error("Error while fetching documents", zap.Error(err))
		ctx.JSON(500, gin.H{"error": "Error while fetching stocks"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var result bson.M
		err := cursor.Decode(&result)
		if err != nil {
			zap.L().Error("Error while decoding stocks", zap.Error(err))
			ctx.JSON(500, gin.H{"error": "Error while decoding stocks"})
			return
		}

		// Create stock detail with investment recommendation data
		stockDetail := make(map[string]interface{})
		stockDetail["name"] = result["name"]
		stockDetail["marketCapValue"] = result["marketCap"]
		stockDetail["url"] = result["url"]
		stockDetail["marketCap"] = helpers.GetMarketCapCategory(fmt.Sprintf("%v", result["marketCap"]))
		stockDetail["stockRate"] = result["rank"]
		stockDetail["fScore"] = result["fScore"]

		// Add investment recommendation data
		stockDetail["currentPrice"] = result["currentPrice"]
		stockDetail["targetPrice"] = result["targetPrice"]
		stockDetail["recommendation"] = result["recommendation"]
		stockDetail["upsideDownside"] = result["upsideDownside"]
		stockDetail["dcfValue"] = result["dcfValue"]
		stockDetail["relativeValue"] = result["relativeValue"]
		stockDetail["scenarioValue"] = result["scenarioValue"]

		// Add last updated timestamp
		stockDetail["lastUpdated"] = result["lastUpdated"]

		// Marshal the stock detail to JSON
		stockDataMarshal, err := json.Marshal(stockDetail)
		if err != nil {
			zap.L().Error("Error marshalling data", zap.Error(err))
			continue
		}

		// Send each stockDetail as JSON with a newline separator
		_, err = ctx.Writer.Write(append(stockDataMarshal, '\n'))
		if err != nil {
			zap.L().Error("Error writing data", zap.Error(err))
			break
		}
		ctx.Writer.Flush() // Flush each chunk immediately to the client
	}

	ctx.JSON(200, gin.H{"message": "Stocks with investment recommendations fetched"})
}
