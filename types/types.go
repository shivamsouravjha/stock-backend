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

type GeminiResponse struct {
	Name       string   `json:"name"`
	Percentage float64  `json:"percentage"`
	Strong     []string `json:"strong"`
	Error      string   `json:"error,omitempty"`
}

type GeminiRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	GenerationConfig map[string]interface{} `json:"generationConfig,omitempty"`
}

type Candidate struct {
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
}

type APIResponse struct {
	Candidates []Candidate `json:"candidates"`
}

type Instrument struct {
	Name        string `json:"name"`
	Isin        string `json:"isin"`
	Industry    string `json:"industry"`
	Quantity    string `json:"quantity"`
	MarketValue string `json:"marketValue"`
	Percentage  string `json:"percentage"`
}

type MutualFundData struct {
	MutualFundName string       `json:"mutualFundName"`
	FundData       []Instrument `json:"fundData"`
}

type MFInstrument struct {
	Name        string `json:"name"`
	Instruments []Instrument
}

type OverlapMutualFund struct {
	CommonStocks          []Instrument
	Fund1Percentage       string
	Fund2Percentage       string
	Fund1PercentageWeight string
	Fund2PercentageWeight string
}
