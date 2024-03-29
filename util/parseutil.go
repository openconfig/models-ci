// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package util contain utility functions for doing YANG model validation.
package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/prototext"

	pb "github.com/openconfig/models-ci/proto/results"
)

const (
	// PYANG_MSG_TEMPLATE_STRING sets up an output template for pyang using
	// its commandline option --msg-template.
	PYANG_MSG_TEMPLATE_STRING = `PYANG_MSG_TEMPLATE='messages:{{path:"{file}" line:{line} code:"{code}" type:"{type}" level:{level} message:'"'{msg}'}}"`
)

var (
	// stdErrorRegex recognizes the error/warning lines from pyang/confd
	// It currently recognizes the following two patterns:
	// - path:line#:status:message
	// - path:line#(subpath:line#):status:message
	//     NOTE: The subpath info in brackets is currently lumped into one group.
	// TODO(wenovus): Should use --msg-template to ingest pyang output as
	// textproto instead of using regex.
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

// ParsePyangTextprotoOutput parses textproto-formatted pyang output into a
// proto message. It assumes that the input string has format
// defined by PYANG_MSG_TEMPLATE_STRING.
func ParsePyangTextprotoOutput(textprotoOut string) (*pb.PyangOutput, error) {
	output := &pb.PyangOutput{}

	// Go through each line, and escape single quotes within the error
	// message so that they can be parsed by prototext.Unmarshal.
	var escapedOutput []byte
	const messageStart = "message:'"
	for _, line := range strings.Split(textprotoOut, "\n") {
		if len(line) == 0 {
			continue
		}
		i := strings.Index(line, messageStart)
		if i == -1 {
			return nil, fmt.Errorf("pyang output contains unrecognized line: %q", line)
		}
		i += len(messageStart)
		j := strings.LastIndex(line, "'")
		lineBytes := []byte(line)
		escapedOutput = append(escapedOutput, lineBytes[:i]...)
		for _, c := range lineBytes[i:j] {
			if c == '\'' {
				escapedOutput = append(escapedOutput, '\\')
			}
			escapedOutput = append(escapedOutput, c)
		}
		escapedOutput = append(escapedOutput, lineBytes[j:]...)
	}

	err := prototext.Unmarshal(escapedOutput, output)
	return output, err
}
