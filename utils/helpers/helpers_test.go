package helpers

import (
	"reflect"
	"stockbackend/types"
	"strings"
	"testing"

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

func TestCheckInstrumentName_Valid(t *testing.T) {
	input := "Name of the Instrument"
	result := CheckInstrumentName(input)
	if !result {
		t.Errorf("Expected true, got %v", result)
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

func TestGetMarketCapCategory_LargeCap(t *testing.T) {
	input := "20000"
	expected := "Large Cap"
	result := GetMarketCapCategory(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestGetMarketCapCategory_SmallCap(t *testing.T) {
	input := "4999"
	expected := "Small Cap"
	result := GetMarketCapCategory(input)
	if result != expected {
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

func TestGetMarketCapCategory_MidCap(t *testing.T) {
	input := "15000"
	expected := "Mid Cap"
	result := GetMarketCapCategory(input)
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

func TestParsePeersTable(t *testing.T) {
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
	stock := map[string]interface{}{
		"balanceSheet": map[string]interface{}{
			"Total Assets": primitive.A{"1000", "2000"},
		},
		"cashFlows": map[string]interface{}{
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
	stock := map[string]interface{}{
		"balanceSheet": map[string]interface{}{
			"Total Assets": primitive.A{"5000", "6000"},
		},
		"cashFlows": map[string]interface{}{
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
	stock := map[string]interface{}{
		"profitLoss": map[string]interface{}{
			"Net Profit +": primitive.A{"1000", "2000"},
		},
		"cashFlows": map[string]interface{}{
			"Cash from Operating Activity +": primitive.A{"500", "600"},
		},
	}
	result := calculateProfitabilityScore(stock)
	expected := -1
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestCalculateLeverageScore_MissingBorrowingsField(t *testing.T) {
	stock := map[string]interface{}{
		"balanceSheet": map[string]interface{}{
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
	stock := map[string]interface{}{
		"balanceSheet": map[string]interface{}{
			"Borrowings +":        primitive.A{"2000", "1500"},
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
