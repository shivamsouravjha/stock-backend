package services

import (
	"context"
	"os"
	mongo_client "stockbackend/clients/mongo"
	"stockbackend/utils/helpers"

	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"gopkg.in/mgo.v2/bson"
)

// write a function that travereses throguht al the documents of the mongo db
// and updates the rank of the stock
func UpdateRating() {
	collection := mongo_client.Client.Database(os.Getenv("DATABASE")).Collection(os.Getenv("COLLECTION"))
	findOptions := options.Find()
	cursor, err := collection.Find(context.Background(), bson.M{}, findOptions)
	if err != nil {
		zap.L().Error("Error while fetching documents", zap.Error(err))
	}
	defer cursor.Close(context.Background())
	var stockRate float64
	for cursor.Next(context.Background()) {
		var result bson.M
		err := cursor.Decode(&result)
		if err != nil {
			zap.L().Error("Error while decoding document", zap.Error(err))
		}
		stockRate = helpers.RateStock(result)
		fscore := helpers.GenerateFScore(result)
		_, err = collection.UpdateOne(context.Background(), bson.M{"_id": result["_id"]}, bson.M{"$set": bson.M{"rank": stockRate, "fScore": fscore}})
		if err != nil {
			zap.L().Error("Error while updating document", zap.Error(err))
		}
		zap.L().Info("Updated rank for stock", zap.Any("stock", result["name"]), zap.Any("rate", stockRate))
	}
}
