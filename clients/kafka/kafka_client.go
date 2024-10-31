package kafka_client

import (
	"context"
	"os"
	"stockbackend/types"
	"strconv"
	"time"

	"encoding/json"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

var (
	KafkaProducer    *kafka.Producer
	KafkaAdminClient *kafka.AdminClient
)

func SendMessage(event types.StockbackendEvent) {
	topic := os.Getenv("KAFKA_TOPIC")
	message, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}

	zap.L().Sugar().Infof("Sending message to kafka: %s", message)
	err = KafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(message),
	}, nil)
	if err != nil {
		zap.L().Error("Error sending message to kafka: ", zap.Any("error", err.Error()))
	}
}

func init() {
	logger, err := zap.NewProductionConfig().Build()
	if err != nil {
		panic("oh noes!")
	}

	zap.ReplaceGlobals(logger)

	zap.L().Info("KAFKA_BOOTSTRAPSERVERS: ", zap.String("uri", os.Getenv("KAFKA_BOOTSTRAPSERVERS")))

	NewProducer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BOOTSTRAPSERVERS"),
		"client.id":         "myProducer",
		"acks":              "all",
	})
	if err != nil {
		zap.L().Error("Kafka Producer initialization failed: ", zap.Any("error", err.Error()))
	}
	KafkaProducer = NewProducer

	NewAdminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BOOTSTRAPSERVERS"),
	})
	if err != nil {
		zap.L().Error("Kafka Producer initialization failed: ", zap.Any("error", err.Error()))
	}
	KafkaAdminClient = NewAdminClient

	// Delivery report handler for produced messages
	go func() {
		for e := range KafkaProducer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					zap.L().Error("Kafka Delivery failed: ", zap.Any("error", ev.TopicPartition.Error.Error()))
				} else {
					zap.L().Sugar().Infof("Delivered message to %s", *ev.TopicPartition.Topic)
				}
			}
		}
	}()

	KafkaProducer.Flush(50)

	topic := os.Getenv("KAFKA_TOPIC")
	numParts, err := strconv.Atoi(os.Getenv("KAFKA_TOPIC_PARTITIONS"))
	replicationFactor, err := strconv.Atoi(os.Getenv("KAFKA_TOPIC_REPL_FACTOR"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Create topics on cluster.
	// Set Admin options to wait for the operation to finish (or at most 60s)
	maxDur, err := time.ParseDuration("60s")
	if err != nil {
		panic("ParseDuration(60s)")
	}
	results, err := KafkaAdminClient.CreateTopics(
		ctx,
		// Multiple topics can be created simultaneously
		// by providing more TopicSpecification structs here.
		[]kafka.TopicSpecification{{
			Topic:             topic,
			NumPartitions:     numParts,
			ReplicationFactor: replicationFactor}},
		// Admin options
		kafka.SetAdminOperationTimeout(maxDur))
	if err != nil {
		zap.L().Error("Failed to create topic: ", zap.Any("error", err.Error()))
	}

	zap.L().Sugar().Infof("Connected to Kafka %s", results)
}
