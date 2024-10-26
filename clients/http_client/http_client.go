package http_client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"stockbackend/types"
	"strings"

	"go.uber.org/zap"
)

func SearchCompany(queryString string) ([]types.Company, error) {
	// Replace "corporation" with "Corpn" and "limited" with "Ltd"
	queryString = strings.ReplaceAll(queryString, " Corporation ", " Corpn ")
	queryString = strings.ReplaceAll(queryString, " corporation ", " Corpn ")
	queryString = strings.ReplaceAll(queryString, " Limited", " Ltd ")
	queryString = strings.ReplaceAll(queryString, " limited", " Ltd ")
	queryString = strings.ReplaceAll(queryString, " and ", " & ")
	queryString = strings.ReplaceAll(queryString, " And ", " & ")
	// Base URL for the Screener API
	baseURL := os.Getenv("COMPANY_URL") + "/api/company/search/"

	// Create the URL with query parameters
	params := url.Values{}
	params.Add("q", queryString)
	params.Add("v", "3")
	params.Add("fts", "1")

	// Create the request
	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	// Create the client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var searchResponse []types.Company
	err = json.Unmarshal(body, &searchResponse)
	if err != nil {
		zap.L().Error("Failed to unmarshal search response", zap.Error(err))
		return nil, err
	}

	// Return the list of results
	return searchResponse, nil
}

func GetCompanyPage(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the URL: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to retrieve the content, status code: %d", resp.StatusCode)
	}

	respBody := resp.Body
	return respBody, nil
}
