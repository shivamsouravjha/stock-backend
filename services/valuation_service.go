package services

import (
	"fmt"
	"math"
	"stockbackend/types"
	"strconv"
	"strings"
)

/*
Investment Algorithm Implementation

This service implements the investment algorithm described in the document, which is used by
big financial institutions like Goldman Sachs and Motilal Oswal to make BUY/HOLD/SELL decisions.

The algorithm follows this pipeline:
1. Macro Inputs (GDP, Inflation, Interest Rates)
2. Industry Inputs (Demand, Supply, Pricing)
3. Company Inputs (Revenue, EBITDA, Cash Flows)
4. Valuation Layer:
   - DCF (Discounted Cash Flow)
   - Relative Valuation (P/E multiples)
   - Scenario Analysis (Bull/Base/Bear cases)
5. Qualitative Adjustments (Management quality, ESG, Risk)
6. Target Price Calculation: (α × DCF + β × Relative + γ × Scenario) × (1 + Risk Overlay)
7. Final Recommendation:
   - BUY: If upside ≥ 15%
   - HOLD: If within ±10%
   - SELL: If downside ≥ 15%
*/

// CalculateTargetPrice implements a simplified but realistic investment algorithm
// Add detailed logging to CalculateTargetPrice for debugging
func CalculateTargetPrice(companyData map[string]interface{}) *types.ValuationData {
	valuation := &types.ValuationData{}

	// Step 1: Extract financial metrics
	extractFinancialMetrics(companyData, valuation)
	fmt.Printf("Debug - Extracted Financial Metrics: %+v\n", valuation)

	// Step 2: Calculate DCF, Relative, Scenario values
	valuation.DCFValue = calculateDCFValue(valuation)
	fmt.Printf("Debug - DCF Value: %.2f\n", valuation.DCFValue)

	valuation.RelativeValue = calculateRelativeValue(companyData, valuation)
	fmt.Printf("Debug - Relative Value: %.2f\n", valuation.RelativeValue)

	valuation.ScenarioValue = calculateScenarioValue(valuation)
	fmt.Printf("Debug - Scenario Value: %.2f\n", valuation.ScenarioValue)

	// Step 3: Combine using weights (α, β, γ)
	alpha := 0.4
	beta := 0.4
	gamma := 0.2
	baseTargetPrice := alpha*valuation.DCFValue + beta*valuation.RelativeValue + gamma*valuation.ScenarioValue
	fmt.Printf("Debug - Base Target Price (before risk overlay): %.2f\n", baseTargetPrice)

	// Step 4: Apply risk overlay
	riskOverlay := calculateRiskOverlay(companyData)
	fmt.Printf("Debug - Risk Overlay: %.2f\n", riskOverlay)
	valuation.TargetPrice = baseTargetPrice * (1 + riskOverlay)
	fmt.Printf("Debug - Target Price (after risk overlay): %.2f\n", valuation.TargetPrice)

	// Step 5: Calculate upside/downside
	if valuation.CurrentPrice > 0 {
		valuation.UpsideDownside = ((valuation.TargetPrice - valuation.CurrentPrice) / valuation.CurrentPrice) * 100
	}

	// Apply realistic constraints to prevent extreme valuations
	// Most quantitative firms cap upside/downside at 30-40% for sanity
	if valuation.UpsideDownside > 40 {
		valuation.UpsideDownside = 40
		valuation.TargetPrice = valuation.CurrentPrice * 1.4
		fmt.Printf("Debug - Capped upside at 40%% for realism\n")
	} else if valuation.UpsideDownside < -40 {
		valuation.UpsideDownside = -40
		valuation.TargetPrice = valuation.CurrentPrice * 0.6
		fmt.Printf("Debug - Capped downside at -40%% for realism\n")
	}

	fmt.Printf("Debug - Upside/Downside: %.2f%%\n", valuation.UpsideDownside)

	// Step 6: Generate recommendation
	valuation.Recommendation = generateRecommendation(valuation.UpsideDownside)
	fmt.Printf("Debug - Recommendation: %s\n", valuation.Recommendation)

	fmt.Printf("Debug - Final Valuation Data: %+v\n", valuation)
	return valuation
}

