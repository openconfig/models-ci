package octest

import (
	"flag"
	"fmt"
	"regexp"
	"testing"

	"gooctest"
)

var ocdir string

type YANGLeaf struct {
	module string
	name   string
}

type RegexpTest struct {
	inData    string
	wantMatch bool
}

func TestRegexps(t *testing.T) {
	tests := []struct {
		name     string
		modules  []string
		leaf     YANGLeaf
		testData []RegexpTest
	}{{
		name:    "simple ipv4 address",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "ipv4-address"},
		testData: []RegexpTest{
			RegexpTest{`1.1.1.1`, true},
			RegexpTest{`1.1.1.256`, false},
			RegexpTest{`256.1.1.1%eth0`, false},
		},
	}, {
		name:     "failing ipv4 address",
		modules:  []string{"testdata/test.yang"},
		leaf:     YANGLeaf{"regexp-test", "ipv4-address"},
		testData: []RegexpTest{RegexpTest{"invalid-data", false}},
	}, {
		name:    "union ip address",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "ip-address"},
		testData: []RegexpTest{
			RegexpTest{`255.255.255.255`, true},
			RegexpTest{`2001:db8::1`, true},
			RegexpTest{"invalid-data", false},
			RegexpTest{`::FFFF:192.0.2.1`, true},
			RegexpTest{`::1`, true},
		},
	}, {
		name:    "bgp-standard-community",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "bgp-std-community"},
		testData: []RegexpTest{
			RegexpTest{`15169:42`, true},
			RegexpTest{`6643:21438`, true},
			RegexpTest{`29636:4444`, true},
			RegexpTest{`65535:65535`, true},
			RegexpTest{`0:0`, true},
		},
	}}

	for _, tt := range tests {
		yangE, errs := gooctest.ProcessModules(tt.modules, []string{ocdir})
		if len(errs) != 0 {
			t.Fatalf("%s: could not parse modules: %v", tt.name, errs)
		}

		mod, modok := yangE[tt.leaf.module]
		if !modok {
			t.Fatalf("%s: could not find expected module: %s (%v)", tt.name, tt.leaf.module, yangE)
		}

		leaf, leafok := mod.Dir[tt.leaf.name]
		if !leafok {
			t.Fatalf("%s: could not find expected leaf: %s", tt.name, tt.leaf.name)
		}

		if len(leaf.Errors) != 0 {
			t.Errorf("%s: leaf had associated errors: %v", tt.name, leaf.Errors)
			continue
		}

		for _, tc := range tt.testData {
			var gotMatch bool
			if len(leaf.Type.Type) == 0 {
				gotMatch = true
				for _, pattern := range leaf.Type.Pattern {
					if r, err := regexp.CompilePOSIX(fmt.Sprintf("^%s$", pattern)); err != nil {
						t.Fatalf("%s: cannot compile regexp %s for leaf", tt.name, pattern)
					} else {
						if r.MatchString(tc.inData) == false {
							gotMatch = false
						}
					}
				}
			} else {
				// Handle unions
				results := make([]bool, 0)
				for _, membertype := range leaf.Type.Type {
					matchedAllForType := true
					for _, pattern := range membertype.Pattern {
						if r, err := regexp.CompilePOSIX(fmt.Sprintf("^%s$", pattern)); err != nil {
							t.Fatalf("%s: cannot compile regexp %s for leaf", tt.name, pattern)
						} else {
							if r.MatchString(tc.inData) == false {
								matchedAllForType = false
							}
						}
					}
					results = append(results, matchedAllForType)
				}

				gotMatch = false
				for _, r := range results {
					if r == true {
						gotMatch = true
					}
				}
			}

			if gotMatch != tc.wantMatch {
				t.Errorf("%s: string %s did not have expected result: %v", tt.name, tc.inData, tc.wantMatch)
			}
		}
	}
}

// init sets up the test, particularly parsing the OpenConfig path which is
// supplied as a command line argument.
func init() {
	flag.StringVar(&ocdir, "ocdir", "../..", "Path to OpenConfig models repo")
	flag.Parse()
}
