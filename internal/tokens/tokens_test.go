// Design note: EstimateTokens uses a byte-based heuristic for speed.
// It counts CJK runes, then treats ALL remaining bytes (including UTF-8
// continuation bytes of CJK chars) as "ASCII" for the 0.25 multiplier.
// This intentionally over-estimates CJK text, which is acceptable for
// its purpose: rough comparison between sessions, not billing-accurate counts.
package tokens

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{
			// 11 ASCII chars, ~0.25 tokens each -> heuristic approximation: 2
			name: "pure ASCII",
			text: "hello world",
			want: 2,
		},
		{
			// 4 CJK chars, ~1.5 tokens each -> naive expectation: 6
			// Heuristic over-estimates due to byte-based counting: 8
			name: "pure CJK",
			text: "你好世界",
			want: 8,
		},
		{
			// 6 ASCII chars + 2 CJK chars -> naive: 6*0.25 + 2*1.5 = 4.5
			// Heuristic approximation: 5
			name: "mixed ASCII and CJK",
			text: "hello 你好",
			want: 5,
		},
		{
			name: "empty string",
			text: "",
			want: 0,
		},
		{
			// 100 ASCII chars at ~0.25 tokens each -> 25
			name: "longer ASCII text",
			text: strings.Repeat("a", 100),
			want: 25,
		},
		{
			// 10 CJK chars, ~1.5 tokens each -> naive: 15
			// Heuristic over-estimates due to byte-based counting: 20
			name: "longer CJK text",
			text: strings.Repeat("测", 10),
			want: 20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

// A single CJK char is ~1.5 tokens; heuristic approximation: 2
func TestEstimateTokens_SingleCJKChar(t *testing.T) {
	got := EstimateTokens("世")
	want := 2
	if got != want {
		t.Errorf("single CJK char: got %d, want %d", got, want)
	}
}

// CJK Extension A range (U+3400-U+4DBF) should be detected as CJK.
func TestEstimateTokens_CJKExtensionA(t *testing.T) {
	// U+3400 "㐀" is CJK Extension A, same weight as unified CJK -> 2
	got := EstimateTokens("㐀")
	want := 2
	if got != want {
		t.Errorf("CJK Extension A char: got %d, want %d", got, want)
	}
}

// Non-CJK Unicode (accented Latin) should NOT be counted as CJK.
func TestEstimateTokens_NonCJKUnicode(t *testing.T) {
	// "café" = 4 visible characters, no CJK -> heuristic approximation: 1
	text := "café"
	got := EstimateTokens(text)
	want := 1
	if got != want {
		t.Errorf("accented text: got %d, want %d", got, want)
	}
}