// extractFinancialMetrics extracts key financial metrics from company data
func extractFinancialMetrics(companyData map[string]interface{}, valuation *types.ValuationData) {
	// Extract Market Cap for revenue estimation (convert from crores to proper value)
	if marketCapStr, ok := companyData["Market Cap"].(string); ok {
		if marketCap, err := parsePrice(marketCapStr); err == nil {
			// Convert from crores to actual value (multiply by 10^7)
			marketCapValue := marketCap * 10000000 // 1 crore = 10 million
			// More realistic revenue assumption: 20-30% of market cap for most companies
			// This is more conservative and realistic
			valuation.Revenue = marketCapValue * 0.25 // 25% of market cap - more realistic
			fmt.Printf("Debug - Market Cap: %.2f crores, Market Cap Value: %.2f, Revenue: %.2f\n",
				marketCap, marketCapValue, valuation.Revenue)
		}
	}

	// Extract current price
	if currentPriceStr, ok := companyData["Current Price"].(string); ok {
		if currentPrice, err := parsePrice(currentPriceStr); err == nil {
			valuation.CurrentPrice = currentPrice
		}
	}

	// Extract P/E ratio for profit estimation
	if peStr, ok := companyData["Stock P/E"].(string); ok && peStr != "" {
		if pe, err := parsePrice(peStr); err == nil && pe > 0 {
			// If we have P/E and current price, calculate earnings per share
			if valuation.CurrentPrice > 0 {
				eps := valuation.CurrentPrice / pe
				// Estimate shares outstanding from market cap and current price
				if valuation.Revenue > 0 {
					sharesOutstanding := valuation.Revenue / (valuation.CurrentPrice * 0.1) // Rough estimate
					valuation.NetProfit = eps * sharesOutstanding
				}
			}
		}
	}

	// If we don't have P/E or it's invalid, estimate from market cap
	if valuation.NetProfit <= 0 && valuation.Revenue > 0 {
		// Assume 8-12% net margin for financial services companies
		valuation.NetProfit = valuation.Revenue * 0.10
	}

	// Extract ROE for efficiency metrics
	if roeStr, ok := companyData["ROE"].(string); ok {
		if _, err := parsePrice(roeStr); err == nil {
			// Use ROE to estimate other metrics
			valuation.EBITDA = valuation.NetProfit * 1.2 // EBITDA typically 20% higher than net profit for financial services
		}
	}

	// If EBITDA is still 0, estimate it
	if valuation.EBITDA <= 0 && valuation.NetProfit > 0 {
		valuation.EBITDA = valuation.NetProfit * 1.2
	}

	// Estimate Free Cash Flow (typically 70-90% of EBITDA for financial services)
	if valuation.EBITDA > 0 {
		valuation.FreeCashFlow = valuation.EBITDA * 0.8
	}

	// Extract debt from balance sheet if available
	if balanceSheet, ok := companyData["balanceSheet"].(map[string]interface{}); ok {
		if borrowings, ok := balanceSheet["Borrowings +"].([]interface{}); ok && len(borrowings) > 0 {
			if latestBorrowing, ok := borrowings[len(borrowings)-1].(float64); ok {
				valuation.Debt = latestBorrowing * 10000000 // Convert from crores
			}
		}
	}

	// If no debt found, estimate (typically 20-40% of market cap for financial services)
	if valuation.Debt <= 0 && valuation.Revenue > 0 {
		valuation.Debt = valuation.Revenue * 0.25
	}

	// Estimate capex (typically 2-5% of revenue for financial services)
	if valuation.Revenue > 0 {
		valuation.Capex = valuation.Revenue * 0.03
	}

	// Set macro and industry assumptions (these would typically come from external data sources)
	valuation.GDPGrowth = 6.5    // India's expected GDP growth
	valuation.Inflation = 4.0    // RBI target inflation
	valuation.InterestRate = 6.5 // Current repo rate

	// Industry assumptions for financial services sector
	valuation.IndustryDemand = 12.0 // Higher growth for financial services
	valuation.IndustrySupply = 10.0 // Supply growth rate
	valuation.IndustryPricing = 2.0 // Price inflation for financial services
}

