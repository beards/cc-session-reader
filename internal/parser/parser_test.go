package parser

import (
	"testing"
)

// --- FormatTimestamp ---

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "ISO 8601 with Z suffix",
			input: "2025-03-15T14:30:00Z",
			want:  "03-15 14:30",
		},
		{
			name:  "ISO 8601 with positive offset",
			input: "2025-03-15T14:30:00+08:00",
			want:  "03-15 14:30",
		},
		{
			name:  "ISO 8601 with negative offset",
			input: "2025-12-01T09:05:00-05:00",
			want:  "12-01 09:05",
		},
		{
			name:  "ISO 8601 with milliseconds",
			input: "2025-06-20T23:59:59.123+00:00",
			want:  "06-20 23:59",
		},
		{
			name:  "ISO 8601 with microseconds",
			input: "2025-01-01T00:00:00.000000+00:00",
			want:  "01-01 00:00",
		},
		{
			name:  "invalid string returns placeholder",
			input: "not-a-timestamp",
			want:  "??-?? ??:??",
		},
		{
			name:  "empty string returns placeholder",
			input: "",
			want:  "??-?? ??:??",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTimestamp(tt.input)
			if got != tt.want {
				t.Errorf("FormatTimestamp(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- ExtractText ---

func TestExtractText(t *testing.T) {
	tests := []struct {
		name    string
		content interface{}
		want    string
	}{
		{
			name:    "string content returns as-is",
			content: "Hello, world!",
			want:    "Hello, world!",
		},
		{
			name: "list with text blocks concatenated with newline",
			content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Line one"},
				map[string]interface{}{"type": "text", "text": "Line two"},
			},
			want: "Line one\nLine two",
		},
		{
			name: "list with mixed blocks only extracts text",
			content: []interface{}{
				map[string]interface{}{"type": "text", "text": "visible text"},
				map[string]interface{}{"type": "tool_use", "name": "Bash"},
				map[string]interface{}{"type": "text", "text": "more text"},
			},
			want: "visible text\nmore text",
		},
		{
			name:    "nil returns empty",
			content: nil,
			want:    "",
		},
		{
			name:    "empty list returns empty",
			content: []interface{}{},
			want:    "",
		},
		{
			name:    "integer returns empty (unsupported type)",
			content: 42,
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractText(tt.content)
			if got != tt.want {
				t.Errorf("ExtractText() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- GetToolUses ---

func TestGetToolUses(t *testing.T) {
	tests := []struct {
		name    string
		content interface{}
		wantLen int
	}{
		{
			name: "list with tool_use blocks",
			content: []interface{}{
				map[string]interface{}{"type": "text", "text": "intro"},
				map[string]interface{}{"type": "tool_use", "name": "Bash", "id": "t1"},
				map[string]interface{}{"type": "tool_use", "name": "Read", "id": "t2"},
			},
			wantLen: 2,
		},
		{
			name: "list without tool_use",
			content: []interface{}{
				map[string]interface{}{"type": "text", "text": "just text"},
			},
			wantLen: 0,
		},
		{
			name:    "nil returns empty",
			content: nil,
			wantLen: 0,
		},
		{
			name:    "string content returns empty",
			content: "not a list",
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetToolUses(tt.content)
			if len(got) != tt.wantLen {
				t.Errorf("GetToolUses() returned %d items, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// Verify that returned tool_use blocks preserve their fields.
func TestGetToolUses_PreservesFields(t *testing.T) {
	content := []interface{}{
		map[string]interface{}{
			"type":  "tool_use",
			"name":  "Bash",
			"id":    "tool-123",
			"input": map[string]interface{}{"command": "ls"},
		},
	}
	results := GetToolUses(content)
	if len(results) != 1 {
		t.Fatalf("expected 1 tool_use, got %d", len(results))
	}
	if results[0]["name"] != "Bash" {
		t.Errorf("name = %q, want %q", results[0]["name"], "Bash")
	}
	if results[0]["id"] != "tool-123" {
		t.Errorf("id = %q, want %q", results[0]["id"], "tool-123")
	}
}

// --- IsNoise ---

func TestIsNoise(t *testing.T) {
	// Exhaustively test each noise type defined in NoiseTypes
	for noiseType := range NoiseTypes {
		t.Run("noise_type_"+noiseType, func(t *testing.T) {
			entry := map[string]interface{}{"type": noiseType}
			if !IsNoise(entry) {
				t.Errorf("IsNoise should return true for type %q", noiseType)
			}
		})
	}

	// "system" is filtered by the || clause, not by NoiseTypes map
	t.Run("system_is_noise", func(t *testing.T) {
		entry := map[string]interface{}{"type": "system"}
		if !IsNoise(entry) {
			t.Error("IsNoise should return true for type 'system'")
		}
	})

	t.Run("user_is_not_noise", func(t *testing.T) {
		entry := map[string]interface{}{"type": "user"}
		if IsNoise(entry) {
			t.Error("IsNoise should return false for type 'user'")
		}
	})

	t.Run("assistant_is_not_noise", func(t *testing.T) {
		entry := map[string]interface{}{"type": "assistant"}
		if IsNoise(entry) {
			t.Error("IsNoise should return false for type 'assistant'")
		}
	})

	t.Run("empty_type_is_not_noise", func(t *testing.T) {
		entry := map[string]interface{}{}
		if IsNoise(entry) {
			t.Error("IsNoise should return false for entry with no type")
		}
	})
}

// --- ExtractToolResultText ---

func TestExtractToolResultText(t *testing.T) {
	tests := []struct {
		name          string
		entry         map[string]interface{}
		wantText      string
		wantToolUseID string
	}{
		{
			name: "content is string",
			entry: map[string]interface{}{
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type":        "tool_result",
							"tool_use_id": "tool-abc",
							"content":     "output text here",
						},
					},
				},
			},
			wantText:      "output text here",
			wantToolUseID: "tool-abc",
		},
		{
			name: "content is list of text blocks",
			entry: map[string]interface{}{
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type":        "tool_result",
							"tool_use_id": "tool-xyz",
							"content": []interface{}{
								map[string]interface{}{"type": "text", "text": "first part"},
								map[string]interface{}{"type": "text", "text": "second part"},
							},
						},
					},
				},
			},
			wantText:      "first part\nsecond part",
			wantToolUseID: "tool-xyz",
		},
		{
			name: "no tool_result block",
			entry: map[string]interface{}{
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "just text",
						},
					},
				},
			},
			wantText:      "",
			wantToolUseID: "",
		},
		{
			name:          "empty entry",
			entry:         map[string]interface{}{},
			wantText:      "",
			wantToolUseID: "",
		},
		{
			name: "no message field",
			entry: map[string]interface{}{
				"type": "user",
			},
			wantText:      "",
			wantToolUseID: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotID := ExtractToolResultText(tt.entry)
			if gotText != tt.wantText {
				t.Errorf("text = %q, want %q", gotText, tt.wantText)
			}
			if gotID != tt.wantToolUseID {
				t.Errorf("tool_use_id = %q, want %q", gotID, tt.wantToolUseID)
			}
		})
	}
}
