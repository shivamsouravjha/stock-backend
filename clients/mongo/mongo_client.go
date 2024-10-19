package mongo_client

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"gopkg.in/mgo.v2/bson"
)

var (
	Client *mongo.Client
)

func init() {
	zap.L().Info("MONGO_URI: ", zap.String("uri", os.Getenv("MONGO_URI")))
	zap.L().Info("CLOUDINARY_URL", zap.String("uri", os.Getenv("CLOUDINARY_URL")))

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	mongoURI := os.Getenv("MONGO_URI")
	// zap.L().Info("Mongo URI", zap.String("uri", mongoURI))
	opts := options.Client().ApplyURI(mongoURI).SetServerAPIOptions(serverAPI)

	// Create a new client and connect to the server
	var err error // This is to ensure Client is not redeclared in the local scope
	Client, err = mongo.Connect(context.TODO(), opts)
	if err != nil {
		panic(err)
	}

	// Send a ping to confirm a successful connection
	pingCmd := bson.M{"ping": 1}
	if err := Client.Database("admin").RunCommand(context.TODO(), pingCmd).Err(); err != nil {
		panic(err)
	}

	zap.L().Info("Connected to MongoDB")
}
