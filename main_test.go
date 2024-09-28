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

func TestCommitAggregateActionStatus(t *testing.T) {
	tests := []struct {
		name     string
		actions  []Action
		expected ActionStatus
	}{
		{
			name:     "No actions",
			actions:  []Action{},
			expected: "",
		},
		{
			name: "All succeeded",
			actions: []Action{
				{Status: succeeded},
				{Status: succeeded},
			},
			expected: succeeded,
		},
		{
			name: "One running with another succeeded action",
			actions: []Action{
				{Status: succeeded},
				{Status: running},
			},
			expected: running,
		},
		{
			name: "One running with another failed action",
			actions: []Action{
				{Status: failed},
				{Status: running},
			},
			expected: running,
		},
		{
			name: "One failed",
			actions: []Action{
				{Status: succeeded},
				{Status: failed},
			},
			expected: failed,
		},
		{
			name: "One failed, different order",
			actions: []Action{
				{Status: failed},
				{Status: succeeded},
			},
			expected: failed,
		},
		{
			name: "Optional action failed",
			actions: []Action{
				{Status: failed, Optional: true},
				{Status: succeeded},
			},
			expected: succeeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commit := Commit{LatestActions: tt.actions}
			result := commit.AggregateActionStatus()
			if result != tt.expected {
				t.Errorf("Expected status %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestActionElapsedTime(t *testing.T) {
	action := Action{
		StartedAt:  time.Now().Add(-5 * time.Minute),
		FinishedAt: time.Now(),
	}

	elapsed := action.ElapsedTime()
	assert.InDelta(t, 5*time.Minute, elapsed, float64(time.Second), "ElapsedTime should be approximately 5 minutes")
}
