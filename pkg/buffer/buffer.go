package buffer

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
)

// ProcessResponseAsRingBufferToEnd reads the body of an HTTP response line by line,
// storing only the last maxJobLogLines lines using a ring buffer (sliding window).
// This efficiently retains the most recent lines, overwriting older ones as needed.
//
// Parameters:
//
//	httpResp:        The HTTP response whose body will be read.
//	maxJobLogLines:  The maximum number of log lines to retain.
//
// Returns:
//
//	string:          The concatenated log lines (up to maxJobLogLines), separated by newlines.
//	int:             The total number of lines read from the response.
//	*http.Response:  The original HTTP response.
//	error:           Any error encountered during reading.
//
// The function uses a ring buffer to efficiently store only the last maxJobLogLines lines.
// If the response contains more lines than maxJobLogLines, only the most recent lines are kept.
func ProcessResponseAsRingBufferToEnd(httpResp *http.Response, maxJobLogLines int) (string, int, *http.Response, error) {
	if maxJobLogLines <= 0 {
		return "", 0, httpResp, nil
	}

	lines := make([]string, maxJobLogLines)
	totalLines := 0
	writeIndex := 0

	scanner := bufio.NewScanner(httpResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		lines[writeIndex] = scanner.Text()
		totalLines++
		writeIndex = (writeIndex + 1) % maxJobLogLines
	}

	if err := scanner.Err(); err != nil {
		return "", 0, httpResp, fmt.Errorf("failed to read log content: %w", err)
	}

	// Calculate how many lines we actually have in the buffer
	linesInBuffer := totalLines
	if linesInBuffer > maxJobLogLines {
		linesInBuffer = maxJobLogLines
	}

	// Pre-allocate the result slice for efficiency
	result := make([]string, linesInBuffer)

	// Determine the starting index for reading from the ring buffer
	startIndex := 0
	if totalLines > maxJobLogLines {
		startIndex = writeIndex
	}

	// Copy lines from ring buffer to result in correct order
	for i := 0; i < linesInBuffer; i++ {
		result[i] = lines[(startIndex+i)%maxJobLogLines]
	}

	return strings.Join(result, "\n"), totalLines, httpResp, nil
}
