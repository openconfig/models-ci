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

// ocdiff checks for backwards-compatibility between two sets of YANG
// files.
package ocdiff

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/goyang/pkg/yangentry"
	"golang.org/x/exp/slices"
)

// ReportDiff returns a diff report given options for compiling two sets of
// YANG files.
func ReportDiff(oldpaths, newpaths, oldfiles, newfiles []string) (*DiffReport, error) {
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

type YANGNode struct {
	path          string
	schema        *yang.Entry
	allowIncompat bool
}

type YANGNodeUpdate struct {
	path             string
	oldSchema        *yang.Entry
	newSchema        *yang.Entry
	allowIncompat    bool
	incompatComments []string
	oldVersion       *semver.Version
	NewVersion       *semver.Version
}

type DiffReport struct {
	NewNodes          []*YANGNode
	UpdatedNodes      []*YANGNodeUpdate
	DeletedNodes      []*YANGNode
	OldModuleVersions map[string]*semver.Version
	NewModuleVersions map[string]*semver.Version
}

// ReportDisallowedIncompats reports any backward-incompatible changes disallowed by version increments.
func (r *DiffReport) ReportDisallowedIncompats() string {
	r.Sort()
	var b strings.Builder
	for _, del := range r.DeletedNodes {
		if !del.allowIncompat && (del.schema.IsLeaf() || del.schema.IsLeafList()) {
			b.WriteString(fmt.Sprintf("leaf deleted: %s\n", del.schema.Path()))
		}
	}
	for _, upd := range r.UpdatedNodes {
		if !upd.allowIncompat && len(upd.incompatComments) > 0 {
			b.WriteString(fmt.Sprintf("node updated %s: %s\n", upd.oldSchema.Path(), strings.Join(upd.incompatComments, "\n\t")))
		}
	}
	return b.String()
}

// ReportAll reports all YANG changes.
func (r *DiffReport) ReportAll() string {
	r.Sort()
	var b strings.Builder
	for _, del := range r.DeletedNodes {
		if del.schema.IsLeaf() || del.schema.IsLeafList() {
			b.WriteString(fmt.Sprintf("leaf deleted: %s\n", del.schema.Path()))
		}
	}
	for _, upd := range r.UpdatedNodes {
		if len(upd.incompatComments) > 0 {
			b.WriteString(fmt.Sprintf("node updated %s: %s\n", upd.oldSchema.Path(), strings.Join(upd.incompatComments, "\n\t")))
		} else {
			b.WriteString(fmt.Sprintf("node updated %s\n", upd.oldSchema.Path()))
		}
	}
	for _, added := range r.NewNodes {
		if added.schema.IsLeaf() || added.schema.IsLeafList() {
			b.WriteString(fmt.Sprintf("leaf added: %s\n", added.schema.Path()))
		}
	}
	return b.String()
}

func (r *DiffReport) Sort() {
	slices.SortFunc(r.NewNodes, func(a, b *YANGNode) int { return strings.Compare(a.path, b.path) })
	slices.SortFunc(r.DeletedNodes, func(a, b *YANGNode) int { return strings.Compare(a.path, b.path) })
	slices.SortFunc(r.UpdatedNodes, func(a, b *YANGNodeUpdate) int { return strings.Compare(a.path, b.path) })
}

func getKind(e *yang.Entry) string {
	if e.Type != nil {
		return fmt.Sprint(e.Type.Kind)
	} else {
		return fmt.Sprint(e.Kind)
	}
}

func (r *DiffReport) entryAllowsIncompat(e *yang.Entry) bool {
	moduleName := belongingModule(yang.RootNode(e.Node))
	oldVersion, newVersion := r.OldModuleVersions[moduleName], r.NewModuleVersions[moduleName]
	if oldVersion == nil || newVersion == nil {
		// This happens if the openconfig-version is not found (e.g. in IETF modules).
		return true
	}
	return oldVersion.Major() == 0 || oldVersion.Major() > newVersion.Major()
}

func (r *DiffReport) addPair(o *yang.Entry, n *yang.Entry) error {
	switch {
	case o == nil && n == nil:
	case o == nil:
		r.NewNodes = append(r.NewNodes, &YANGNode{
			schema: n,
			path:   n.Path(),
		})
	case n == nil:
		r.DeletedNodes = append(r.DeletedNodes, &YANGNode{
			schema:        o,
			path:          o.Path(),
			allowIncompat: r.entryAllowsIncompat(o),
		})
	default:
		upd := &YANGNodeUpdate{
			oldSchema:     o,
			newSchema:     n,
			path:          o.Path(),
			allowIncompat: r.entryAllowsIncompat(o),
		}
		updated := false
		if oldKind, newKind := getKind(o), getKind(n); oldKind != newKind {
			upd.incompatComments = append(upd.incompatComments, fmt.Sprintf("type changed from %s to %s", oldKind, newKind))
			updated = true
		}
		if updated {
			r.UpdatedNodes = append(r.UpdatedNodes, upd)
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
		OldModuleVersions: oldModuleVersions,
		NewModuleVersions: newModuleVersions,
	}
	for path, oldEntry := range oldEntries {
		report.addPair(oldEntry, newEntries[path])
	}
	for path, newEntry := range newEntries {
		report.addPair(oldEntries[path], newEntry)
	}
	return report
}
