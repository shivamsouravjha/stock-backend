package helpers

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMatchHeader_NonMatchingPattern(t *testing.T) {
	cellValue := "Random Header"
	patterns := []string{"Name of (the )?Instrument"}
	result := MatchHeader(cellValue, patterns)
	if result {
		t.Errorf("Expected false, got %v", result)
	}
}

func TestCheckInstrumentName_Valid(t *testing.T) {
	input := "Name of the Instrument"
	result := CheckInstrumentName(input)
	if !result {
		t.Errorf("Expected true, got %v", result)
	}
}

func TestToFloat_StringWithCommas(t *testing.T) {
	input := "1,234.56"
	expected := 1234.56
	result := ToFloat(input)
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

func TestToStringArray(t *testing.T) {
	input := primitive.A{"one", "two", "three"}
	expected := []string{"one", "two", "three"}
	result := ToStringArray(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestNormalizeString(t *testing.T) {
	input := "  HeLLo WoRLd  "
	expected := "hello world"
	result := NormalizeString(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}
