#!/bin/bash
set -e

export MONGO_URI="mongodb://mongodb:27017"
export DATABASE="stockbackend"
export COLLECTION="companies"
export COMPANY_URL="localhost:4000"

go run main.go