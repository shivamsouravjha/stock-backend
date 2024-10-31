package rabbitmq_client

import (
	"fmt"
	"os"
	"stockbackend/types"

	"encoding/json"

	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

var (
	Connection *amqp.Connection
	Channel    *amqp.Channel
	Queue      amqp.Queue
)

// GetEnv retrieves the environment variable with a default value if not set.
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func Close() {
	Channel.Close()
	Connection.Close()
}

func SendMessage(event types.StockbackendEvent) {
	message, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}

	zap.L().Sugar().Infof("Sending message to rabbitmq: %s", message)

	// 4. Publish a message to the queue
	err = Channel.Publish(
		"",         // Exchange (empty means default)
		Queue.Name, // Routing key (queue name in this case)
		false,      // Mandatory
		false,      // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(message),
		})

	if err != nil {
		zap.L().Error("Error publishing message to rabbitmq: ", zap.Any("error", err.Error()))
		return
	}

	zap.L().Info("Successfully sent message to rabbitmq.")
}

func init() {
	logger, err := zap.NewProductionConfig().Build()
	if err != nil {
		panic("oh noes!")
	}

	zap.ReplaceGlobals(logger)

	// 1. Connect to RabbitMQ server
	rabbitServer := GetEnv("RABBITMQ_SERVER", "localhost")
	rabbitPort := GetEnv("RABBITMQ_PORT", "5672")
	rabbitUser := GetEnv("RABBITMQ_USER", "guest")
	rabbitPass := GetEnv("RABBITMQ_PASS", "guest")

	zap.L().Sugar().Infof("RabbitMQ Server: %s", rabbitServer)
	zap.L().Sugar().Infof("RabbitMQ Port: %s", rabbitPort)
	zap.L().Sugar().Infof("RabbitMQ User: %s", rabbitUser)
	zap.L().Sugar().Debugf("RabbitMQ Pass: %s", rabbitPass)

	newConn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitUser, rabbitPass, rabbitServer, rabbitPort))
	if err != nil {
		zap.L().Error("RabbitMQ initialization failed: ", zap.Any("error", err.Error()))
	}
	Connection = newConn

	// 2. Create a channel
	ch, err := Connection.Channel()
	if err != nil {
		zap.L().Error("RabbitMQ - Failed to open a channel: ", zap.Any("error", err.Error()))
	}

	Channel = ch

	// 3. Declare a queue to ensure it exists before publishing messages
	queueName := "stockbackend"
	q, err := ch.QueueDeclare(
		queueName, // Name of the queue
		true,      // Durable
		false,     // Delete when unused
		false,     // Exclusive
		false,     // No-wait
		nil,       // Arguments
	)
	if err != nil {
		zap.L().Error("RabbitMQ - Failed to declare a queue: ", zap.Any("error", err.Error()))
	}

	Queue = q

	zap.L().Info("Connected to RabbitMQ.")
}
