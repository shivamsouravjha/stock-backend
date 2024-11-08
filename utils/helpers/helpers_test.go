package helpers

import (
	"reflect"
	"stockbackend/types"
	"strings"
	"testing"

	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/PuerkitoBio/goquery"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/mgo.v2/bson"
)

func TestMatchHeader_NonMatchingPattern(t *testing.T) {
	patterns := []string{"^Name of (the )?Instrument$"}
	result := MatchHeader("Instrument Name", patterns)
	if result {
		t.Errorf("Expected false, got %v", result)
	}
}

func TestToFloat_StringWithCommas(t *testing.T) {
	result := ToFloat("1,234.56")
	expected := 1234.56
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestToFloat_StringWithPercentage(t *testing.T) {
	result := ToFloat("12.34%")
	expected := 0.1234
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestToFloat_NonNumericString(t *testing.T) {
	input := "abc"
	expected := 0.0
	result := ToFloat(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestToFloat_StringWithOnlyCommas(t *testing.T) {
	input := ",,"
	expected := 0.0
	result := ToFloat(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestToFloat(t *testing.T) {
	input := "-a%"
	expected := 0.0
	result := ToFloat(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCheckInstrumentName_Valid(t *testing.T) {
	input := "Name of the Instrument"
	result := CheckInstrumentName(input)
	if !result {
		t.Errorf("Expected true, got %v", result)
	}
}

func TestGetMarketCapCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"20000", "Large Cap"},
		{"4999", "Small Cap"},
		{"15000", "Mid Cap"},
	}

	for _, test := range tests {
		result := GetMarketCapCategory(test.input)

		if result != test.expected {
			t.Errorf("Expected %v, got %v", test.expected, result)
		}
	}
}

func TestToStringArray_PrimitiveArray(t *testing.T) {
	input := primitive.A{"one", "two", "three"}
	result := ToStringArray(input)
	expected := []string{"one", "two", "three"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestToStringArray_InvalidInput(t *testing.T) {
	input := "invalid input"
	expected := []string{}
	result := ToStringArray(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestToFloat_NonStringInput(t *testing.T) {
	input := 1234.56
	expected := 0.0
	result := ToFloat(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestMatchHeader_NormalizedStringMatch(t *testing.T) {
	patterns := []string{"^name of (the )?instrument$"}
	result := MatchHeader(" Name of the Instrument ", patterns)
	if !result {
		t.Errorf("Expected true, got %v", result)
	}
}

func TestParseFloat_VariousInputs(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected float64
	}{
		{"123.45", 123.45},
		{123.45, 123.45},
		{123, 123.0},
		{"abc", 0.0},
	}

	for _, test := range tests {
		result := ParseFloat(test.input)
		if result != test.expected {
			t.Errorf("Expected %v, got %v", test.expected, result)
		}
	}
}

func TestFetchPeerData_HTTPError(t *testing.T) {
	_, err := FetchPeerData("invalidID")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestNormalizeString_NoSpaces(t *testing.T) {
	input := "TESTSTRING"
	expected := "teststring"
	result := NormalizeString(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestNormalizeString_WithSpaces(t *testing.T) {
	input := "   TEST STRING   "
	expected := "test string"
	result := NormalizeString(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestParseFloat_InvalidInput(t *testing.T) {
	input := []interface{}{true, []int{1, 2, 3}, map[string]string{"key": "value"}}
	for _, val := range input {
		result := ParseFloat(val)
		expected := 0.0
		if result != expected {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	}
}

func TestNormalizeString_LeadingTrailingSpaces(t *testing.T) {
	input := "   TEST STRING   "
	expected := "test string"
	result := NormalizeString(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestParsePeersTable_CorrectParsing(t *testing.T) {
	html := `
    <html>
    <body>
        <div id="peers">
            <table>
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>PE</th>
                        <th>Market Cap</th>
                        <th>Dividend Yield</th>
                        <th>ROCE</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td>Peer 1</td>
                        <td>10.0</td>
                        <td>8000</td>
                        <td>2.0%</td>
                        <td>18.0</td>
                    </tr>
                    <tr>
                        <td>Peer 2</td>
                        <td>12.0</td>
                        <td>9000</td>
                        <td>2.2%</td>
                        <td>19.0</td>
                    </tr>
                </tbody>
            </table>
        </div>
    </body>
    </html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	result := ParsePeersTable(doc, "#peers")
	expected := []map[string]string{
		{"Name": "Peer 1", "PE": "10.0", "Market Cap": "8000", "Dividend Yield": "2.0%", "ROCE": "18.0"},
		{"Name": "Peer 2", "PE": "12.0", "Market Cap": "9000", "Dividend Yield": "2.2%", "ROCE": "19.0"},
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateRoa_CorrectCalculation(t *testing.T) {
	netProfit := "1000"
	totalAssets := "5000"
	expected := 0.2
	result := calculateRoa(netProfit, totalAssets)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateRoa_ValidInputs(t *testing.T) {
	netProfit := "1000"
	totalAssets := "5000"
	expected := 0.2
	result := calculateRoa(netProfit, totalAssets)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCheckInstrumentName_ValidWithoutThe(t *testing.T) {
	input := "Name of Instrument"
	result := CheckInstrumentName(input)
	if !result {
		t.Errorf("Expected true, got %v", result)
	}
}

func TestCheckInstrumentName_Invalid(t *testing.T) {
	input := "Instrument Name"
	result := CheckInstrumentName(input)
	if result {
		t.Errorf("Expected false, got %v", result)
	}
}

func TestParseFloat_IntegerInput(t *testing.T) {
	input := 123
	expected := 123.0
	result := ParseFloat(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestNormalizeString_EmptyString(t *testing.T) {
	input := ""
	expected := ""
	result := NormalizeString(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestNormalizeString_MixedCaseAndSpaces(t *testing.T) {
	input := "  TeSt StRiNg  "
	expected := "test string"
	result := NormalizeString(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestNormalizeString_MixedCase(t *testing.T) {
	input := "TeSt StRiNg"
	expected := "test string"
	result := NormalizeString(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestFetchPeerData_InvalidID(t *testing.T) {
	_, err := FetchPeerData("invalidID")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestIncreaseInRoa(t *testing.T) {
	netProfit := primitive.A{"1000", "1500", "2000"}
	totalAssets := primitive.A{"5000", "5500", "6000"}
	result := increaseInRoa(netProfit, totalAssets)
	if !result {
		t.Errorf("Expected true, got %v", result)
	}
}

func TestIncreaseInRoa_True(t *testing.T) {
	netProfit := primitive.A{"1000", "1500", "2000"}
	totalAssets := primitive.A{"5000", "5500", "6000"}
	result := increaseInRoa(netProfit, totalAssets)
	if !result {
		t.Errorf("Expected true, got %v", result)
	}
}

func TestIncreaseInRoa_False(t *testing.T) {
	netProfit := primitive.A{"2000", "1500", "1000"}
	totalAssets := primitive.A{"6000", "5500", "5000"}
	result := increaseInRoa(netProfit, totalAssets)
	if result {
		t.Errorf("Expected false, got %v", result)
	}
}

func TestParseTableData_MultipleRowsAndColumns(t *testing.T) {
	html := `
    <html>
    <body>
        <section id="data-section">
            <table>
                <thead>
                    <tr>
                        <th>Year</th>
                        <th>2019</th>
                        <th>2020</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td class="text">Revenue</td>
                        <td>1000</td>
                        <td>1500</td>
                    </tr>
                    <tr>
                        <td class="text">Profit</td>
                        <td>200</td>
                        <td>300</td>
                    </tr>
                </tbody>
            </table>
        </section>
    </body>
    </html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	section := doc.Find("#data-section")
	result := ParseTableData(section, "table")
	expected := map[string]interface{}{
		"Revenue": []string{"1000", "1500"},
		"Profit":  []string{"200", "300"},
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateProfitabilityScore_MissingProfitLossField(t *testing.T) {
	stock := bson.M{
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{"1000", "2000"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity +": primitive.A{"500", "600"},
		},
	}
	result := calculateProfitabilityScore(stock)
	expected := -1
	if result != expected {

		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateProfitabilityScore_MissingNetProfitField(t *testing.T) {
	stock := bson.M{
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{"5000", "6000"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity +": primitive.A{"500", "600"},
		},
	}
	result := calculateProfitabilityScore(stock)
	expected := -1
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateProfitabilityScore_MissingTotalAssetsField(t *testing.T) {
	stock := bson.M{
		"profitLoss": bson.M{
			"Net Profit\u00A0+": primitive.A{"1000", "2000"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity +": primitive.A{"500", "600"},
		},
	}
	result := calculateProfitabilityScore(stock)
	expected := -1
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateProfitabilityScore_MissingCashFromOperatingActivity(t *testing.T) {
	stock := bson.M{
		"profitLoss": bson.M{
			"Net Profit\u00A0+": primitive.A{
				"5", "-2", "-2",
				"11", "16", "19",
				"25", "24", "23",
				"35", "42", "56",
				"59",
			},
		},
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{
				"294", "328", "333",
				"334", "363", "376",
				"404", "444", "452",
				"514", "523", "588",
			},
		},
	}
	result := calculateProfitabilityScore(stock)
	expected := -1
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateProfitabilityScore_CurrentCashOpsGreaterThanPrevious(t *testing.T) {
	stock := bson.M{
		"profitLoss": bson.M{
			"Net Profit\u00A0+": primitive.A{
				"5", "-2", "-2",
				"11", "16", "19",
				"25", "24", "23",
				"35", "42", "56",
				"59",
			},
		},
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{
				"294", "328", "333",
				"334", "363", "376",
				"404", "444", "452",
				"514", "523", "588",
			},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity\u00A0+": primitive.A{
				"45", "37", "30",
				"25", "44", "61",
				"53", "45", "52",
				"35", "63", "64",
			},
		},
	}
	result := calculateProfitabilityScore(stock)
	expected := 4
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateLeverageScore_MissingBorrowingsField(t *testing.T) {
	stock := bson.M{
		"balanceSheet": bson.M{
			"Total Assets":        primitive.A{"5000", "4000"},
			"Other Assets +":      primitive.A{"3000", "2500"},
			"Other Liabilities +": primitive.A{"1000", "800"},
			"Equity Capital":      primitive.A{"1000", "1000"},
		},
	}
	result := calculateLeverageScore(stock)
	expected := -1
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateLeverageScore_MissingTotalAssetsField(t *testing.T) {
	stock := bson.M{
		"balanceSheet": bson.M{
			"Borrowings\u00A0+":        primitive.A{"5000", "4000"},
			"Other Assets\u00A0+":      primitive.A{"3000", "2500"},
			"Other Liabilities\u00A0+": primitive.A{"1000", "800"},
			"Equity Capital":           primitive.A{"1000", "1000"},
		},
	}
	result := calculateLeverageScore(stock)
	expected := -1
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateLeverageScore_MissingOtherAssets(t *testing.T) {
	stock := bson.M{
		"balanceSheet": bson.M{
			"Borrowings\u00A0+":        primitive.A{"5000", "4000"},
			"Total Assets":             primitive.A{"3000", "2500"},
			"Other Liabilities\u00A0+": primitive.A{"1000", "800"},
			"Equity Capital":           primitive.A{"1000", "1000"},
		},
	}
	result := calculateLeverageScore(stock)
	expected := -1
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateLeverageScore_MissingOtherLiabilities(t *testing.T) {
	stock := bson.M{
		"balanceSheet": bson.M{
			"Borrowings\u00A0+":   primitive.A{"5000", "4000"},
			"Total Assets":        primitive.A{"3000", "2500"},
			"Other Assets\u00A0+": primitive.A{"1000", "800"},
			"Equity Capital":      primitive.A{"1000", "1000"},
		},
	}
	result := calculateLeverageScore(stock)
	expected := -1
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateLeverageScore_MissingEquityCapital(t *testing.T) {
	stock := bson.M{
		"balanceSheet": bson.M{
			"Borrowings\u00A0+":        primitive.A{"5000", "4000"},
			"Total Assets":             primitive.A{"3000", "2500"},
			"Other Assets\u00A0+":      primitive.A{"1000", "800"},
			"Other Liabilities\u00A0+": primitive.A{"1000", "800"},
		},
	}
	result := calculateLeverageScore(stock)
	expected := -1
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestGetNestedArrayField_ValidField(t *testing.T) {
	stock := map[string]interface{}{
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{"1000", "2000"},
		},
	}
	result, err := getNestedArrayField(stock, "balanceSheet", "Total Assets")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := primitive.A{"1000", "2000"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestGetNestedArrayField_FieldNotFound(t *testing.T) {
	stock := map[string]interface{}{
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{"1000", "2000"},
		},
	}
	_, err := getNestedArrayField(stock, "balanceSheet", "NonExistentField")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestGetNestedArrayField_EmptyFields(t *testing.T) {
	stock := map[string]interface{}{
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{"1000", "2000"},
		},
	}
	_, err := getNestedArrayField(stock)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestGetNestedArrayField_NonStringElements(t *testing.T) {
	stock := map[string]interface{}{
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{"1000", 2000},
		},
	}
	_, err := getNestedArrayField(stock, "balanceSheet", "Total Assets")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestGetNestedArrayField_IncorrectPath(t *testing.T) {
	stock := map[string]interface{}{
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{"1000", "2000"},
		},
	}
	_, err := getNestedArrayField(stock, "balanceSheet", "TotalAssets")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestGetNestedArrayField_KeysWithPlus(t *testing.T) {
	stock := map[string]interface{}{
		"balanceSheet": bson.M{
			"Other Liabilities\u00A0+": primitive.A{
				"22", "8", "20", "20",
				"20", "18", "4", "2",
				"2", "2", "7", "2",
			},
		},
	}

	_, err := getNestedArrayField(stock, "balanceSheet", "Other Liabilities +")
	if err != nil {
		t.Error("Returned error ", err.Error())
	}
}

func TestFetchCompanyData_InvalidURL(t *testing.T) {
	_, err := FetchCompanyData("invalid-url")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestParseShareholdingPattern_CorrectParsing(t *testing.T) {
	html := `
    <html>
    <body>
        <div id="shareholding">
            <div id="quarterly-shp">
                <table>
                    <thead>
                        <tr><th>Category</th><th>Q1</th><th>Q2</th></tr>
                    </thead>
                    <tbody>
                        <tr><td class="text">Promoters</td><td>50%</td><td>51%</td></tr>
                    </tbody>
                </table>
            </div>
            <div id="yearly-shp">
                <table>
                    <thead>
                        <tr><th>Category</th><th>2020</th><th>2021</th></tr>
                    </thead>
                    <tbody>
                        <tr><td class="text">Promoters</td><td>50%</td><td>51%</td></tr>
                    </tbody>
                </table>
            </div>
        </div>
    </body>
    </html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	section := doc.Find("#shareholding")
	result := ParseShareholdingPattern(section)
	expected := map[string]interface{}{
		"quarterly": []map[string]interface{}{
			{"category": "Promoters", "values": map[string]string{"Q1": "50%", "Q2": "51%"}},
		},
		"yearly": []map[string]interface{}{
			{"category": "Promoters", "values": map[string]string{"2020": "50%", "2021": "51%"}},
		},
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestParseTableData_CorrectParsing(t *testing.T) {
	html := `
    <html>
    <body>
        <section id="data-section">
            <table>
                <thead>
                    <tr>
                        <th>Year</th>
                        <th>2019</th>
                        <th>2020</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td class="text">Revenue</td>
                        <td>1000</td>
                        <td>1500</td>
                    </tr>
                    <tr>
                        <td class="text">Profit</td>
                        <td>200</td>
                        <td>300</td>
                    </tr>
                </tbody>
            </table>
        </section>
    </body>
    </html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	section := doc.Find("#data-section")
	result := ParseTableData(section, "table")
	expected := map[string]interface{}{
		"Revenue": []string{"1000", "1500"},
		"Profit":  []string{"200", "300"},
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestParseTableData_EmptyTable(t *testing.T) {
	html := `
    <html>
    <body>
        <section id="data-section">
            <table>
                <thead>
                    <tr>
                        <th>Year</th>
                        <th>2019</th>
                        <th>2020</th>
                    </tr>
                </thead>
                <tbody>
                </tbody>
            </table>
        </section>
    </body>
    </html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	section := doc.Find("#data-section")
	result := ParseTableData(section, "table")
	expected := map[string]interface{}{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestParseTableData_NoTable(t *testing.T) {
	html := `
    <html>
    <body>
        <section id="data-section">
        </section>
    </body>
    </html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	section := doc.Find("#data-section")
	result := ParseTableData(section, "table")

	if !reflect.ValueOf(result).IsNil() {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestRateStock_MissingFields(t *testing.T) {
	stock := map[string]interface{}{
		"name": "Incomplete Stock",
		// Missing other fields
	}
	result := RateStock(stock)
	expected := 0.0
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestRateStock_ValidFields(t *testing.T) {
	stock := map[string]interface{}{
		"name":             "Valid Stock",
		"stockPE":          "15.5",
		"marketCap":        "10000",
		"dividendYield":    "2.5%",
		"roce":             "20.0",
		"cons":             primitive.A{"High debt", "Low liquidity"},
		"pros":             primitive.A{"Strong brand", "High growth potential"},
		"peers":            primitive.A{bson.M{"pe": "10.0", "market_cap": "8000", "div_yield": "2.0%", "roce": "18.0", "sales_qtr": "500", "np_qtr": "50"}, bson.M{"pe": "12.0", "market_cap": "9000", "div_yield": "2.2%", "roce": "19.0", "sales_qtr": "600", "np_qtr": "60"}, bson.M{"pe": "11.0", "market_cap": "8500", "div_yield": "2.1%", "roce": "18.5", "sales_qtr": "550", "np_qtr": "55"}},
		"quarterlyResults": bson.M{"Q1": primitive.A{bson.M{"sales": "1000", "profit": "100"}, bson.M{"sales": "1100", "profit": "110"}}, "Q2": primitive.A{bson.M{"sales": "1200", "profit": "120"}, bson.M{"sales": "1300", "profit": "130"}}},
	}
	result := RateStock(stock)
	if result == 0.0 {
		t.Errorf("Expected non-zero rating, got %v", result)
	}
}

func TestAnalyzeTrend_ValidData(t *testing.T) {
	stock := types.Stock{
		Name:          "Test Stock",
		PE:            15.5,
		MarketCap:     10000,
		DividendYield: 2.5,
		ROCE:          20.0,
	}
	pastData := bson.M{
		"Q1": primitive.A{bson.M{"sales": "1000", "profit": "100"}, bson.M{"sales": "1100", "profit": "110"}},
		"Q2": primitive.A{bson.M{"sales": "1200", "profit": "120"}, bson.M{"sales": "1300", "profit": "130"}},
	}
	result := AnalyzeTrend(stock, pastData)
	if result == 0.0 {
		t.Errorf("Expected non-zero trend score, got %v", result)
	}
}

func TestAnalyzeTrend_DecreasingTrend(t *testing.T) {
	stock := types.Stock{
		Name:          "Test Stock",
		PE:            15.5,
		MarketCap:     10000,
		DividendYield: 2.5,
		ROCE:          20.0,
	}
	pastData := bson.M{
		"Q1": primitive.A{bson.M{"sales": "1000", "profit": "100"}, bson.M{"sales": "1100", "profit": "110"}},
		"Q2": primitive.A{bson.M{"sales": "1200", "profit": "120"}, bson.M{"sales": "1300", "profit": "80"}},
	}
	result := AnalyzeTrend(stock, pastData)

	if result == 0.0 {
		t.Errorf("Expected non-zero trend score, got %v", result)
	}
}

func TestCompareWithPeers_InsufficientPeers(t *testing.T) {
	stock := types.Stock{
		Name:          "Test Stock",
		PE:            15.5,
		MarketCap:     10000,
		DividendYield: 2.5,
		ROCE:          20.0,
	}
	peers := primitive.A{bson.M{"pe": "10.0", "market_cap": "8000", "div_yield": "2.0%", "roce": "18.0"}}
	result := compareWithPeers(stock, peers)
	expected := 0.0
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCompareWithPeers(t *testing.T) {
	stock := types.Stock{
		Name:            "Test Stock",
		PE:              15.5,
		MarketCap:       10000,
		DividendYield:   2.5,
		ROCE:            20.0,
		QuarterlySales:  7000.0,
		QuarterlyProfit: 2000.0,
	}
	peers := primitive.A{
		bson.M{
			"market_cap":     "449311.45",
			"div_yield":      "0.74",
			"qtr_sales_var":  "5.96",
			"roce":           "17.32",
			"name":           "Sun Pharma.Inds.",
			"current_price":  "1872.65",
			"pe":             "42.57",
			"np_qtr":         "2860.51",
			"qtr_profit_var": "25.05",
			"sales_qtr":      "12652.75",
		},
		bson.M{
			"market_cap":     "132360.30",
			"div_yield":      "0.79",
			"qtr_profit_var": "18.05",
			"sales_qtr":      "6693.94",
			"name":           "Cipla",
			"pe":             "29.85",
			"np_qtr":         "1175.46",
			"qtr_sales_var":  "5.77",
			"roce":           "22.80",
			"current_price":  "1639.00",
		},
		bson.M{
			"qtr_sales_var":  "13.88",
			"roce":           "26.53",
			"name":           "Dr Reddy's Labs",
			"pe":             "19.93",
			"div_yield":      "0.60",
			"np_qtr":         "1392.40",
			"qtr_profit_var": "-0.90",
			"sales_qtr":      "7696.10",
			"current_price":  "6638.40",
			"market_cap":     "110774.91",
		},
	}

	result := compareWithPeers(stock, peers)
	expected := 34.0
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCheckArrayElementsAreString_AllElementsAreString(t *testing.T) {
	input := primitive.A{"all", "elements", "are", "string"}
	expected := primitive.A{"all", "elements", "are", "string"}

	result, err := checkArrayElementsAreString(input)
	if err != nil {
		t.Error("Received error: ", err.Error())
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

func TestCheckArrayElementsAreString_AllElementsAreNotString(t *testing.T) {
	input := primitive.A{"all", "elements", "are", 1, "string"}
	expected := primitive.A{}

	result, err := checkArrayElementsAreString(input)
	if err == nil {
		t.Error("Expected error received nil")
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

func TestGenerateFScore_OnProfitabilityError(t *testing.T) {
	stock := bson.M{
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{"1000", "2000"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity +": primitive.A{"500", "600"},
		},
	}

	result := GenerateFScore(stock)
	expected := -1

	if result != expected {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

func TestGenerateFScore_OnLeverageScoreError(t *testing.T) {
	stock := bson.M{
		"profitLoss": bson.M{
			"Net Profit\u00A0+": primitive.A{"5", "-2", "-2"},
		},
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{"294", "328", "333"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity\u00A0+": primitive.A{"45", "37", "30"},
		},
	}

	result := GenerateFScore(stock)
	expected := -1

	if result != expected {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

func TestGenerateFScore_OnOperatingEfficiencyError(t *testing.T) {
	stock := bson.M{
		"profitLoss": bson.M{
			"Net Profit\u00A0+": primitive.A{"5", "-2", "-2"},
		},
		"balanceSheet": bson.M{
			"Total Assets":             primitive.A{"294", "328", "333"},
			"Borrowings\u00A0+":        primitive.A{"1,464", "495", "486"},
			"Other Assets\u00A0+":      primitive.A{"2,193", "1,882", "1,497"},
			"Other Liabilities\u00A0+": primitive.A{"1,408", "529", "548"},
			"Equity Capital":           primitive.A{"9", "9", "9", "9"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity\u00A0+": primitive.A{"45", "37", "30"},
		},
	}

	result := GenerateFScore(stock)
	expected := -1

	if result != expected {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

func TestGenerateTestFScore_ValidInput(t *testing.T) {
	stock := bson.M{
		"profitLoss": bson.M{
			"Net Profit\u00A0+": primitive.A{
				"5", "-2", "-2",
				"11", "16", "19",
				"25", "24", "23",
				"35", "42", "56",
				"59",
			},
			"OPM %": primitive.A{
				"26%", "-19%", "-126%",
				"-122%", "-73%", "-23%",
				"-74%", "-71%", "-52%",
				"-73%", "-9%", "21%",
				"5%",
			},
			"Sales +": primitive.A{
				"752", "568", "210",
				"205", "218", "322",
				"261", "212", "160",
				"160", "290", "472",
				"396",
			},
			"Revenue": primitive.A{
				"1,266", "1,388", "1,575", "1,728", "2,043", "2,587",
			},
		},
		"balanceSheet": bson.M{
			"Total Assets": primitive.A{
				"294", "328", "333",
				"334", "363", "376",
				"404", "444", "452",
				"514", "523", "588",
			},
			"Borrowings\u00A0+": primitive.A{
				"1,464", "495", "486",
				"509", "498", "101",
				"4", "0", "0",
				"0", "4", "5",
			},
			"Other Assets\u00A0+": primitive.A{
				"2,193", "1,882",
				"1,497", "1,263",
				"1,386", "1,683",
				"2,077", "2,121",
				"2,120", "2,190",
				"2,421", "2,633",
			},
			"Other Liabilities\u00A0+": primitive.A{
				"1,408", "529", "548",
				"327", "311", "287",
				"275", "268", "251",
				"272", "347", "293",
			},
			"Equity Capital": primitive.A{
				"9", "9", "9", "9",
				"9", "9", "9", "9",
				"9", "9", "9", "9",
			},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity\u00A0+": primitive.A{
				"45", "37", "30",
				"25", "44", "61",
				"53", "45", "52",
				"35", "63", "54",
			},
		},
	}

	result := GenerateFScore(stock)
	expected := 6
	if result != expected {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

func TestCalculateOperatingEfficiencyScore_MissingNetProfit(t *testing.T) {
	stock := bson.M{
		"balanceSheet": bson.M{
			"Total Assets":             primitive.A{"294", "328", "333"},
			"Borrowings\u00A0+":        primitive.A{"1,464", "495", "486"},
			"Other Assets\u00A0+":      primitive.A{"2,193", "1,882", "1,497"},
			"Other Liabilities\u00A0+": primitive.A{"1,408", "529", "548"},
			"Equity Capital":           primitive.A{"9", "9", "9", "9"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity\u00A0+": primitive.A{"45", "37", "30"},
		},
	}

	result := calculateOperatingEfficiencyScore(stock)
	expected := -1

	if result != expected {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

func TestCalculateOperatingEfficiencyScore_MissingRevenueWithSalesMissing(t *testing.T) {
	stock := bson.M{
		"profitLoss": bson.M{
			"Net Profit\u00A0+": primitive.A{"5", "-2", "-2"},
			"OPM %":             primitive.A{"26%", "-19%", "-126%"},
		},
		"balanceSheet": bson.M{
			"Total Assets":             primitive.A{"294", "328", "333"},
			"Borrowings\u00A0+":        primitive.A{"1,464", "495", "486"},
			"Other Assets\u00A0+":      primitive.A{"2,193", "1,882", "1,497"},
			"Other Liabilities\u00A0+": primitive.A{"1,408", "529", "548"},
			"Equity Capital":           primitive.A{"9", "9", "9", "9"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity\u00A0+": primitive.A{"45", "37", "30"},
		},
	}

	result := calculateOperatingEfficiencyScore(stock)
	expected := -1

	if result != expected {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

func TestCalculateOperatingEfficiencyScore_MissingSales(t *testing.T) {
	stock := bson.M{
		"profitLoss": bson.M{
			"Net Profit\u00A0+": primitive.A{"5", "-2", "-2"},
			"OPM %":             primitive.A{"26%", "-19%", "-126%"},
			"Revenue":           primitive.A{"1,266", "1,388", "1,575"},
		},
		"balanceSheet": bson.M{
			"Total Assets":             primitive.A{"294", "328", "333"},
			"Borrowings\u00A0+":        primitive.A{"1,464", "495", "486"},
			"Other Assets\u00A0+":      primitive.A{"2,193", "1,882", "1,497"},
			"Other Liabilities\u00A0+": primitive.A{"1,408", "529", "548"},
			"Equity Capital":           primitive.A{"9", "9", "9", "9"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity\u00A0+": primitive.A{"45", "37", "30"},
		},
	}

	result := calculateOperatingEfficiencyScore(stock)
	expected := 1

	if result != expected {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

func TestCalculateOperatingEfficiencyScore_CurrentMarginGreaterThanPrevious(t *testing.T) {
	stock := bson.M{
		"profitLoss": bson.M{
			"Net Profit\u00A0+": primitive.A{"5", "10", "-2"},
			"Revenue":           primitive.A{"1,266", "1,388", "1,575"},
			"Sales\u00A0+":      primitive.A{"752", "568", "210"},
		},
		"balanceSheet": bson.M{
			"Total Assets":             primitive.A{"294", "328", "333"},
			"Borrowings\u00A0+":        primitive.A{"1,464", "495", "486"},
			"Other Assets\u00A0+":      primitive.A{"2,193", "1,882", "1,497"},
			"Other Liabilities\u00A0+": primitive.A{"1,408", "529", "548"},
			"Equity Capital":           primitive.A{"9", "9", "9", "9"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity\u00A0+": primitive.A{"45", "37", "30"},
		},
	}

	result := calculateOperatingEfficiencyScore(stock)
	expected := 1

	if result != expected {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

func TestCalculateOperatingEfficiencyScore_ProfitAndRevenueLengthLessThan2(t *testing.T) {
	stock := bson.M{
		"profitLoss": bson.M{
			"Net Profit\u00A0+": primitive.A{"5"},
			"Revenue":           primitive.A{"1,266"},
			"Sales\u00A0+":      primitive.A{"752", "568", "210"},
		},
		"balanceSheet": bson.M{
			"Total Assets":             primitive.A{"294", "328", "333"},
			"Borrowings\u00A0+":        primitive.A{"1,464", "495", "486"},
			"Other Assets\u00A0+":      primitive.A{"2,193", "1,882", "1,497"},
			"Other Liabilities\u00A0+": primitive.A{"1,408", "529", "548"},
			"Equity Capital":           primitive.A{"9", "9", "9", "9"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity\u00A0+": primitive.A{"45", "37", "30"},
		},
	}

	result := calculateOperatingEfficiencyScore(stock)
	expected := -1

	if result != expected {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

func TestCalculateOperatingEfficiencyScore_TotalAssetsMissing(t *testing.T) {
	stock := bson.M{
		"profitLoss": bson.M{
			"Net Profit\u00A0+": primitive.A{"5"},
			"Revenue":           primitive.A{"1,266"},
			"OPM %":             primitive.A{"26%", "-19%", "-126%"},
			"Sales\u00A0+":      primitive.A{"752", "568", "210"},
		},
		"balanceSheet": bson.M{
			"Borrowings\u00A0+":        primitive.A{"1,464", "495", "486"},
			"Other Assets\u00A0+":      primitive.A{"2,193", "1,882", "1,497"},
			"Other Liabilities\u00A0+": primitive.A{"1,408", "529", "548"},
			"Equity Capital":           primitive.A{"9", "9", "9", "9"},
		},
		"cashFlows": bson.M{
			"Cash from Operating Activity\u00A0+": primitive.A{"45", "37", "30"},
		},
	}

	result := calculateOperatingEfficiencyScore(stock)
	expected := -1

	if result != expected {
		t.Errorf("Expected %v got %v", expected, result)
	}
}

// Test generated using Keploy
func TestIncreaseInRoa_NotEnoughData(t *testing.T) {
	netProfit := primitive.A{"1000"}
	totalAssets := primitive.A{"5000"}
	result := increaseInRoa(netProfit, totalAssets)
	if result {
		t.Errorf("Expected false, got %v", result)
	}
}

// Test generated using Keploy
func TestSafeToFloat_EmptyString(t *testing.T) {
	input := ""
	_, err := safeToFloat(input)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

// Test generated using Keploy
func TestFetchPeerData_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a valid HTML response
		fmt.Fprintln(w, `<html><body>
			<table>
				<tbody>
					<tr data-row-company-id="1">
						<td class="text"><a>Peer Company</a></td>
						<td>1000</td>
						<td>15.0</td>
						<td>5000</td>
						<td>2.0%</td>
						<td>100</td>
						<td>5%</td>
						<td>200</td>
						<td>10%</td>
						<td>15%</td>
					</tr>
				</tbody>
				<tfoot>
					<tr>
						<td>Total</td>
						<td>1</td>
						<td>1000</td>
						<td>15.0</td>
						<td>5000</td>
						<td>2.0%</td>
						<td>100</td>
						<td>5%</td>
						<td>200</td>
						<td>10%</td>
						<td>15%</td>
					</tr>
				</tfoot>
			</table>
		</body></html>`)
	}))
	defer server.Close()

	// Override the COMPANY_URL environment variable
	os.Setenv("COMPANY_URL", server.URL)

	dataWarehouseID := "validID"

	peersData, err := FetchPeerData(dataWarehouseID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(peersData) != 2 {
		t.Errorf("Expected 2 peers data, got %d", len(peersData))
	}
}

// Test generated using Keploy
func TestGetMarketCapCategory_NaNInput(t *testing.T) {
	input := "NaN"
	expected := "Unknown Category"
	result := GetMarketCapCategory(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// Test generated using Keploy
func TestGetMarketCapCategory_NonNumericInput(t *testing.T) {
	input := "invalid"
	expected := "Small Cap"
	result := GetMarketCapCategory(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// Test generated using Keploy
func TestCompareWithPeers_MedianParseError(t *testing.T) {
	stock := types.Stock{
		Name: "Test Stock",
		PE:   15.5,
	}
	peers := primitive.A{
		bson.M{
			"pe": "10.0",
		},
		"invalid median data",
	}
	result := compareWithPeers(stock, peers)
	if result == 0.0 {
		t.Errorf("Expected a non-zero score, got %v", result)
	}
}

// Test generated using Keploy
func TestFetchPeerData_Non200Response(t *testing.T) {
	// Create a mock server that returns a 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Override the COMPANY_URL environment variable
	os.Setenv("COMPANY_URL", server.URL)

	dataWarehouseID := "validID"

	_, err := FetchPeerData(dataWarehouseID)
	if err == nil {
		t.Errorf("Expected error due to non-200 response, got nil")
	}
}

// Test generated using Keploy
func TestFetchCompanyData_ValidURL(t *testing.T) {
	// Create a mock server that returns valid HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `<html><body><div data-warehouse-id="123"></div></body></html>`)
	}))
	defer server.Close()

	// Fetch company data using the mock server URL
	data, err := FetchCompanyData(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if data == nil {
		t.Errorf("Expected non-nil data, got nil")
	}
}

// Test generated using Keploy
func TestFetchCompanyData_MissingSections(t *testing.T) {
	html := `
    <html>
    <body>
        <div data-warehouse-id="123"></div>
        <li class="flex flex-space-between" data-source="default">
            <span class="name">Market Cap</span>
            <span class="nowrap value">100000</span>
        </li>
    </body>
    </html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, html)
	}))
	defer server.Close()
	data, err := FetchCompanyData(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if _, ok := data["profitLoss"]; ok {
		t.Errorf("Did not expect profitLoss data")
	}
	if _, ok := data["balanceSheet"]; ok {
		t.Errorf("Did not expect balanceSheet data")
	}
}

// Test generated using Keploy
func TestFetchCompanyData_ProsAndCons(t *testing.T) {
	html := `
    <html>
    <body>
        <div class="pros">
            <ul>
                <li>Strong financials</li>
                <li>Growing revenue</li>
            </ul>
        </div>
        <div class="cons">
            <ul>
                <li>High debt</li>
            </ul>
        </div>
    </body>
    </html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, html)
	}))
	defer server.Close()

	data, err := FetchCompanyData(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedPros := []string{"Strong financials", "Growing revenue"}
	expectedCons := []string{"High debt"}

	if !reflect.DeepEqual(data["pros"], expectedPros) {
		t.Errorf("Expected pros %v, got %v", expectedPros, data["pros"])
	}

	if !reflect.DeepEqual(data["cons"], expectedCons) {
		t.Errorf("Expected cons %v, got %v", expectedCons, data["cons"])
	}
}

// Test generated using Keploy
func TestCompareWithPeers_ValidData(t *testing.T) {
	stock := types.Stock{
		Name:            "Test Stock",
		PE:              15.5,
		MarketCap:       10000,
		DividendYield:   2.5,
		ROCE:            20.0,
		QuarterlySales:  7000.0,
		QuarterlyProfit: 2000.0,
	}
	peers := primitive.A{
		bson.M{"pe": "10.0", "market_cap": "8000", "div_yield": "2.0%", "roce": "18.0", "sales_qtr": "500", "np_qtr": "50"},
		bson.M{"pe": "12.0", "market_cap": "9000", "div_yield": "2.2%", "roce": "19.0", "sales_qtr": "600", "np_qtr": "60"},
		bson.M{"pe": "11.0", "market_cap": "8500", "div_yield": "2.1%", "roce": "18.5", "sales_qtr": "550", "np_qtr": "55"},
	}
	result := compareWithPeers(stock, peers)
	if result == 0.0 {
		t.Errorf("Expected non-zero peer comparison score, got %v", result)
	}
}

// Test generated using Keploy
func TestFetchPeerData_MalformedURL(t *testing.T) {
	// Override the COMPANY_URL environment variable with a malformed URL
	os.Setenv("COMPANY_URL", "http://%41:8080/") // %41 is 'A', but this is a malformed URL

	dataWarehouseID := "validID"
	_, err := FetchPeerData(dataWarehouseID)
	if err == nil {
		t.Errorf("Expected error due to malformed URL, got nil")
	}
}

// Test generated using Keploy
func TestIncreaseInRoa_InvalidElements(t *testing.T) {
	netProfit := primitive.A{1000, 1500, 2000}        // Integers instead of strings
	totalAssets := primitive.A{5000, 5500, "invalid"} // Last element is not a valid string
	result := increaseInRoa(netProfit, totalAssets)
	if result != false {
		t.Errorf("Expected false, got %v", result)
	}
}
