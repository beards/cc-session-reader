package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SessionListEntry unifies metadata-based and JSONL-based sessions for the list command.
type SessionListEntry struct {
	SessionID             string
	ProjectPath           string
	StartTime             string    // ISO 8601 timestamp string
	StartTimeParsed       time.Time // parsed for sorting
	DurationMinutes       int
	UserMessageCount      int
	AssistantMessageCount int
	FirstPrompt           string
	FromMeta              bool // true if from metadata, false if from JSONL fallback
}

// commandTagPrefixes are JSONL user message content prefixes that indicate
// tool/command noise rather than real user prompts. Filtered when extracting
// FirstPrompt from transcripts.
var commandTagPrefixes = []string{
	"<command-name>",
	"<local-command-stdout>",
	"<bash-input>",
	"<bash-stdout>",
	"<bash-stderr>",
	"<local-command-caveat>",
}

// jsonlHeaderEntry is a minimal struct for parsing the first lines of a JSONL
// transcript. Avoids importing claudecodec to keep dependency graph simple.
type jsonlHeaderEntry struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Message   *struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// ScanTranscriptHeaders walks ProjectsDir for .jsonl files and extracts a
// SessionListEntry from the first 20 lines of each file. Files that cannot be
// opened or parsed are silently skipped. Sessions with the same UUID in multiple
// project directories are deduplicated (first walk hit wins).
func (s Store) ScanTranscriptHeaders() []SessionListEntry {
	if s.ProjectsDir == "" {
		return nil
	}

	seen := make(map[string]bool)
	var entries []SessionListEntry

	_ = filepath.Walk(s.ProjectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}

		sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		if seen[sessionID] {
			return nil
		}

		entry, ok := scanJSONLHeader(path, sessionID, filepath.Base(filepath.Dir(path)))
		if !ok {
			return nil
		}

		seen[sessionID] = true
		entries = append(entries, entry)
		return nil
	})

	return entries
}

// scanJSONLHeader reads up to 20 lines of a JSONL file and extracts timestamp
// and first user prompt. Returns (entry, true) on success, (zero, false) if the
// file cannot be opened.
func scanJSONLHeader(path, sessionID, projectDir string) (SessionListEntry, bool) {
	f, err := os.Open(path)
	if err != nil {
		return SessionListEntry{}, false
	}
	defer f.Close()

	entry := SessionListEntry{
		SessionID:   sessionID,
		ProjectPath: projectDir,
		FromMeta:    false,
	}

	scanner := bufio.NewScanner(f)
	linesRead := 0
	for scanner.Scan() && linesRead < 20 {
		linesRead++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var h jsonlHeaderEntry
		if err := json.Unmarshal(line, &h); err != nil {
			continue
		}

		if entry.StartTime == "" && h.Timestamp != "" {
			ts := strings.Replace(h.Timestamp, "Z", "+00:00", 1)
			entry.StartTime = h.Timestamp
			if t, err := parseISO(ts); err == nil {
				entry.StartTimeParsed = t
			}
		}

		if entry.FirstPrompt == "" && h.Message != nil && h.Message.Role == "user" && h.Type == "user" {
			var text string
			if err := json.Unmarshal(h.Message.Content, &text); err == nil {
				if !hasCommandTagPrefix(text) {
					entry.FirstPrompt = truncateRunes(text, 80)
				}
			}
		}

		if entry.StartTime != "" && entry.FirstPrompt != "" {
			break
		}
	}

	return entry, true
}

// hasCommandTagPrefix reports whether s begins with any of the known command
// noise prefixes that should not be surfaced as FirstPrompt.
func hasCommandTagPrefix(s string) bool {
	for _, prefix := range commandTagPrefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// truncateRunes truncates s to at most maxRunes runes, appending "..." if truncation occurs.
func truncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-3]) + "..."
}

// ListAllSessions returns every discoverable session by merging metadata files
// (authoritative) with JSONL transcript headers (fallback for sessions whose
// metadata has not yet been flushed). Metadata entries always win when both
// sources contain the same session ID.
//
// The returned entries are sorted newest-first by StartTimeParsed; entries with
// zero (unparseable) timestamps sort after all timestamped entries.
//
// The second return value contains warning messages for metadata files that
// could not be read or parsed; these are non-fatal and the remaining entries
// are still returned.
func (s Store) ListAllSessions() ([]SessionListEntry, []string) {
	var warnings []string

	// --- metadata entries (authoritative) ---
	metaFiles, _ := s.ListSessionMetaFiles()
	metaEntries := make(map[string]SessionListEntry)
	for _, mf := range metaFiles {
		data, err := os.ReadFile(mf.Path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("warning: skipping unreadable metadata %s: %v", mf.Path, err))
			continue
		}
		data = SanitizeMetaJSON(data)
		var m sessionMetaJSON
		if err := json.Unmarshal(data, &m); err != nil {
			warnings = append(warnings, fmt.Sprintf("warning: skipping invalid metadata %s: %v", mf.Path, err))
			continue
		}

		sid := m.SessionID
		if sid == "" {
			sid = strings.TrimSuffix(filepath.Base(mf.Path), ".json")
		}

		ts := strings.Replace(m.StartTime, "Z", "+00:00", 1)
		var startParsed time.Time
		if t, err := parseISO(ts); err == nil {
			startParsed = t
		}

		metaEntries[sid] = SessionListEntry{
			SessionID:             sid,
			ProjectPath:           m.ProjectPath,
			StartTime:             m.StartTime,
			StartTimeParsed:       startParsed,
			DurationMinutes:       m.DurationMinutes,
			UserMessageCount:      m.UserMessageCount,
			AssistantMessageCount: m.AssistantMessageCount,
			FirstPrompt:           truncateRunes(m.FirstPrompt, 80),
			FromMeta:              true,
		}
	}

	// --- JSONL fallback entries ---
	transcriptEntries := s.ScanTranscriptHeaders()

	// --- merge: metadata wins, JSONL fills gaps ---
	all := make([]SessionListEntry, 0, len(metaEntries)+len(transcriptEntries))
	for _, e := range metaEntries {
		all = append(all, e)
	}
	for _, e := range transcriptEntries {
		if !metaEntries[e.SessionID].FromMeta {
			all = append(all, e)
		}
	}

	// --- sort newest first; zero timestamps sort last ---
	sort.Slice(all, func(i, j int) bool {
		ti, tj := all[i].StartTimeParsed, all[j].StartTimeParsed
		iZero, jZero := ti.IsZero(), tj.IsZero()
		if iZero != jZero {
			return jZero // non-zero before zero
		}
		if iZero {
			return false // both zero: stable (preserve append order)
		}
		return ti.After(tj)
	})

	return all, warnings
}
