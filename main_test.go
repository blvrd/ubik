package main

import (
	"testing"
)

func TestStringToShortcode(t *testing.T) {
	input := "hello world"
	expected := "uU0nu1"
	shortcode := StringToShortcode(input)
	if shortcode != expected {
		t.Errorf("Expected %s, got %s", expected, shortcode)
	}
}

func TestTruncate(t *testing.T) {
	input := "hello world"
	expected := "hello w..."
	result := truncate(input, 10)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		min      int
		max      int
		expected int
	}{
		{"within range", 10, 0, 20, 10},
		{"at max", 20, 0, 20, 20},
		{"above max", 30, 0, 20, 20},
		{"below min", -10, 0, 20, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clamp(tt.value, tt.min, tt.max)
			if result != tt.expected {
				t.Errorf("clamp(%d, %d, %d) = %d; want %d",
					tt.value, tt.min, tt.max, result, tt.expected)
			}
		})
	}
}
