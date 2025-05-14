package services

import (
	"stockbackend/types"
	"strconv"
	"strings"
)

func calculateMutualFundOverlap(mutualFund1, mutualFund2 []types.Instrument) (float64, float64) {
	// Map stocks by ISIN (case-insensitive) for efficient lookup
	stockMap1 := make(map[string]types.Instrument)
	for _, stock := range mutualFund1 {
		stockMap1[strings.ToUpper(stock.Isin)] = stock
	}

	stockMap2 := make(map[string]types.Instrument)
	for _, stock := range mutualFund2 {
		stockMap2[strings.ToUpper(stock.Isin)] = stock
	}
	fund1weightedOverlap, fund2weightedOverlap := 0.0, 0.0
	for isin, stock1 := range stockMap1 {
		if stock2, exists := stockMap2[isin]; exists {
			percentage1 := parsePercentage(stock1.Percentage)
			percentage2 := parsePercentage(stock2.Percentage)
			fund1weightedOverlap += percentage1
			fund2weightedOverlap += percentage2
		}
	}

	return fund1weightedOverlap, fund2weightedOverlap
}

func parsePercentage(percentageStr string) float64 {
	percentageStr = strings.ReplaceAll(percentageStr, "%", "")
	percentageStr = strings.ReplaceAll(percentageStr, ",", "")
	percentageStr = strings.TrimSpace(percentageStr)

	value, err := strconv.ParseFloat(percentageStr, 64)
	if err != nil {
		return 0.0
	}
	return value
}

func calculateOverlapPercentage(fund1, fund2 []types.Instrument) (float64, float64, []types.Instrument) {
	commonStocks := []types.Instrument{}
	fund1Map := make(map[string]struct{})
	for _, stock := range fund1 {
		fund1Map[strings.ToUpper(stock.Isin)] = struct{}{}
	}

	commonStocksCount := 0
	for _, stock := range fund2 {
		if _, exists := fund1Map[strings.ToUpper(stock.Isin)]; exists {
			commonStocks = append(commonStocks, stock)
			commonStocksCount++
		}
	}

	fund1OverlapPercentage := 0.0
	if len(fund1) > 0 {
		fund1OverlapPercentage = (float64(commonStocksCount) / float64(len(fund1))) * 100.0
	}

	fund2OverlapPercentage := 0.0
	if len(fund2) > 0 {
		fund2OverlapPercentage = (float64(commonStocksCount) / float64(len(fund2))) * 100.0
	}

	return fund1OverlapPercentage, fund2OverlapPercentage, commonStocks
}
