package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"stockbackend/types"
	"strings"
	"sync"
)

var once sync.Once

var GEMINI_API_URL = ""
var GEMINI_API_KEY = ""

func init() {
	once.Do(func() {
		GEMINI_API_URL = os.Getenv("GEMINI_API_URL")
		GEMINI_API_KEY = os.Getenv("GEMINI_API_KEY")
	})
}

func CallGeminiAPI(sheet [][]string) types.MutualFundData {
	if len(sheet) == 0 {
		return types.MutualFundData{}
	}
	prompt := fmt.Sprintf(`
	You are a data processing assistant. Your task is to convert mutual fund portfolio data into JSON.
First, identify the mutual fund name which typically appears:
- At the top of the document as a title/heading
- In formats like "[Company] Mutual Fund", "Portfolio of [Fund Name]", or "[Fund Type] Fund"
- Often includes fund house names like HDFC, SBI, ICICI, Aditya Birla, etc.

Extract this name BEFORE processing the holdings data. If multiple potential fund names exist, choose the most complete one.

Convert each portfolio holding row into a JSON object with these fields:
- name: Name of the instrument
- isin: ISIN code
- industry: Rating/Industry
- quantity: Quantity
- marketValue: Market value (Rs. In lakhs)
- percentage: %% to Net Assets

Return ONLY this JSON structure:
{
  "nameOfMutualFund": "[EXTRACTED FUND NAME]",
  "fundData": [
    {
      "name": "...",
      "isin": "...",
      "industry": "...",
      "quantity": "...",
      "marketValue": "...",
      "percentage": "..."
    },
    ...
  ]
}

The sheet data is:
%s
`, sheet)
	requestData := types.GeminiRequest{
		Contents: []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{
			{
				Parts: []struct {
					Text string `json:"text"`
				}{
					{
						Text: prompt,
					},
				},
			},
		},
		GenerationConfig: map[string]interface{}{
			"maxOutputTokens": 200000,
		},
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return types.MutualFundData{}
	}

	apiEndpoint := GEMINI_API_URL + "?key=" + GEMINI_API_KEY
	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return types.MutualFundData{}
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return types.MutualFundData{}
	}
	defer resp.Body.Close()

	var rawResponse types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&rawResponse)
	if err != nil {
		return types.MutualFundData{}
	}

	content := rawResponse.Candidates[0].Content
	if len(content.Parts) > 0 {
		part := content.Parts[0]
		generatedText := part.Text
		cleanedText := strings.TrimPrefix(generatedText, "```json")
		cleanedText = strings.TrimSuffix(cleanedText, "```")
		cleanedText = strings.TrimSpace(cleanedText)
		//convert cleanedText to array Instrument
		var mutualFundData types.MutualFundData
		err = json.Unmarshal([]byte(cleanedText), &mutualFundData)
		if err != nil {
			return types.MutualFundData{}
		}

		sanitisedMutualFundData := types.MutualFundData{}
		for _, fundData := range mutualFundData.FundData {
			if strings.Contains(fundData.Isin, "IN") && len(fundData.Isin) == 12 {
				sanitisedMutualFundData.FundData = append(sanitisedMutualFundData.FundData, fundData)
			}
		}
		return sanitisedMutualFundData
	}
	return types.MutualFundData{}
}

