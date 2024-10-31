package types

// Stock represents the data of a stock
type Stock struct {
	Name            string
	PE              float64
	MarketCap       float64
	DividendYield   float64
	ROCE            float64
	QuarterlySales  float64
	QuarterlyProfit float64
	Cons            []string
	Pros            []string
}

type Company struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type StockbackendEvent struct {
	EventType     string                 `json:"eventType"`
	EventId       string                 `json:"eventId"`
	Timestamp     string                 `json:"timestamp"`
	Data          map[string]interface{} `json:"data"`
	CorrelationId string                 `json:"correlationId"`
}
