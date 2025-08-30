package services

import (
	"context"
	"os"
	mongo_client "stockbackend/clients/mongo"
	"stockbackend/utils/helpers"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// UpdateCompanyData fetches fresh data for all companies in the database
// and updates their information while preserving the ID
func UpdateCompanyData() {
	zap.L().Info("Starting company data update process")

	collection := mongo_client.Client.Database(os.Getenv("DATABASE")).Collection(os.Getenv("STOCK_COLLECTION"))

	// Find all documents that have a URL field
	filter := bson.M{"url": bson.M{"$exists": true, "$ne": ""}}
	findOptions := options.Find()

	cursor, err := collection.Find(context.Background(), filter, findOptions)
	if err != nil {
		zap.L().Error("Error while fetching documents with URLs", zap.Error(err))
		return
	}
	defer cursor.Close(context.Background())

	updatedCount := 0
	errorCount := 0

	for cursor.Next(context.Background()) {
		var result bson.M
		err := cursor.Decode(&result)
		if err != nil {
			zap.L().Error("Error while decoding document", zap.Error(err))
			errorCount++
			continue
		}

		// Extract URL and ID
		url, urlExists := result["url"].(string)
		companyID, idExists := result["_id"]
		companyName, _ := result["name"].(string)

		if !urlExists || url == "" {
			zap.L().Warn("Skipping document without valid URL", zap.Any("_id", companyID))
			continue
		}

		if !idExists {
			zap.L().Warn("Skipping document without ID", zap.String("name", companyName))
			continue
		}

		zap.L().Info("Processing company",
			zap.String("name", companyName),
			zap.String("url", url),
			zap.Any("_id", companyID))

		// Fetch fresh company data
		companyData, err := helpers.FetchCompanyData(url)
		if err != nil {
			zap.L().Error("Error fetching company data",
				zap.String("company", companyName),
				zap.String("url", url),
				zap.Error(err))
			errorCount++
			continue
		}

		// Prepare update data - preserve ID and update all other fields
		updateData := bson.M{
			"$set": bson.M{
				"marketCap":           companyData["Market Cap"],
				"currentPrice":        companyData["Current Price"],
				"highLow":             companyData["High / Low"],
				"stockPE":             companyData["Stock P/E"],
				"bookValue":           companyData["Book Value"],
				"dividendYield":       companyData["Dividend Yield"],
				"roce":                companyData["ROCE"],
				"roe":                 companyData["ROE"],
				"faceValue":           companyData["Face Value"],
				"pros":                companyData["pros"],
				"cons":                companyData["cons"],
				"quarterlyResults":    companyData["quarterlyResults"],
				"profitLoss":          companyData["profitLoss"],
				"balanceSheet":        companyData["balanceSheet"],
				"cashFlows":           companyData["cashFlows"],
				"ratios":              companyData["ratios"],
				"shareholdingPattern": companyData["shareholdingPattern"],
				"peers":               companyData["peers"],
				"lastUpdated":         time.Now(),
				"update_at":           time.Now(),
			},
		}

		// Update the document by ID
		updateFilter := bson.M{"_id": companyID}
		updateOptions := options.Update().SetUpsert(false) // Don't create new documents

		updateResult, err := collection.UpdateOne(context.Background(), updateFilter, updateData, updateOptions)
		if err != nil {
			zap.L().Error("Error updating company data",
				zap.String("company", companyName),
				zap.Any("_id", companyID),
				zap.Error(err))
			errorCount++
			continue
		}

		if updateResult.ModifiedCount > 0 {
			updatedCount++
			zap.L().Info("Successfully updated company data",
				zap.String("company", companyName),
				zap.Any("_id", companyID))
		} else {
			zap.L().Warn("No changes made to company data",
				zap.String("company", companyName),
				zap.Any("_id", companyID))
		}

		// Add a small delay to avoid overwhelming the external API
		time.Sleep(1 * time.Second)
	}

	zap.L().Info("Company data update process completed",
		zap.Int("updated", updatedCount),
		zap.Int("errors", errorCount))
}
