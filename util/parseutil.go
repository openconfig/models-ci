// Package util contain utility functions for doing YANG model validation.
package util

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// stdErrorRegex recognizes the error/warning lines from pyang/confd
	// It currently recognizes the following two patterns:
	// - path:line#:status:message
	// - path:line#(subpath:line#):status:message
	//     NOTE: The subpath info in brackets is currently lumped into one group.
	stdErrorRegex = regexp.MustCompile(`^([^:]+):\s*(\d+)\s*(\([^\)]+\))?\s*:([^:]+):(.+)$`)
)

// StandardErrorLine contains a parsed commandline output from pyang.
type StandardErrorLine struct {
	Path    string
	LineNo  int32
	Status  string
	Message string
}

// StandardOutput contains the parsed commandline outputs from pyang.
type StandardOutput struct {
	ErrorLines   []*StandardErrorLine
	WarningLines []*StandardErrorLine
	OtherLines   []string
}

// ParseStandardOutput parses raw pyang/confd output into a structured format.
// It recognizes two formats of output from pyang and confD:
// <file path>:<line no>:<error/warning>:<message>
// <file path>:<line#>(<import file path>:<line#>):<error/warning>:<message>
func ParseStandardOutput(rawOut string) StandardOutput {
	var out StandardOutput
	for _, line := range strings.Split(rawOut, "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		matches := stdErrorRegex.FindStringSubmatch(line)
		if matches == nil {
			out.OtherLines = append(out.OtherLines, line)
			continue
		}

		filePath := strings.TrimSpace(matches[1])
		lineNumber, err := strconv.ParseInt(strings.TrimSpace(matches[2]), 10, 32)
		if err != nil {
			out.OtherLines = append(out.OtherLines, line)
			continue
		}
		status := strings.ToLower(strings.TrimSpace(matches[4]))
		message := strings.TrimSpace(matches[5])

		switch {
		case strings.Contains(status, "error"):
			out.ErrorLines = append(out.ErrorLines, &StandardErrorLine{
				Path:    filePath,
				LineNo:  int32(lineNumber),
				Status:  status,
				Message: message,
			})
		case strings.Contains(status, "warning"):
			out.WarningLines = append(out.WarningLines, &StandardErrorLine{
				Path:    filePath,
				LineNo:  int32(lineNumber),
				Status:  status,
				Message: message,
			})
		default: // Unrecognized line, so classify as "other".
			out.OtherLines = append(out.OtherLines, line)
		}
	}
	return out
}
