package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestStringToShortcode(t *testing.T) {
	input := "hello world"
	expected := "uU0nuZ"
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

func TestCommitAggregateCheckStatus(t *testing.T) {
	tests := []struct {
		name     string
		checks   []Check
		expected CheckStatus
	}{
		{
			name:     "No checks",
			checks:   []Check{},
			expected: "",
		},
		{
			name: "All succeeded",
			checks: []Check{
				{Status: succeeded},
				{Status: succeeded},
			},
			expected: succeeded,
		},
		{
			name: "One running",
			checks: []Check{
				{Status: succeeded},
				{Status: running},
			},
			expected: running,
		},
		{
			name: "One failed",
			checks: []Check{
				{Status: succeeded},
				{Status: failed},
			},
			expected: failed,
		},
		{
			name: "One failed, different order",
			checks: []Check{
				{Status: failed},
				{Status: succeeded},
			},
			expected: failed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commit := Commit{LatestChecks: tt.checks}
			result := commit.AggregateCheckStatus()
			if result != tt.expected {
				t.Errorf("Expected status %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCheckElapsedTime(t *testing.T) {
	check := Check{
		StartedAt:  time.Now().Add(-5 * time.Minute),
		FinishedAt: time.Now(),
	}

	elapsed := check.ElapsedTime()
	assert.InDelta(t, 5*time.Minute, elapsed, float64(time.Second), "ElapsedTime should be approximately 5 minutes")
}
