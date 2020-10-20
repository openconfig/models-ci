package regexp_test

import (
	"log"
	"os"
	"regexp"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"

	"gooctest"
)

var ocdir string

// YANGLeaf is a structure sued to describe a particular leaf of YANG schema.
type YANGLeaf struct {
	module string
	name   string
}

// RegexpTest specifies a test case for a particular regular expression check.
type RegexpTest struct {
	inData    string
	wantMatch bool
}

// TestRegexps tests mock input data against a set of leaves that have patterns
// specified for them. It ensures that the regexp compiles as a POSIX regular
// expression according to the OpenConfig style guide.
func TestRegexps(t *testing.T) {
	tests := []struct {
		name     string
		modules  []string
		leaf     YANGLeaf
		testData []RegexpTest
	}{{
		name:    "ipv4 address",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "ipv4-address"},
		testData: []RegexpTest{
			RegexpTest{`1.1.1.1`, true},
			RegexpTest{`1.1.1.256`, false},
			RegexpTest{`256.1.1.1%eth0`, false},
		},
	}, {
		name:    "union ip address",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "ip-address"},
		testData: []RegexpTest{
			RegexpTest{`255.255.255.255`, true},
			RegexpTest{`2001:db8::1`, true},
			RegexpTest{"invalid-data", false},
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
			RegexpTest{`65536:1`, false},
			RegexpTest{`1:65536`, false},
			RegexpTest{`425353:comm`, false},
		},
	}, {
		name:    "ipv4-prefix",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "ipv4-prefix"},
		testData: []RegexpTest{
			RegexpTest{`0.0.0.0/0`, true},
			RegexpTest{`255.255.255.255/32`, true},
			RegexpTest{`256.0.0.0/31`, false},
			RegexpTest{`1.2.3.0/24`, true},
			RegexpTest{`1.2.3.4/33`, false},
		},
	}, {
		name:    "ipv6-prefix",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "ipv6-prefix"},
		testData: []RegexpTest{
			RegexpTest{`::/0`, true},
			RegexpTest{`FFFF:FFFF:FFFF:FFFF:FFFF:FFFF:FFFF:FFFF/128`, true},
			RegexpTest{`FFFF:FFFF:FFFF:NOTVALID:FFFF:FFFF:FFFF:FFFF/64`, false},
			RegexpTest{`2001:DB8::/32`, true},
			RegexpTest{`2001:4C20::/129`, false},
		},
	}, {
		name:    "ip-prefix",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "ip-prefix"},
		testData: []RegexpTest{
			RegexpTest{"0.0.0.0/0", true},
			RegexpTest{"192.0.2.1/32", true},
			RegexpTest{"192.0.2.2/33", false},
			RegexpTest{"FE80::CAFE/128", true},
			RegexpTest{"FE81::CAFE:DEAD:BEEF/129", false},
		},
	}, {
		name:    "ipv6-address",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "ipv6-address"},
		testData: []RegexpTest{
			RegexpTest{"2620::1000:3202:23e:e1ff:fec7:7112", true},
			RegexpTest{"fe80::23e:e1ff:fec7:7112", true},
			RegexpTest{"fe80::23e:NOTVALID:fec7:7112", false},
			RegexpTest{"FFFF::NOTE::FFFF", false},
			RegexpTest{"FFFF::1::42", false},
		},
	}, {
		name:    "bgp-extended-community",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "bgp-ext-community"},
		testData: []RegexpTest{
			// Type 1 extended communities (2b AS: 4b integer)
			RegexpTest{`29636:10`, true},
			RegexpTest{`5413:4294967295`, true},
			RegexpTest{`4445:0`, true},
			RegexpTest{`1273:4294967296`, false},
			RegexpTest{`2856:400`, true},
			RegexpTest{`5400:invalid`, false},
			RegexpTest{`i6643:10`, false},
			RegexpTest{`15169:22432`, true},
			// Type 2 extended communities: (4b IP: 2b integer)
			RegexpTest{`1.1.1.1:4294967296`, false},
			RegexpTest{`1.2.3.4.5:10`, false},
			RegexpTest{`82.42.12.35:65535`, true},
			RegexpTest{`82.42.12.35:66536`, false},
			RegexpTest{`254.254.256.254:10`, false},
			RegexpTest{`0.0.0.0:200`, true},
			RegexpTest{`leading192.0.2.1:65535`, false},
			// 4b AS : 2b integer
			RegexpTest{`4294967296:65535`, false},
			RegexpTest{`4294967295:65535`, true},
			RegexpTest{`0:65535`, true},
			RegexpTest{`4294967295:0`, true},
			RegexpTest{`4294967296:0`, false},
			// Route Target Type 1 - route-target:<2b AS>:<4b local>
			RegexpTest{`route-target:64`, false},
			RegexpTest{`route-target:65535:10`, true},
			RegexpTest{`route-TARGET:65535:10`, false},
			RegexpTest{`route-target:15169:4294967296`, false},
			RegexpTest{`route-target:15169:4294967295`, true},
			// Route Target Type 2 - route-target:<ipv4>:<2b local>
			RegexpTest{`route-target:256.0.2.36:10`, false},
			RegexpTest{`route-target:192.0.2.1:10`, true},
			RegexpTest{`route-target:192.0.2.1:65536`, false},
			// Route Target w/ 4B AS:<2b local>
			RegexpTest{`route-target:4294967295:10`, true},
			RegexpTest{`route-target:4294967296:10`, false},
			RegexpTest{`route-target:5413:65535`, true},
			// Route Origin Type 1 - route-target:<2b AS>:<4b local>
			RegexpTest{`route-origin:53`, false},
			RegexpTest{`route-origin:65535:10`, true},
			RegexpTest{`route-ORIGINTRAIL:65535:10`, false},
			RegexpTest{`route-origin:15169:4294967296`, false},
			RegexpTest{`route-origin:15169:4294967295`, true},
			// Route Origin Type 2 - route-target:<ipv4>:<2b local>
			RegexpTest{`route-origin:512.0.2.36:10`, false},
			RegexpTest{`route-origin:10.18.253.24:10`, true},
			RegexpTest{`route-origin:192.168.1.1:65536`, false},
			// Route Origin w/ 4B AS:<2b local>
			RegexpTest{`route-origin:4294967295:5353`, true},
			RegexpTest{`route-origin:4294967296:9009`, false},
			RegexpTest{`route-origin:5413:65535`, true},
		},
	}, {
		name:    "domain-name",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "domain-name"},
		testData: []RegexpTest{
			RegexpTest{"ambroseesorbma", true},
			RegexpTest{"ambrose esorbma", false},
			RegexpTest{"claire.", true},
			RegexpTest{"~~~-+=.", false},
			RegexpTest{"012345678901234567890123456789012345678901234567890123456789012.", true},
			RegexpTest{"0123456789012345678901234567890123456789012345678901234567890123", false},
		},
	}, {
		name:    "port-num-range",
		modules: []string{"testdata/test.yang"},
		leaf:    YANGLeaf{"regexp-test", "port-num-range"},
		testData: []RegexpTest{
			RegexpTest{"0..0", true},
			RegexpTest{"00000..00000", true},
			RegexpTest{"00..00", true},
			RegexpTest{"01..01", true},
			RegexpTest{"0..65535", true},
			RegexpTest{"01111..09999", true},
			RegexpTest{"9999..59999", true},
			RegexpTest{"60536..60999", true},
			RegexpTest{"60999..61999", true},
			RegexpTest{"62999..63999", true},
			RegexpTest{"64999..65535", true},
			RegexpTest{"65535..66646", false},
			RegexpTest{"65535..65536", false},
			RegexpTest{"65535..65545", false},
			RegexpTest{"65535..65635", false},
			RegexpTest{"66535..65535", false},
			RegexpTest{"66999..67890", false},
			RegexpTest{"70000..70000", false},
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
				_, gotMatch = checkPattern(tc.inData, leaf.Type.POSIXPattern)
			} else {
				// Handle unions
				results := make([]bool, 0)
				for _, membertype := range leaf.Type.Type {
					// Only do the test when there is a pattern specified against the
					// type as it may not be a string.
					if membertype.Kind != yang.Ystring || len(membertype.POSIXPattern) == 0 {
						continue
					}
					matchedAllForType := true
					_, matchedAllForType = checkPattern(tc.inData, membertype.POSIXPattern)
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
				t.Errorf("%s: string %s did not have expected result: %v",
					tt.name, tc.inData, tc.wantMatch)
			}
		}
	}
}

// checkPattern builds and compils
func checkPattern(testData string, patterns []string) (compileErr error, matched bool) {
	for _, pattern := range patterns {
		if r, err := regexp.CompilePOSIX(pattern); err != nil {
			return
		} else {
			matched = r.MatchString(testData)
		}
	}
	return
}

// init sets up the test, particularly parsing the OpenConfig path which is
// supplied as a command line argument.
func init() {
	ocdir = os.Getenv("OCDIR")
	if ocdir == "" {
		log.Fatal("missing environment variable $OCDIR for specifying model root directory")
	}
}
