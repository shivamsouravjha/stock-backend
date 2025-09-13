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

// ValuationData represents the comprehensive valuation data for a company
type ValuationData struct {
	// Macro Inputs
	GDPGrowth    float64 `json:"gdpGrowth"`
	Inflation    float64 `json:"inflation"`
	InterestRate float64 `json:"interestRate"`

	// Industry Inputs
	IndustryDemand  float64 `json:"industryDemand"`
	IndustrySupply  float64 `json:"industrySupply"`
	IndustryPricing float64 `json:"industryPricing"`

	// Company Inputs
	Revenue      float64 `json:"revenue"`
	EBITDA       float64 `json:"ebitda"`
	NetProfit    float64 `json:"netProfit"`
	FreeCashFlow float64 `json:"freeCashFlow"`
	Debt         float64 `json:"debt"`
	Capex        float64 `json:"capex"`

	// Valuation Results
	DCFValue       float64 `json:"dcfValue"`
	RelativeValue  float64 `json:"relativeValue"`
	ScenarioValue  float64 `json:"scenarioValue"`
	TargetPrice    float64 `json:"targetPrice"`
	CurrentPrice   float64 `json:"currentPrice"`
	Recommendation string  `json:"recommendation"` // BUY, HOLD, SELL
	UpsideDownside float64 `json:"upsideDownside"` // percentage
}

// RecommendationType represents the investment recommendation
type RecommendationType string

const (
	BUY  RecommendationType = "BUY"
	HOLD RecommendationType = "HOLD"
	SELL RecommendationType = "SELL"
)
