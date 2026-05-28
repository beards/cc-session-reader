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
			// "hello world" = 11 ASCII chars * 0.25 = 2.75 -> int(2.75) = 2
			name: "pure ASCII",
			text: "hello world",
			want: 2,
		},
		{
			// 4 CJK runes, len("你好世界") = 12 bytes.
			// cjkCount = 4, asciiCount = len(text) - cjkCount = 12 - 4 = 8.
			// int(4*1.5 + 8*0.25) = int(6 + 2) = 8
			name: "pure CJK",
			text: "你好世界",
			want: 8,
		},
		{
			// "hello 你好": len = 12 bytes (6 ASCII + 2 CJK * 3 bytes each).
			// cjkCount = 2, asciiCount = 12 - 2 = 10.
			// int(2*1.5 + 10*0.25) = int(3 + 2.5) = 5
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
			// 100 ASCII chars * 0.25 = 25
			name: "longer ASCII text",
			text: strings.Repeat("a", 100),
			want: 25,
		},
		{
			// 10 CJK runes, len = 30 bytes. asciiCount = 30 - 10 = 20.
			// int(10*1.5 + 20*0.25) = int(15 + 5) = 20
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

// Verifies the formula: asciiCount = len(text) - cjkCount (byte-based, not rune-based).
// A single CJK char is 3 bytes UTF-8: cjkCount=1, asciiCount=3-1=2.
// int(1*1.5 + 2*0.25) = int(2.0) = 2
func TestEstimateTokens_SingleCJKChar(t *testing.T) {
	got := EstimateTokens("世")
	want := 2
	if got != want {
		t.Errorf("single CJK char: got %d, want %d", got, want)
	}
}

// Verify that CJK Extension A range (U+3400-U+4DBF) is detected as CJK.
func TestEstimateTokens_CJKExtensionA(t *testing.T) {
	// U+3400 "㐀": 3 bytes, 1 CJK rune. Same formula as above -> 2.
	got := EstimateTokens("㐀")
	want := 2
	if got != want {
		t.Errorf("CJK Extension A char: got %d, want %d", got, want)
	}
}

// Verify non-CJK Unicode (accented Latin) is not counted as CJK.
// Use explicit byte construction to avoid Unicode normalization issues.
func TestEstimateTokens_NonCJKUnicode(t *testing.T) {
	// "caf" + e-acute (U+00E9, 2 bytes UTF-8): total 5 bytes, 0 CJK.
	// asciiCount = 5, int(5*0.25) = 1
	text := "café"
	got := EstimateTokens(text)
	want := 1
	if got != want {
		t.Errorf("accented text: got %d, want %d", got, want)
	}
}
