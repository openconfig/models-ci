// Binary docgen generates a script that can be used to generate openconfig
// model documentation in pyang, along with a sitemap JSON document that can
// be used to dynamically build an index of the generated models.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	log "github.com/golang/glog"

	"github.com/openconfig/models-ci/commonci"
)

var (
	ocPath     = flag.String("repo_path", "", "Path to OpenConfig repo.")
	ocpyangDir = flag.String("ocpyang_path", "", "Path to the oc-pyang repo.")
	outputDir  = flag.String("output_path", "", "Output directory.")
	outputFile = flag.String("output_file", "", "File to write docgen script to.")
	outputMap  = flag.String("output_map", "", "File to write the directory map to.")
)

const (
	docTemplate string = `
pyang --plugindir={{ .PluginDir }}/openconfig_pyang/plugins/ \
  -p {{ .RepoDir }} \
  --doc-format=html \
  -o {{ .OutputFile }} \
  -f docs \
  {{- $dir := .RepoDir -}}
  {{- range $i, $file := .Files }}
  {{- if $i }} \{{ end }}
  {{ $dir }}/{{ $file }}
  {{- end }}
pyang --plugindir={{ .PluginDir }}/openconfig_pyang/plugins/ \
  -p {{ .RepoDir }} \
  -o {{ .OutputTree }} \
  -f oc-jstree \
  --oc-jstree-strip \
  {{- $dir := .RepoDir -}}
  {{- range $i, $file := .Files }}
  {{- if $i }} \{{ end }}
  {{ $dir }}/{{ $file }}
  {{- end }}`
)

var (
	// docsTmpl is the generated template defined based on the docTemplate
	// constant.
	docsTmpl = template.Must(template.New("docs").Parse(docTemplate))
)

func main() {
	flag.Parse()

	if *ocPath == "" {
		log.Exitf("unspecified path to openconfig/public repo.")
	}

	if *ocpyangDir == "" {
		log.Exitf("unspecified path to openconfig/oc-pyang repo.")
	}

	if *outputDir == "" {
		log.Exitf("unspecified output directory.")
	}

	if *outputFile == "" {
		log.Exitf("unspecified output file.")
	}

	rules, err := commonci.ParseOCModels(*ocPath)
	if err != nil {
		log.Exitf("cannot parse models, err: %v", err)
	}

	docScript, index, err := generateScript(rules, *outputDir, *ocPath, *ocpyangDir)
	if err != nil {
		log.Exitf("cannot generate index, err: %v", err)
	}

	js, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		log.Exitf("cannot write index file, %v", err)
	}
	if err := os.WriteFile(*outputMap, js, 0644); err != nil {
		log.Exitf("cannot write site map file, %v", err)
	}

	if err := os.WriteFile(*outputFile, docScript.Bytes(), 0755); err != nil {
		log.Exitf("cannot write output file, %v", err)
	}
}

// indexMap is a map keyed by the name of a build rule with a value which maps
// a label to a file path for the generated document.
//
// For example, if the script is called with a rule map that contains only a build
// rule called "openconfig-foo" that generates a "doc" and "tree"
// file in the build script, the indexMap is:
//
//		{
//			"openconfig-foo": {
//				"docs": "files/foo-docs.html",
//				"tree": "files/foo-tree.html",
//			}
//		}
type indexMap map[string]map[string]string

// generateScript takes an input OpenConfigModelMap (extracted from build rules) and returns
// a buffer containing the script to generate documentation for those models, and the index
// as described above. If errors are encountered an error is returned.
//
// The script contained in the buffer uses the constant template declared above.
func generateScript(rules commonci.OpenConfigModelMap, outdir, repodir, plugindir string) (*bytes.Buffer, indexMap, error) {
	docScript := &bytes.Buffer{}
	docScript.WriteString("#!/bin/bash")

	index := indexMap{}
	for dir, r := range rules.ModelInfoMap {
		for _, s := range r {
			if len(s.DocFiles) == 0 {
				continue
			}

			opath := fmt.Sprintf("%s/%s.html", outdir, s.Name)
			treepath := fmt.Sprintf("%s/%s-tree.html", outdir, s.Name)
			index[s.Name] = map[string]string{
				"docs": opath,
				"tree": treepath,
			}

			rwFiles := []string{}
			for _, f := range s.DocFiles {
				rwFiles = append(rwFiles, strings.Replace(f, "yang/", "release/models/", 1))
			}
			sort.Strings(rwFiles)

			tmp := struct {
				RepoDir    string
				PluginDir  string
				OutputFile string
				OutputTree string
				Files      []string
			}{
				RepoDir:    repodir,
				PluginDir:  plugindir,
				OutputFile: opath,
				OutputTree: treepath,
				Files:      rwFiles,
			}

			if err := docsTmpl.Execute(docScript, tmp); err != nil {
				return nil, nil, fmt.Errorf("cannot write docs entry for rule %s->%s, err: %v", dir, s.Name, err)
			}
		}
	}

	return docScript, index, nil
}
