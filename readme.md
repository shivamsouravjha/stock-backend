
# Stock Analysis API

This project is an API built using the Gin framework for analyzing stock data. It processes Excel files, fetches stock data, calculates various stock metrics, and compares them with peer companies to derive useful insights.

## Features

- Upload Excel files to parse and analyze stock data.
- Analyze stock performance by comparing with peers.
- Fetch detailed financial data from external sources.
- Calculate a rating score for stocks based on various parameters.
- Implements a robust system to handle stock trend analysis, pros and cons evaluation, and peer comparison.
- Supports MongoDB for data storage and querying.
- Graceful shutdown and CORS middleware for stability and security.

## Requirements

- [Go 1.18+](https://golang.org/dl/)
- [MongoDB](https://www.mongodb.com/)
- [Gin Framework](https://github.com/gin-gonic/gin)
- [goquery](https://github.com/PuerkitoBio/goquery)
- [Excelize](https://github.com/xuri/excelize)
  
## Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/your-repo/stock-analysis-api.git
   cd stock-analysis-api
   ```

2. Install Go dependencies:
   ```bash
   go mod tidy
   ```

3. Set environment variables for MongoDB:
   ```bash
   export MONGO_URI="your_mongo_connection_string"
   export DATABASE="your_database_name"
   export COLLECTION="your_collection_name"
   export COMPANY_URL="your_company_api_url"
   ```

4. Run the API:
   ```bash
   go run main.go
   ```

   The API will run on `localhost:4000`.

## Endpoints

### Upload Stock Excel Data
- **Endpoint:** `/api/uploadXlsx`
- **Method:** `POST`
- **Description:** Upload one or more Excel files to parse stock data.
  
#### Request:
Upload Excel files through form data.

#### Response:
Returns parsed stock data along with calculated metrics in JSON format.

#### Example cURL:
```bash
curl -X POST http://localhost:4000/api/uploadXlsx   -F "files=@/path/to/your/excel_file.xlsx"
```

### Sample Stock Analysis Flow

1. **Upload XLSX file**: The file is parsed to extract stock information.
2. **Stock Processing**: Stock data is fetched from MongoDB or external APIs.
3. **Comparison**: Each stock is compared with its peers based on various financial parameters (e.g., PE, market cap).
4. **Trend Analysis**: The stock's quarterly performance is analyzed.
5. **Final Score**: A final score is calculated for the stock based on peer comparison, trend analysis, and other factors.

## MongoDB Integration

The application connects to MongoDB to query existing stock data and update stock information if necessary. It uses a combination of text search and structured querying to retrieve relevant information about stocks.

To set up MongoDB, ensure the following:

1. MongoDB is running locally or in the cloud.
2. Environment variables `MONGO_URI`, `DATABASE`, and `COLLECTION` are set.

## Key Components

- **Stock Structure**: Represents individual stock data with fields like PE, Market Cap, Dividend Yield, ROCE, etc.
- **Peer Comparison**: Calculates a peer score by comparing the stock’s financial metrics with peer companies.
- **Trend Analysis**: Evaluates historical performance trends for a stock.
- **CORS Middleware**: Provides CORS support for cross-origin requests.
- **Graceful Shutdown**: Ensures a smooth server shutdown when receiving termination signals.

## Libraries Used

- **[Gin](https://github.com/gin-gonic/gin)**: A lightweight, high-performance HTTP web framework.
- **[goquery](https://github.com/PuerkitoBio/goquery)**: Used for parsing and extracting data from HTML documents.
- **[Excelize](https://github.com/xuri/excelize)**: A Go library for reading and writing Excel files.
- **[MongoDB Driver](https://github.com/mongodb/mongo-go-driver)**: Official MongoDB driver for Go.

## Graceful Shutdown

This project handles system interrupts and shuts down the server gracefully using the following signal handlers:
- `SIGTERM`
- `SIGKILL`
- `SIGINT`

## Example Stock Rating Logic

The application rates stocks based on multiple factors:

- **PE Ratio**: Stocks with lower PE than peers score higher.
- **Market Cap**: Higher market cap results in a better score.
- **Dividend Yield**: Stocks with higher dividend yield outperform peers.
- **ROCE**: Return on Capital Employed is considered while rating.
- **Quarterly Performance**: Sales and profit growth are analyzed over time.

Example function for rating a stock:

```go
func rateStock(stock map[string]interface{}) float64 {
    // Logic to calculate final score
}
```

## Contribution

1. Fork the repository.
2. Create a new branch for your feature or bugfix.
3. Submit a pull request with detailed information about your changes.

