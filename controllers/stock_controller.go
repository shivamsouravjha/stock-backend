package controllers

import (
	"encoding/json"
	"fmt"
	"os"
	mongo_client "stockbackend/clients/mongo"
	"stockbackend/utils/helpers"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"gopkg.in/mgo.v2/bson"
)

type StockControllerI interface {
	GetStocks(ctx *gin.Context)
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
