package summarizer

import (
	"strings"
	"testing"
)

func TestSummarizeToolUse_Bash_WithDescription(t *testing.T) {
	inp := map[string]interface{}{
		"command":     "ls -la /some/path",
		"description": "List files in directory",
	}
	got := SummarizeToolUse("Bash", inp)
	want := "[Bash] List files in directory"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSummarizeToolUse_Bash_WithoutDescription(t *testing.T) {
	inp := map[string]interface{}{
		"command": "ls -la /some/path",
	}
	got := SummarizeToolUse("Bash", inp)
	want := "[Bash] ls -la /some/path"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSummarizeToolUse_Bash_LongCommandTruncation(t *testing.T) {
	// Command longer than 80 runes should be truncated
	longCmd := strings.Repeat("a", 100)
	inp := map[string]interface{}{
		"command": longCmd,
	}
	got := SummarizeToolUse("Bash", inp)
	// "[Bash] " prefix + 80 runes of command
	wantPrefix := "[Bash] "
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("expected prefix %q, got %q", wantPrefix, got)
	}
	commandPart := strings.TrimPrefix(got, wantPrefix)
	if len([]rune(commandPart)) != 80 {
		t.Errorf("command part should be %d runes, got %d", 80, len([]rune(commandPart)))
	}
}

// Guards against byte-vs-rune truncation bug:
// CJK characters are 3 bytes each in UTF-8, so a string with 50 CJK chars
// is 150 bytes but only 50 runes. It must NOT be truncated since 50 < 80.
func TestSummarizeToolUse_Bash_CJKRuneVsByteTruncation(t *testing.T) {
	// 50 CJK characters = 150 bytes, but only 50 runes (< 80)
	cjkCmd := strings.Repeat("世", 50) // "世" repeated 50 times
	inp := map[string]interface{}{
		"command": cjkCmd,
	}
	got := SummarizeToolUse("Bash", inp)
	want := "[Bash] " + cjkCmd
	if got != want {
		t.Errorf("50 CJK runes should NOT be truncated.\ngot  %q\nwant %q", got, want)
	}
}

// CJK string longer than 80 runes should be truncated at the rune boundary.
func TestSummarizeToolUse_Bash_CJKLongTruncation(t *testing.T) {
	cjkCmd := strings.Repeat("世", 100) // 100 CJK runes
	inp := map[string]interface{}{
		"command": cjkCmd,
	}
	got := SummarizeToolUse("Bash", inp)
	commandPart := strings.TrimPrefix(got, "[Bash] ")
	runes := []rune(commandPart)
	if len(runes) != 80 {
		t.Errorf("expected %d runes after truncation, got %d", 80, len(runes))
	}
	// Each rune should still be the original CJK character (no mid-byte corruption)
	for i, r := range runes {
		if r != '世' {
			t.Errorf("rune %d corrupted: got %U, want U+4E16", i, r)
			break
		}
	}
}

func TestSummarizeToolUse_Read(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "deep path shows last 2 segments",
			path: "/Users/maple/project/internal/parser/parser.go",
			want: "[Read] parser/parser.go",
		},
		{
			name: "single segment path",
			path: "file.txt",
			want: "[Read] file.txt",
		},
		{
			name: "empty path shows question mark",
			path: "",
			want: "[Read] ?",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inp := map[string]interface{}{"file_path": tt.path}
			got := SummarizeToolUse("Read", inp)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSummarizeToolUse_EditAndWrite(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		path     string
		want     string
	}{
		{
			name:     "Edit extracts filename from path",
			toolName: "Edit",
			path:     "/Users/maple/project/main.go",
			want:     "[Edit] main.go",
		},
		{
			name:     "Write extracts filename from path",
			toolName: "Write",
			path:     "/Users/maple/project/config.yaml",
			want:     "[Write] config.yaml",
		},
		{
			name:     "Edit with no slash returns full path",
			toolName: "Edit",
			path:     "standalone.txt",
			want:     "[Edit] standalone.txt",
		},
		{
			name:     "Write with empty path shows question mark",
			toolName: "Write",
			path:     "",
			want:     "[Write] ?",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inp := map[string]interface{}{"file_path": tt.path}
			got := SummarizeToolUse(tt.toolName, inp)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSummarizeToolUse_Agent(t *testing.T) {
	tests := []struct {
		name string
		inp  map[string]interface{}
		want string
	}{
		{
			name: "with subagent_type",
			inp: map[string]interface{}{
				"description":   "Explore codebase structure",
				"subagent_type": "explorer",
			},
			want: "[Agent(explorer)] Explore codebase structure",
		},
		{
			name: "without subagent_type",
			inp: map[string]interface{}{
				"description": "Implement feature X",
			},
			want: "[Agent] Implement feature X",
		},
		{
			name: "empty description fallback",
			inp:  map[string]interface{}{},
			want: "[Agent] ?",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeToolUse("Agent", tt.inp)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSummarizeToolUse_AskUserQuestion_SingleQuestion(t *testing.T) {
	inp := map[string]interface{}{
		"questions": []interface{}{
			map[string]interface{}{"question": "What framework do you prefer?"},
		},
	}
	got := SummarizeToolUse("AskUserQuestion", inp)
	want := "[AskUserQuestion] Q1: What framework do you prefer?"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSummarizeToolUse_AskUserQuestion_ThreeQuestions(t *testing.T) {
	inp := map[string]interface{}{
		"questions": []interface{}{
			map[string]interface{}{"question": "First question?"},
			map[string]interface{}{"question": "Second question?"},
			map[string]interface{}{"question": "Third question?"},
		},
	}
	got := SummarizeToolUse("AskUserQuestion", inp)
	want := "[AskUserQuestion] Q1: First question?\n  [AskUserQuestion] Q2: Second question?\n  [AskUserQuestion] Q3: Third question?"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSummarizeToolUse_AskUserQuestion_NoQuestions(t *testing.T) {
	inp := map[string]interface{}{}
	got := SummarizeToolUse("AskUserQuestion", inp)
	if got != "[AskUserQuestion]" {
		t.Errorf("got %q, want %q", got, "[AskUserQuestion]")
	}
}

func TestSummarizeToolUse_Skill(t *testing.T) {
	tests := []struct {
		name string
		inp  map[string]interface{}
		want string
	}{
		{
			name: "short skill",
			inp:  map[string]interface{}{"skill": "pm", "args": "build login page"},
			want: "[Skill] /pm build login page",
		},
		{
			// "[Skill] /develop " = 17 runes, truncated to 80 total = 17 + 63 x's
			name: "skill truncated at 80 runes",
			inp: map[string]interface{}{
				"skill": "develop",
				"args":  strings.Repeat("x", 100),
			},
			want: "[Skill] /develop " + strings.Repeat("x", 63),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeToolUse("Skill", tt.inp)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if len([]rune(got)) > 80 {
				t.Errorf("skill summary exceeds 80 runes: %d", len([]rune(got)))
			}
		})
	}
}

func TestSummarizeToolUse_Grep(t *testing.T) {
	tests := []struct {
		name string
		inp  map[string]interface{}
		want string
	}{
		{
			name: "with path",
			inp:  map[string]interface{}{"pattern": "TODO", "path": "./src"},
			want: `[Grep] "TODO" in ./src`,
		},
		{
			name: "without path",
			inp:  map[string]interface{}{"pattern": "FIXME"},
			want: `[Grep] "FIXME"`,
		},
		{
			name: "empty pattern",
			inp:  map[string]interface{}{},
			want: `[Grep] "?"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeToolUse("Grep", tt.inp)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSummarizeToolUse_Glob(t *testing.T) {
	inp := map[string]interface{}{"pattern": "**/*.go"}
	got := SummarizeToolUse("Glob", inp)
	want := "[Glob] **/*.go"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSummarizeToolUse_UnknownTool(t *testing.T) {
	got := SummarizeToolUse("WebSearch", map[string]interface{}{})
	want := "[WebSearch]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- SummarizeToolResult ---

func TestSummarizeToolResult(t *testing.T) {
	tests := []struct {
		name  string
		entry map[string]interface{}
		want  string
	}{
		{
			name: "success with content",
			entry: map[string]interface{}{
				"toolUseResult": map[string]interface{}{"success": true},
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type":    "tool_result",
							"content": "Command completed successfully",
						},
					},
				},
			},
			want: " -> ok: Command completed successfully",
		},
		{
			name: "failure with content",
			entry: map[string]interface{}{
				"toolUseResult": map[string]interface{}{"success": false},
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type":    "tool_result",
							"content": "Permission denied",
						},
					},
				},
			},
			want: " -> FAILED: Permission denied",
		},
		{
			name: "success without content",
			entry: map[string]interface{}{
				"toolUseResult": map[string]interface{}{"success": true},
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type":    "tool_result",
							"content": "",
						},
					},
				},
			},
			want: " -> ok",
		},
		{
			name:  "no toolUseResult field",
			entry: map[string]interface{}{},
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeToolResult(tt.entry)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// --- IsUserAnswer ---

func TestIsUserAnswer(t *testing.T) {
	tests := []struct {
		name  string
		entry map[string]interface{}
		want  bool
	}{
		{
			name: "prefix: User has answered",
			entry: map[string]interface{}{
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type":    "tool_result",
							"content": "User has answered your questions: yes I agree",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "prefix: Your questions have been answered",
			entry: map[string]interface{}{
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type":    "tool_result",
							"content": "Your questions have been answered: the answer is 42",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "normal tool result is not user answer",
			entry: map[string]interface{}{
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type":    "tool_result",
							"content": "Command output here",
						},
					},
				},
			},
			want: false,
		},
		{
			name:  "empty entry",
			entry: map[string]interface{}{},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUserAnswer(tt.entry)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// --- ExtractUserAnswers ---

func TestExtractUserAnswers(t *testing.T) {
	tests := []struct {
		name  string
		entry map[string]interface{}
		want  string
	}{
		{
			name: "extracts full answer text",
			entry: map[string]interface{}{
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type":    "tool_result",
							"content": "User has answered your questions: deploy to staging",
						},
					},
				},
			},
			want: "User has answered your questions: deploy to staging",
		},
		{
			name:  "empty entry returns empty",
			entry: map[string]interface{}{},
			want:  "",
		},
		{
			name: "non-answer tool result returns empty",
			entry: map[string]interface{}{
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type":    "tool_result",
							"content": "regular output",
						},
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractUserAnswers(tt.entry)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// --- truncate (tested indirectly via SummarizeToolUse) ---

func TestTruncate_ViaToolUse(t *testing.T) {
	tests := []struct {
		name    string
		command string
		// wantLen is the expected rune count of the command portion (after "[Bash] ")
		wantLen       int
		wantUnchanged bool
	}{
		{
			name:          "shorter than max is unchanged",
			command:       "echo hello",
			wantUnchanged: true,
		},
		{
			name:          "ASCII at exactly max is unchanged",
			command:       strings.Repeat("x", 80),
			wantUnchanged: true,
		},
		{
			name:    "ASCII longer than max is truncated",
			command: strings.Repeat("x", 80+20),
			wantLen: 80,
		},
		{
			name:          "CJK under max runes but over max bytes is NOT truncated",
			command:       strings.Repeat("测", 70), // 70 runes, 210 bytes
			wantUnchanged: true,
		},
		{
			name:    "CJK over max runes is truncated at rune boundary",
			command: strings.Repeat("测", 100), // 100 runes
			wantLen: 80,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inp := map[string]interface{}{"command": tt.command}
			got := SummarizeToolUse("Bash", inp)
			commandPart := strings.TrimPrefix(got, "[Bash] ")

			if tt.wantUnchanged {
				if commandPart != tt.command {
					t.Errorf("expected command unchanged, but got different result.\ngot  rune-len=%d\nwant rune-len=%d",
						len([]rune(commandPart)), len([]rune(tt.command)))
				}
				return
			}
			gotLen := len([]rune(commandPart))
			if gotLen != tt.wantLen {
				t.Errorf("expected %d runes, got %d", tt.wantLen, gotLen)
			}
		})
	}
}