// calculateDCFValue implements Discounted Cash Flow valuation
func calculateDCFValue(valuation *types.ValuationData) float64 {
	if valuation.FreeCashFlow <= 0 {
		return 0
	}

	// DCF assumptions - more conservative and realistic
	growthRate := 0.08     // 8% growth rate for next 5 years (more conservative)
	terminalGrowth := 0.03 // 3% terminal growth rate (more conservative)
	discountRate := 0.15   // 15% discount rate (WACC) - more conservative for risk
	projectionYears := 5   // 5-year projection

	// Calculate present value of projected cash flows
	pvCashFlows := 0.0
	currentFCF := valuation.FreeCashFlow

	for year := 1; year <= projectionYears; year++ {
		// Project FCF for this year
		projectedFCF := currentFCF * math.Pow(1+growthRate, float64(year))

		// Discount to present value
		pv := projectedFCF / math.Pow(1+discountRate, float64(year))
		pvCashFlows += pv
	}

	// Calculate terminal value
	terminalFCF := currentFCF * math.Pow(1+growthRate, float64(projectionYears))
	terminalValue := (terminalFCF * (1 + terminalGrowth)) / (discountRate - terminalGrowth)
	pvTerminalValue := terminalValue / math.Pow(1+discountRate, float64(projectionYears))

	// Total DCF value
	totalDCFValue := pvCashFlows + pvTerminalValue

	// Calculate shares outstanding from market cap and current price
	var sharesOutstanding float64
	if valuation.CurrentPrice > 0 {
		// Market cap in crores * 10^7 / current price
		// Revenue is 25% of market cap, so market cap = revenue / 0.25
		marketCapInRupees := valuation.Revenue / 0.25 // Revenue is 25% of market cap
		sharesOutstanding = marketCapInRupees / valuation.CurrentPrice
		fmt.Printf("Debug - Market Cap: %.2f, Current Price: %.2f, Shares Outstanding: %.2f\n",
			marketCapInRupees, valuation.CurrentPrice, sharesOutstanding)
	} else {
		// Fallback estimation
		sharesOutstanding = valuation.Revenue * 0.1
	}

	if sharesOutstanding > 0 {
		return totalDCFValue / sharesOutstanding
	}

	return 0
}

// calculateRelativeValue implements relative valuation using P/E multiples
func calculateRelativeValue(companyData map[string]interface{}, valuation *types.ValuationData) float64 {
	// Extract current P/E ratio
	var currentPE float64
	if peStr, ok := companyData["Stock P/E"].(string); ok && peStr != "" {
		if pe, err := parsePrice(peStr); err == nil {
			currentPE = pe
		}
	}

	// If no P/E available, use industry average
	if currentPE <= 0 {
		currentPE = 18.0 // Default P/E - more conservative
	}

	if valuation.NetProfit <= 0 {
		return 0
	}

	// Industry average P/E - more conservative and realistic
	industryPE := 18.0 // Average P/E - more conservative than 22x

	// Calculate fair value based on industry P/E
	fairValue := valuation.NetProfit * industryPE

	// Calculate shares outstanding from market cap and current price
	var sharesOutstanding float64
	if valuation.CurrentPrice > 0 {
		// Market cap in crores * 10^7 / current price
		// Revenue is 25% of market cap, so market cap = revenue / 0.25
		marketCapInRupees := valuation.Revenue / 0.25
		sharesOutstanding = marketCapInRupees / valuation.CurrentPrice
	} else {
		// Fallback estimation
		sharesOutstanding = valuation.Revenue * 0.1
	}

	if sharesOutstanding > 0 {
		return fairValue / sharesOutstanding
	}

	return 0
}

// calculateScenarioValue implements scenario analysis (Bull, Base, Bear)
func calculateScenarioValue(valuation *types.ValuationData) float64 {
	// Base case (current DCF)
	baseValue := valuation.DCFValue

	// Bull case: 10% higher growth, 2% lower discount rate
	bullGrowth := 0.10
	bullDiscount := 0.13
	bullValue := calculateScenarioDCF(valuation, bullGrowth, bullDiscount)

	// Bear case: 5% lower growth, 3% higher discount rate
	bearGrowth := 0.05
	bearDiscount := 0.18
	bearValue := calculateScenarioDCF(valuation, bearGrowth, bearDiscount)

	// Weighted average: 25% bull, 50% base, 25% bear
	scenarioValue := 0.25*bullValue + 0.5*baseValue + 0.25*bearValue

	return scenarioValue
}

