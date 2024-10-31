#! /bin/bash
set -e

go mod tidy

mongorestore -v mongodb://mongodb:27017 mongo_backup/stockbackend/companies.bson
