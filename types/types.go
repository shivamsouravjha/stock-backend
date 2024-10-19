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
