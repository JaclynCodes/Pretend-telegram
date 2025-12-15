package buffer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// mockHTTPResponse creates a mock HTTP response with the given body content
func mockHTTPResponse(body string) *http.Response {
	return &http.Response{
		Body: io.NopCloser(bytes.NewBufferString(body)),
	}
}

func TestProcessResponseAsRingBufferToEnd_BasicFunctionality(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		maxLines         int
		expectedLines    string
		expectedTotal    int
		expectedLineList []string
	}{
		{
			name:             "fewer lines than buffer size",
			body:             "line1\nline2\nline3",
			maxLines:         5,
			expectedLines:    "line1\nline2\nline3",
			expectedTotal:    3,
			expectedLineList: []string{"line1", "line2", "line3"},
		},
		{
			name:             "exact buffer size",
			body:             "line1\nline2\nline3",
			maxLines:         3,
			expectedLines:    "line1\nline2\nline3",
			expectedTotal:    3,
			expectedLineList: []string{"line1", "line2", "line3"},
		},
		{
			name:             "more lines than buffer size keeps last N",
			body:             "line1\nline2\nline3\nline4\nline5",
			maxLines:         3,
			expectedLines:    "line3\nline4\nline5",
			expectedTotal:    5,
			expectedLineList: []string{"line3", "line4", "line5"},
		},
		{
			name:             "single line",
			body:             "only line",
			maxLines:         10,
			expectedLines:    "only line",
			expectedTotal:    1,
			expectedLineList: []string{"only line"},
		},
		{
			name:             "empty body",
			body:             "",
			maxLines:         5,
			expectedLines:    "",
			expectedTotal:    0,
			expectedLineList: []string{},
		},
		{
			name:             "buffer size of 1",
			body:             "line1\nline2\nline3",
			maxLines:         1,
			expectedLines:    "line3",
			expectedTotal:    3,
			expectedLineList: []string{"line3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := mockHTTPResponse(tt.body)
			result, totalLines, returnedResp, err := ProcessResponseAsRingBufferToEnd(resp, tt.maxLines)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if totalLines != tt.expectedTotal {
				t.Errorf("totalLines = %d, want %d", totalLines, tt.expectedTotal)
			}

			if result != tt.expectedLines {
				t.Errorf("result = %q, want %q", result, tt.expectedLines)
			}

			if returnedResp != resp {
				t.Error("returned response should be the same as input response")
			}

			// Verify the individual lines
			var resultLines []string
			if result != "" {
				resultLines = strings.Split(result, "\n")
			}
			if len(resultLines) != len(tt.expectedLineList) {
				t.Errorf("got %d lines, want %d lines", len(resultLines), len(tt.expectedLineList))
			}
			for i, line := range tt.expectedLineList {
				if i < len(resultLines) && resultLines[i] != line {
					t.Errorf("line %d = %q, want %q", i, resultLines[i], line)
				}
			}
		})
	}
}

func TestProcessResponseAsRingBufferToEnd_EdgeCases(t *testing.T) {
	t.Run("zero max lines returns empty", func(t *testing.T) {
		resp := mockHTTPResponse("line1\nline2")
		result, totalLines, _, err := ProcessResponseAsRingBufferToEnd(resp, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("result should be empty for zero max lines, got %q", result)
		}
		if totalLines != 0 {
			t.Errorf("totalLines should be 0, got %d", totalLines)
		}
	})

	t.Run("negative max lines returns empty", func(t *testing.T) {
		resp := mockHTTPResponse("line1\nline2")
		result, totalLines, _, err := ProcessResponseAsRingBufferToEnd(resp, -1)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("result should be empty for negative max lines, got %q", result)
		}
		if totalLines != 0 {
			t.Errorf("totalLines should be 0, got %d", totalLines)
		}
	})

	t.Run("lines with empty strings", func(t *testing.T) {
		resp := mockHTTPResponse("line1\n\nline3")
		result, totalLines, _, err := ProcessResponseAsRingBufferToEnd(resp, 5)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if totalLines != 3 {
			t.Errorf("totalLines = %d, want 3", totalLines)
		}
		expected := "line1\n\nline3"
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("very long lines", func(t *testing.T) {
		longLine := strings.Repeat("x", 10000)
		body := longLine + "\n" + "short"
		resp := mockHTTPResponse(body)
		result, totalLines, _, err := ProcessResponseAsRingBufferToEnd(resp, 5)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if totalLines != 2 {
			t.Errorf("totalLines = %d, want 2", totalLines)
		}
		lines := strings.Split(result, "\n")
		if len(lines) != 2 {
			t.Errorf("expected 2 lines, got %d", len(lines))
		}
		if lines[0] != longLine {
			t.Error("first line should be the long line")
		}
		if lines[1] != "short" {
			t.Errorf("second line = %q, want %q", lines[1], "short")
		}
	})
}

func TestProcessResponseAsRingBufferToEnd_RingBufferCorrectness(t *testing.T) {
	// This test ensures the ring buffer correctly wraps around
	t.Run("ring buffer wraparound", func(t *testing.T) {
		// Create 10 lines but only keep last 3
		lines := make([]string, 10)
		for i := range 10 {
			lines[i] = strings.Repeat(string(rune('a'+i)), i+1) // a, bb, ccc, dddd, etc.
		}
		body := strings.Join(lines, "\n")
		resp := mockHTTPResponse(body)

		result, totalLines, _, err := ProcessResponseAsRingBufferToEnd(resp, 3)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if totalLines != 10 {
			t.Errorf("totalLines = %d, want 10", totalLines)
		}

		// Should have the last 3 lines: hhhhhhhh, iiiiiiiii, jjjjjjjjjj
		expected := strings.Join([]string{lines[7], lines[8], lines[9]}, "\n")
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})

	t.Run("multiple wraparounds", func(t *testing.T) {
		// Create 100 lines but only keep last 7
		lines := make([]string, 100)
		for i := range 100 {
			lines[i] = fmt.Sprintf("line%03d", i)
		}
		body := strings.Join(lines, "\n")
		resp := mockHTTPResponse(body)

		result, totalLines, _, err := ProcessResponseAsRingBufferToEnd(resp, 7)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if totalLines != 100 {
			t.Errorf("totalLines = %d, want 100", totalLines)
		}

		// Should have the last 7 lines
		expected := strings.Join(lines[93:100], "\n")
		if result != expected {
			t.Errorf("result = %q, want %q", result, expected)
		}
	})
}
