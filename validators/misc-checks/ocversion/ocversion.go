// Copyright 2015 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/pborman/getopt/v2"
)

// printOCVersions prints all files, along with their openconfig-version value, if present.
func printOCVersions(w io.Writer, entries []*yang.Entry) {
	for _, e := range entries {
		m, ok := e.Node.(*yang.Module)
		if !ok {
			fmt.Fprintf(os.Stderr, "error: cannot convert entry %q to *yang.Module", e.Name)
			continue
		}

		fmt.Fprintf(w, "%s.yang:", m.Name)

		for _, e := range m.Extensions {
			keywordParts := strings.Split(e.Keyword, ":")
			if len(keywordParts) != 2 {
				// Unrecognized extension declaration
				continue
			}
			pfx, ext := strings.TrimSpace(keywordParts[0]), strings.TrimSpace(keywordParts[1])
			if ext == "openconfig-version" {
				extMod := yang.FindModuleByPrefix(m, pfx)
				if extMod == nil {
					fmt.Fprintf(os.Stderr, "unable to find module using prefix %q from referencing module %q\n", pfx, m.Name)
				} else if extMod.Name == "openconfig-extensions" {
					fmt.Fprintf(w, " openconfig-version:%q", e.Argument)
				}
			}
		}

		fmt.Fprintf(w, "\n")
	}
}

func buildModuleEntries(paths, files []string) ([]*yang.Entry, []error) {
	var errs []error
	for _, path := range paths {
		expanded, err := yang.PathsWithModules(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		yang.AddPath(expanded...)
	}

	ms := yang.NewModules()

	for _, name := range files {
		if err := ms.Read(name); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	if errs != nil {
		return nil, errs
	}

	// Process the read files, exiting if any errors were found.
	if errs := ms.Process(); errs != nil {
		return nil, errs
	}

	// Keep track of the top level modules we read in.
	// Those are the only modules we want to print below.
	mods := map[string]*yang.Module{}
	var names []string

	for _, m := range ms.Modules {
		if _, ok := mods[m.Name]; !ok {
			mods[m.Name] = m
			names = append(names, m.Name)
		}
	}
	for _, m := range ms.SubModules {
		if _, ok := mods[m.Name]; !ok {
			mods[m.Name] = m
			names = append(names, m.Name)
		}
	}
	sort.Strings(names)
	entries := make([]*yang.Entry, len(names))
	for x, n := range names {
		entries[x] = yang.ToEntry(mods[n])
	}

	return entries, nil
}

func main() {
	pathsPtr := getopt.ListLong("path", 'p', "comma separated list of directories to add to search path", "DIR[,DIR...]")

	if err := getopt.Getopt(nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		getopt.PrintUsage(os.Stderr)
		os.Exit(1)
	}

	files := getopt.Args()

	entries, errs := buildModuleEntries(*pathsPtr, files)
	if errs != nil {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}

	printOCVersions(os.Stdout, entries)
}
