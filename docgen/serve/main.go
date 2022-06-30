// Binary serve creates a simple http server that serves:
//  - a dynamically generated index written based on an input sitemap.json file (embedded using embed.FS)
//	- a static set of files that are contained in the following subdirectories.
//		- js
//		- css
//		- static
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"

	log "github.com/golang/glog"
)

//go:embed "sitemap.json"
var fs embed.FS

func main() {
	flag.Parse()
	sitemap := map[string]map[string]string{}

	f, err := fs.ReadFile("sitemap.json")
	if err != nil {
		log.Exitf("cannot read sitemap, %v", err)
	}
	if err := json.Unmarshal(f, &sitemap); err != nil {
		log.Exitf("cannot unmarshal sitemap, %v", err)
	}

	http.Handle("/js/", http.StripPrefix("/static/", http.FileServer(http.Dir("./js"))))
	http.Handle("/css/", http.StripPrefix("/static/", http.FileServer(http.Dir("./css"))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", mkIndex(sitemap))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Infof("defaulted to port %s", port)
	}

	log.Infof("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("cannot listen, err: %v", err)
	}
}

// mkIndex generates the index HTML page.
func mkIndex(sitemap map[string]map[string]string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "<html><head>")
		fmt.Fprintf(w, "<title>OpenConfig Model Documentation</title>")
		fmt.Fprintf(w, `<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">`)
		fmt.Fprintf(w, "</head><body>")
		fmt.Fprintf(w, "<h1>OpenConfig Model Documentation</h1><ul>")

		keys := []string{}
		for k := range sitemap {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			labels := []string{}
			for l := range sitemap[k] {
				labels = append(labels, l)
			}
			sort.Strings(labels)
			fmt.Fprintf(w, "<li><strong>%s</strong><ul>", k)
			for _, l := range labels {
				fmt.Fprintf(w, `<li><a href="%s">%s</a></li>`, sitemap[k][l], l)
			}
			fmt.Fprintf(w, "</ul></li>")
		}
		fmt.Fprintf(w, "</ul></body></html>")
	}
}
