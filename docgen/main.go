package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
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
  {{- $dir := .RepoDir -}}
  {{- range $i, $file := .Files }}
  {{- if $i }} \{{ end }}
  {{ $dir }}/{{ $file }}
  {{- end }}`
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

	docsTmpl := template.Must(template.New("docs").Parse(docTemplate))

	rules, err := commonci.ParseOCModels(*ocPath)
	if err != nil {
		log.Exitf("cannot parse models, err: %v", err)
	}

	docScript := &bytes.Buffer{}
	docScript.WriteString("#!/bin/bash")

	index := map[string]map[string]string{}

	for dir, r := range rules.ModelInfoMap {
		for _, s := range r {
			if len(s.DocFiles) == 0 {
				continue
			}

			opath := fmt.Sprintf("%s/%s.html", *outputDir, s.Name)
			treepath := fmt.Sprintf("%s/%s-tree.html", *outputDir, s.Name)
			index[s.Name] = map[string]string{
				"docs": opath,
				"tree": treepath,
			}

			rwFiles := []string{}
			for _, f := range s.DocFiles {
				rwFiles = append(rwFiles, strings.Replace(f, "yang/", "release/models/", 1))
			}

			tmp := struct {
				RepoDir    string
				PluginDir  string
				OutputFile string
				OutputTree string
				Files      []string
			}{
				RepoDir:    *ocPath,
				PluginDir:  *ocpyangDir,
				OutputFile: opath,
				OutputTree: treepath,
				Files:      rwFiles,
			}

			if err := docsTmpl.Execute(docScript, tmp); err != nil {
				log.Exitf("cannot write docs entry for rule %s->%s, err: %v", dir, s.Name, err)
			}
		}
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