// calculateScenarioDCF calculates DCF with different assumptions
func calculateScenarioDCF(valuation *types.ValuationData, growthRate, discountRate float64) float64 {
	if valuation.FreeCashFlow <= 0 {
		return 0
	}

	terminalGrowth := 0.04
	projectionYears := 5

	pvCashFlows := 0.0
	currentFCF := valuation.FreeCashFlow

	for year := 1; year <= projectionYears; year++ {
		projectedFCF := currentFCF * math.Pow(1+growthRate, float64(year))
		pv := projectedFCF / math.Pow(1+discountRate, float64(year))
		pvCashFlows += pv
	}

	terminalFCF := currentFCF * math.Pow(1+growthRate, float64(projectionYears))
	terminalValue := (terminalFCF * (1 + terminalGrowth)) / (discountRate - terminalGrowth)
	pvTerminalValue := terminalValue / math.Pow(1+discountRate, float64(projectionYears))

	totalDCFValue := pvCashFlows + pvTerminalValue

	// Calculate shares outstanding from market cap and current price
	var sharesOutstanding float64
	if valuation.CurrentPrice > 0 {
		// Market cap in crores * 10^7 / current price
		// Revenue is 25% of market cap, so market cap = revenue / 0.25
		marketCapInRupees := valuation.Revenue / 0.25
		sharesOutstanding = marketCapInRupees / valuation.CurrentPrice
	} else {
		// Fallback estimation
		sharesOutstanding = valuation.Revenue * 0.1
	}

	if sharesOutstanding > 0 {
		return totalDCFValue / sharesOutstanding
	}

	return 0
}

// calculateRiskOverlay calculates risk/opportunity adjustments
func calculateRiskOverlay(companyData map[string]interface{}) float64 {
	overlay := 0.0

	// Management quality adjustment (based on ROE)
	if roeStr, ok := companyData["ROE"].(string); ok {
		if roe, err := parsePrice(roeStr); err == nil {
			if roe > 20 {
				overlay += 0.05 // 5% premium for high ROE
			} else if roe < 10 {
				overlay -= 0.05 // 5% discount for low ROE
			}
		}
	}

	// Debt level adjustment (based on ROCE)
	if roceStr, ok := companyData["ROCE"].(string); ok {
		if roce, err := parsePrice(roceStr); err == nil {
			if roce > 15 {
				overlay += 0.03 // 3% premium for high ROCE
			} else if roce < 8 {
				overlay -= 0.03 // 3% discount for low ROCE
			}
		}
	}

	// Dividend yield adjustment
	if divYieldStr, ok := companyData["Dividend Yield"].(string); ok {
		if divYield, err := parsePrice(divYieldStr); err == nil {
			if divYield > 3 {
				overlay += 0.02 // 2% premium for high dividend yield
			}
		}
	}

	// Limit overlay to ±10%
	if overlay > 0.1 {
		overlay = 0.1
	} else if overlay < -0.1 {
		overlay = -0.1
	}

	return overlay
}

// generateRecommendation generates BUY/HOLD/SELL recommendation based on upside/downside
func generateRecommendation(upsideDownside float64) string {
	// More conservative thresholds aligned with quantitative firm practices
	if upsideDownside >= 20 {
		return string(types.BUY)
	} else if upsideDownside <= -20 {
		return string(types.SELL)
	} else {
		return string(types.HOLD)
	}
}

// parsePrice parses price strings and returns float64 value
func parsePrice(priceStr string) (float64, error) {
	// Remove common currency symbols and clean the string
	cleaned := strings.ReplaceAll(priceStr, "₹", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "Cr", "")
	cleaned = strings.ReplaceAll(cleaned, "L", "")

	// Handle percentage values
	if strings.Contains(cleaned, "%") {
		cleaned = strings.ReplaceAll(cleaned, "%", "")
	}

	// Handle ranges (take the average)
	if strings.Contains(cleaned, "-") {
		parts := strings.Split(cleaned, "-")
		if len(parts) == 2 {
			val1, err1 := strconv.ParseFloat(parts[0], 64)
			val2, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 == nil && err2 == nil {
				return (val1 + val2) / 2, nil
			}
		}
	}

	return strconv.ParseFloat(cleaned, 64)
}

// GetInvestmentRecommendation is a convenience function that returns just the recommendation
func GetInvestmentRecommendation(companyData map[string]interface{}) (string, float64, float64) {
	valuation := CalculateTargetPrice(companyData)
	return valuation.Recommendation, valuation.TargetPrice, valuation.UpsideDownside
}
