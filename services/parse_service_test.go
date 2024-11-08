package services

import (
    "testing"
)


// Test generated using Keploy
func TestNormalizeWhitespace(t *testing.T) {
    input := "This  is   a  test"
    expected := "This is a test"
    result := normalizeWhitespace(input)
    if result != expected {
        t.Errorf("Expected %s but got %s", expected, result)
    }
}

// Test generated using Keploy
func TestRemoveZeroWidthChars(t *testing.T) {
    input := "Hello\u200BWorld"
    expected := "HelloWorld"
    result := removeZeroWidthChars(input)
    if result != expected {
        t.Errorf("Expected %s but got %s", expected, result)
    }
}


// Test generated using Keploy
func TestExtractMonth(t *testing.T) {
    fileName := "report-03-2023.xlsx"
    expected := "2023-03-01"
    result := extractMonth(fileName)
    if result != expected {
        t.Errorf("Expected %s but got %s", expected, result)
    }
}


// Test generated using Keploy
func TestExtractFileName(t *testing.T) {
    url := "https://example.com/path/to/file.xlsx"
    expected := "file"
    result := extractFileName(url)
    if result != expected {
        t.Errorf("Expected %v but got %v", expected, result)
    }
}


// Test generated using Keploy
func TestCleanHTMLContent(t *testing.T) {
    input := "Hello\u200B World   !"
    expected := "Hello World !"
    result := cleanHTMLContent(input)
    if result != expected {
        t.Errorf("Expected %v but got %v", expected, result)
    }
}

