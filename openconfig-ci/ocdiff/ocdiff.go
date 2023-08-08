// Copyright 2023 Google Inc.
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

// ocdiff produces a report between two sets of OpenConfig YANG files.
package ocdiff

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/goyang/pkg/yangentry"
	"golang.org/x/exp/slices"
)

// NewDiffReport returns a diff report given options for compiling two sets of
// YANG files.
func NewDiffReport(oldpaths, newpaths, oldfiles, newfiles []string) (*DiffReport, error) {
	oldEntries, oldModuleVersions, err := flattenedEntries(oldpaths, oldfiles)
	if err != nil {
		return nil, err
	}

	newEntries, newModuleVersions, err := flattenedEntries(newpaths, newfiles)
	if err != nil {
		return nil, err
	}

	return diffMaps(oldEntries, newEntries, oldModuleVersions, newModuleVersions), nil
}

// yangNodeInfo contains all information of a single new/deleted node necessary
// for printing a report.
type yangNodeInfo struct {
	path              string
	schema            *yang.Entry
	allowIncompat     bool
	versionChangeDesc string
}

// yangNodeUpdateInfo contains all information of a single updated node necessary
// for printing a report.
type yangNodeUpdateInfo struct {
	path              string
	oldSchema         *yang.Entry
	newSchema         *yang.Entry
	allowIncompat     bool
	versionChangeDesc string
	incompatComments  []string
}

// DiffReport contains information necessary to print out a diff report between
// two sets of OpenConfig YANG files.
type DiffReport struct {
	newNodes          []*yangNodeInfo
	updatedNodes      []*yangNodeUpdateInfo
	deletedNodes      []*yangNodeInfo
	oldModuleVersions map[string]*semver.Version
	newModuleVersions map[string]*semver.Version
}

// Option can be used to modify the report outputs.
type Option func(*reportOptions)

// WithDisallowedIncompatsOnly indicates to report only backward-incompatible
// changes disallowed by version increments.
func WithDisallowedIncompatsOnly() Option {
	return func(o *reportOptions) {
		o.onlyReportDisallowedIncompats = true
	}
}

// WithGithubCommentStyle indicates to report with GitHub comment styling.
func WithGithubCommentStyle() Option {
	return func(o *reportOptions) {
		o.githubComment = true
	}
}