func CallGeminiAPI2(sheet []string) string {
	if len(sheet) != 2 {
		return `{"error": "Expected two sheet data strings."}`
	}
	sheet1Str := sheet[0]
	sheet2Str := sheet[1]

	prompt := fmt.Sprintf(`You are a data processing assistant. Your task is to analyze the overlap of stocks between two mutual fund holdings provided below.
	
	The format for each mutual fund holding is a CSV structure:
	Name,ISIN,Industry,Quantity,Market Value (Rs. In lakhs),%% to Net Assets
	[... rows of stock data ...]
	
	Mutual Fund 1 Data:
	
	%s
	
	Mutual Fund 2 Data:
	%s
	
	Identify the common stocks (based on ISIN) present in both mutual funds. For these common stocks, also list their percentage to net assets in both funds. Finally, calculate the percentage of overlap in terms of the number of common stocks relative to the total number of unique stocks in each mutual fund.
	
	The output should be a JSON object with the following structure:
	{
	  "commonStocks": [
		{
		  "isin": "[ISIN of common stock]",
		  "nameMF1": "[Name of common stock in MF1]",
		  "nameMF2": "[Name of common stock in MF2]",
		  "percentageMF1": "[%% to Net Assets in MF1]",
		  "percentageMF2": "[%% to Net Assets in MF2]"
		},
		{
		  "...": "..."
		}
	  ],
	  "overlapPercentageMF1": "[Percentage of overlap for MF1 (common stocks / total unique MF1 stocks) as a string]",
	  "overlapPercentageMF2": "[Percentage of overlap for MF2 (common stocks / total unique MF2 stocks) as a string]"
	}
	
	Ensure the output is a valid JSON object.`, sheet1Str, sheet2Str)

	requestData := types.GeminiRequest{
		Contents: []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{
			{
				Parts: []struct {
					Text string `json:"text"`
				}{
					{
						Text: prompt,
					},
				},
			},
		},
		GenerationConfig: map[string]interface{}{
			"maxOutputTokens": 200000,
		},
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return ""
	}

	apiEndpoint := GEMINI_API_URL + "?key=" + GEMINI_API_KEY
	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return ""
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var rawResponse types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&rawResponse)
	if err != nil {
		return ""
	}

	content := rawResponse.Candidates[0].Content
	if len(content.Parts) > 0 {
		part := content.Parts[0]
		generatedText := part.Text
		cleanedText := strings.TrimPrefix(generatedText, "```json")
		cleanedText = strings.TrimSuffix(cleanedText, "```")
		cleanedText = strings.TrimSpace(cleanedText)
		return cleanedText
	}
	return ""
}

func CallGeminiAPI3(sheet []types.MFInstrument) string {
	if len(sheet) < 2 {
		return `{"error": "Expected two sheet data strings."}`
	}
	sheet1Str := sheet[0]
	sheet2Str := sheet[1]

	prompt := fmt.Sprintf(`You are a data processing assistant. Your task is to analyze the overlap between two mutual fund portfolios.

	First, extract the names of both mutual funds from the data provided.
	
	For each fund's holdings:
	1. Parse each row to extract ISIN, name, and percentage to net assets
	2. Treat ISINs as case-insensitive unique identifiers
	
	Calculate two overlap metrics:
	1. COUNT OVERLAP: (Number of common stocks / Average number of stocks in both funds) * 100
	2. WEIGHT OVERLAP: Sum of the MINIMUM percentage between common holdings across both funds


Return this JSON structure:
{
  "mutualFund2": "[Name of second mutual fund]",
  "overlapSummary": {
    "countOverlapPercentage": "XX.XX",
    "weightOverlapPercentage": "XX.XX" 
  },
  "commonHoldings": [
    {
      "isin": "[ISIN]",
      "nameMF1": "[Name in MF1]",
      "nameMF2": "[Name in MF2]",
      "percentageMF1": "X.XX",
      "percentageMF2": "X.XX",
      "minPercentage": "X.XX"
    }
  ]
}Mutual Fund 1 Data:
%s

Mutual Fund 2 Data:
%s
`, sheet1Str, sheet2Str)
	requestData := types.GeminiRequest{
		Contents: []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{
			{
				Parts: []struct {
					Text string `json:"text"`
				}{
					{
						Text: prompt,
					},
				},
			},
		},
		GenerationConfig: map[string]interface{}{
			"maxOutputTokens": 200000,
		},
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return ""
	}

	apiEndpoint := GEMINI_API_URL + "?key=" + GEMINI_API_KEY
	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return ""
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var rawResponse types.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&rawResponse)
	if err != nil {
		return ""
	}

	content := rawResponse.Candidates[0].Content
	if len(content.Parts) > 0 {
		part := content.Parts[0]
		generatedText := part.Text
		cleanedText := strings.TrimPrefix(generatedText, "```json")
		cleanedText = strings.TrimSuffix(cleanedText, "```")
		cleanedText = strings.TrimSpace(cleanedText)
		return cleanedText
	}
	return ""
}
