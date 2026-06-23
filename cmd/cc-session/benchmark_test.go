package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintCompressionSection_UsesNewSessionTotalContext(t *testing.T) {
	results := []sessionBenchResult{
		{
			shortID:          "aaaaaaaa",
			contextTokens:    100_000,
			filteredTokens:   23_456,
			newContextTokens: 60_000,
			savedPct:         40.0,
		},
		{
			shortID:          "bbbbbbbb",
			contextTokens:    200_000,
			filteredTokens:   87_654,
			newContextTokens: 120_000,
			savedPct:         40.0,
		},
	}

	var out bytes.Buffer
	printCompressionSection(&out, results)
	got := out.String()

	if !strings.Contains(got, "Context      NewCtx") {
		t.Fatalf("compression header must compare total contexts, got:\n%s", got)
	}
	if strings.Contains(got, "Filtered") {
		t.Fatalf("compression table must not label history-only tokens as the comparable total context:\n%s", got)
	}
	if !strings.Contains(got, "aaaaaaaa") ||
		!strings.Contains(got, "100,000") ||
		!strings.Contains(got, "60,000") ||
		!strings.Contains(got, "40.0%") {
		t.Fatalf("compression row missing new session total context:\n%s", got)
	}
	if strings.Contains(got, "23,456") || strings.Contains(got, "87,654") {
		t.Fatalf("compression row leaked filtered-history-only token count:\n%s", got)
	}
}

func TestMedianFloat64_GivenEvenCount_ThenAveragesMiddleValues(t *testing.T) {
	got := medianFloat64([]float64{78.9, 90.3})
	want := 84.6
	if got != want {
		t.Fatalf("medianFloat64(even) = %.1f, want %.1f", got, want)
	}
}

func TestPrintCompressionSection_GivenEvenCount_ThenPrintsAveragedMedian(t *testing.T) {
	results := []sessionBenchResult{
		{shortID: "aaaaaaaa", contextTokens: 100, newContextTokens: 20, savedPct: 78.9},
		{shortID: "bbbbbbbb", contextTokens: 100, newContextTokens: 10, savedPct: 90.3},
	}

	var out bytes.Buffer
	printCompressionSection(&out, results)

	if got := out.String(); !strings.Contains(got, "Median: 84.6%") {
		t.Fatalf("compression summary must average the two middle values for even counts:\n%s", got)
	}
}