// resolveOpts applies all the options and returns a struct containing the result.
func resolveOpts(opts []Option) *reportOptions {
	o := &reportOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type reportOptions struct {
	onlyReportDisallowedIncompats bool
	githubComment                 bool
}

// Report outputs a report on the diff between the two sets of OpenConfig YANG files.
func (r *DiffReport) Report(options ...Option) string {
	opts := resolveOpts(options)
	r.Sort()
	fmtstr := "%s %s: %s (%s)\n"
	if opts.githubComment {
		fmtstr = "%s %s: `%s`\n* (%s)\n\n"
	}
	var b strings.Builder
	for _, del := range r.deletedNodes {
		if !opts.onlyReportDisallowedIncompats || !del.allowIncompat {
			if del.schema.IsLeaf() || del.schema.IsLeafList() {
				b.WriteString(fmt.Sprintf(fmtstr, "leaf", "deleted", del.path, del.versionChangeDesc))
			}
		}
	}
	for _, upd := range r.updatedNodes {
		nodeTypeDesc := "non-leaf"
		if upd.oldSchema.IsLeaf() || upd.oldSchema.IsLeafList() {
			nodeTypeDesc = "leaf"
		}
		if len(upd.incompatComments) > 0 {
			fmtstr := "%s updated: %s: %s (%s)\n"
			comments := strings.Join(upd.incompatComments, "\n\t")
			if opts.githubComment {
				fmtstr = "%s updated: `%s`\n* %s\n* (%s)\n\n"
				comments = strings.Join(upd.incompatComments, "\n* ")
			}
			b.WriteString(fmt.Sprintf(fmtstr, nodeTypeDesc, upd.path, comments, upd.versionChangeDesc))
		} else {
			b.WriteString(fmt.Sprintf(fmtstr, nodeTypeDesc, "updated", upd.path, upd.versionChangeDesc))
		}
	}
	if !opts.onlyReportDisallowedIncompats {
		for _, added := range r.newNodes {
			if added.schema.IsLeaf() || added.schema.IsLeafList() {
				b.WriteString(fmt.Sprintf(fmtstr, "leaf", "added", added.path, added.versionChangeDesc))
			}
		}
	}
	return b.String()
}

func (r *DiffReport) Sort() {
	slices.SortFunc(r.newNodes, func(a, b *yangNodeInfo) int { return strings.Compare(a.path, b.path) })
	slices.SortFunc(r.deletedNodes, func(a, b *yangNodeInfo) int { return strings.Compare(a.path, b.path) })
	slices.SortFunc(r.updatedNodes, func(a, b *yangNodeUpdateInfo) int { return strings.Compare(a.path, b.path) })
}

func getKind(e *yang.Entry) string {
	if e.Type != nil {
		return fmt.Sprint(e.Type.Kind)
	} else {
		return fmt.Sprint(e.Kind)
	}
}

func definingModuleName(e *yang.Entry) string {
	if e == nil {
		return ""
	}
	if definingModule := yang.RootNode(e.Node); definingModule != nil {
		return belongingModule(definingModule)
	}
	return ""
}

func (r *DiffReport) getModuleAndVersions(e *yang.Entry) (string, *semver.Version, *semver.Version) {
	moduleName := definingModuleName(e)
	return moduleName, r.oldModuleVersions[moduleName], r.newModuleVersions[moduleName]
}

func incompatAllowed(oldVersion, newVersion *semver.Version) bool {
	switch {
	case oldVersion == nil, newVersion == nil:
		// This can happen if the openconfig-version is not found (e.g. in IETF modules).
		//
		// In other cases, we will just be conservative and allow the
		// incompatibility since we don't want to block the PR.
		return true
	case oldVersion.Major() == 0:
		return true
	case newVersion.Major() > oldVersion.Major():
		return true
	default:
		return false
	}
}

func (r *DiffReport) addPair(o *yang.Entry, n *yang.Entry) error {
	moduleName, oldVersion, newVersion := r.getModuleAndVersions(o)
	versionChangeDesc := fmt.Sprintf("%q: openconfig-version %v -> %v", moduleName, oldVersion, newVersion)
	allowIncompat := incompatAllowed(oldVersion, newVersion)

	switch {
	case o == nil && n == nil:
	case o == nil:
		newModuleName, oldVersion, newVersion := r.getModuleAndVersions(n)
		r.newNodes = append(r.newNodes, &yangNodeInfo{
			schema:            n,
			path:              n.Path(),
			versionChangeDesc: fmt.Sprintf("%q: openconfig-version %v -> %v", newModuleName, oldVersion, newVersion),
		})
	case n == nil:
		r.deletedNodes = append(r.deletedNodes, &yangNodeInfo{
			schema:            o,
			path:              o.Path(),
			allowIncompat:     allowIncompat,
			versionChangeDesc: versionChangeDesc,
		})
	default:
		upd := &yangNodeUpdateInfo{
			oldSchema:         o,
			newSchema:         n,
			path:              o.Path(),
			allowIncompat:     allowIncompat,
			versionChangeDesc: versionChangeDesc,
		}
		updated := false
		if oldKind, newKind := getKind(o), getKind(n); oldKind != newKind {
			upd.incompatComments = append(upd.incompatComments, fmt.Sprintf("type changed from %s to %s", oldKind, newKind))
			updated = true
		}
		if updated {
			r.updatedNodes = append(r.updatedNodes, upd)
		}
	}
	return nil
}

// belongingModule returns the module name if m is a module and the belonging
// module name if m is a submodule.
func belongingModule(m *yang.Module) string {
	if m.Kind() == "submodule" {
		return m.BelongsTo.Name
	}
	return m.Name
}

func getOpenConfigModuleVersion(e *yang.Entry) (*semver.Version, error) {
	m, ok := e.Node.(*yang.Module)
	if !ok {
		return nil, fmt.Errorf("cannot convert entry %q to *yang.Module", e.Name)
	}

	for _, e := range m.Extensions {
		keywordParts := strings.Split(e.Keyword, ":")
		if len(keywordParts) != 2 {
			// Unrecognized extension declaration
			continue
		}
		pfx, ext := strings.TrimSpace(keywordParts[0]), strings.TrimSpace(keywordParts[1])
		if ext == "openconfig-version" {
			if extMod := yang.FindModuleByPrefix(m, pfx); extMod != nil && belongingModule(extMod) == "openconfig-extensions" {
				v, err := semver.StrictNewVersion(e.Argument)
				if err != nil {
					return nil, err
				}
				return v, nil
			}
		}
	}
	return nil, fmt.Errorf("did not find openconfig-extensions:openconfig-version statement in module %q", m.Name)
}

func flattenedEntries(paths, files []string) (map[string]*yang.Entry, map[string]*semver.Version, error) {
	moduleEntryMap, errs := yangentry.Parse(files, paths)
	if errs != nil {
		return nil, nil, fmt.Errorf("%v", errs)
	}

	moduleVersions := map[string]*semver.Version{}
	var entries []*yang.Entry
	for moduleName, entry := range moduleEntryMap {
		entries = append(entries, flattenedEntriesAux(entry)...)
		if version, err := getOpenConfigModuleVersion(entry); err == nil {
			moduleVersions[moduleName] = version
		}
	}

	entryMap := map[string]*yang.Entry{}
	for _, entry := range entries {
		entryMap[entry.Path()] = entry
	}
	return entryMap, moduleVersions, nil
}

func flattenedEntriesAux(entry *yang.Entry) []*yang.Entry {
	entries := []*yang.Entry{entry}
	for _, entry := range entry.Dir {
		entries = append(entries, flattenedEntriesAux(entry)...)
	}
	return entries
}

func diffMaps(oldEntries, newEntries map[string]*yang.Entry, oldModuleVersions, newModuleVersions map[string]*semver.Version) *DiffReport {
	report := &DiffReport{
		oldModuleVersions: oldModuleVersions,
		newModuleVersions: newModuleVersions,
	}
	for path, oldEntry := range oldEntries {
		report.addPair(oldEntry, newEntries[path])
	}
	for path, newEntry := range newEntries {
		if oldEntries[path] == nil {
			report.addPair(oldEntries[path], newEntry)
		}
	}
	return report
}
