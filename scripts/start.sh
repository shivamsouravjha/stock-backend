#!/bin/bash
set -e

export MONGO_URI="mongodb://mongodb:27017"
export DATABASE="stockbackend"
export COLLECTION="companies"

export KAFKA_BOOTSTRAPSERVERS="broker:9092"
export KAFKA_TOPIC="stockbackend"
export KAFKA_TOPIC_PARTITIONS="1"
export KAFKA_TOPIC_REPL_FACTOR="1"

export SENTRY_DSN=
export SENTRY_SAMPLE_RATE=1.0
export ENVIRONMENT=development
export TICKER_ENABLED=false

export COMPANY_URL="localhost:4000"


go run main.go