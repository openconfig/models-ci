package main

import "testing"

func TestHasBreaking(t *testing.T) {
	tests := []struct {
		desc         string
		inVersions   versionRecordSlice
		wantBreaking bool
	}{{
		desc: "deleted",
		inVersions: versionRecordSlice{{
			File:            "openconfig-deleted.yang",
			OldMajorVersion: 1,
			NewMajorVersion: 0,
			OldVersion:      "1.0.0",
			NewVersion:      "",
		}},
		wantBreaking: true,
	}, {
		desc: "minor",
		inVersions: versionRecordSlice{{
			File:            "openconfig-acl-submodule.yang",
			OldMajorVersion: 1,
			NewMajorVersion: 1,
			OldVersion:      "1.1.3",
			NewVersion:      "1.2.3",
		}},
		wantBreaking: false,
	}, {
		desc: "patch",
		inVersions: versionRecordSlice{{
			File:            "openconfig-acl.yang",
			OldMajorVersion: 1,
			NewMajorVersion: 1,
			OldVersion:      "1.2.2",
			NewVersion:      "1.2.3",
		}},
		wantBreaking: false,
	}, {
		desc: "one",
		inVersions: versionRecordSlice{{
			File:            "openconfig-interface-submodule.yang",
			OldMajorVersion: 0,
			NewMajorVersion: 1,
			OldVersion:      "0.5.0",
			NewVersion:      "1.0.0",
		}},
		wantBreaking: false,
	}, {
		desc: "major",
		inVersions: versionRecordSlice{{
			File:            "openconfig-interface.yang",
			OldMajorVersion: 1,
			NewMajorVersion: 2,
			OldVersion:      "1.1.3",
			NewVersion:      "2.0.0",
		}},
		wantBreaking: true,
	}, {
		desc: "minor",
		inVersions: versionRecordSlice{{
			File:            "openconfig-packet-match.yang",
			OldMajorVersion: 1,
			NewMajorVersion: 1,
			OldVersion:      "1.1.2",
			NewVersion:      "1.2.0",
		}},
		wantBreaking: false,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := tt.inVersions.hasBreaking(), tt.wantBreaking; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestMajorVersionChanges(t *testing.T) {
	prevCommitSHA := commitSHA
	commitSHA = "a0"
	defer func() {
		commitSHA = prevCommitSHA
	}()
	tests := []struct {
		desc                    string
		inVersions              versionRecordSlice
		wantMajorVersionChanges string
	}{{
		desc: "deleted",
		inVersions: versionRecordSlice{{
			File:            "openconfig-deleted.yang",
			OldMajorVersion: 1,
			NewMajorVersion: 0,
			OldVersion:      "1.0.0",
			NewVersion:      "",
		}},
		wantMajorVersionChanges: "Major YANG version changes in commit a0:\nopenconfig-deleted.yang: `1.0.0` -> ``\n",
	}, {
		desc: "minor",
		inVersions: versionRecordSlice{{
			File:            "openconfig-acl-submodule.yang",
			OldMajorVersion: 1,
			NewMajorVersion: 1,
			OldVersion:      "1.1.3",
			NewVersion:      "1.2.3",
		}},
		wantMajorVersionChanges: `No major YANG version changes in commit a0`,
	}, {
		desc: "patch",
		inVersions: versionRecordSlice{{
			File:            "openconfig-acl.yang",
			OldMajorVersion: 1,
			NewMajorVersion: 1,
			OldVersion:      "1.2.2",
			NewVersion:      "1.2.3",
		}},
		wantMajorVersionChanges: `No major YANG version changes in commit a0`,
	}, {
		desc: "one",
		inVersions: versionRecordSlice{{
			File:            "openconfig-interface-submodule.yang",
			OldMajorVersion: 0,
			NewMajorVersion: 1,
			OldVersion:      "0.5.0",
			NewVersion:      "1.0.0",
		}},
		wantMajorVersionChanges: "Major YANG version changes in commit a0:\nopenconfig-interface-submodule.yang: `0.5.0` -> `1.0.0`\n",
	}, {
		desc: "major",
		inVersions: versionRecordSlice{{
			File:            "openconfig-interface.yang",
			OldMajorVersion: 1,
			NewMajorVersion: 2,
			OldVersion:      "1.1.3",
			NewVersion:      "2.0.0",
		}},
		wantMajorVersionChanges: "Major YANG version changes in commit a0:\nopenconfig-interface.yang: `1.1.3` -> `2.0.0`\n",
	}, {
		desc: "minor",
		inVersions: versionRecordSlice{{
			File:            "openconfig-packet-match.yang",
			OldMajorVersion: 1,
			NewMajorVersion: 1,
			OldVersion:      "1.1.2",
			NewVersion:      "1.2.0",
		}},
		wantMajorVersionChanges: `No major YANG version changes in commit a0`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := tt.inVersions.MajorVersionChanges(), tt.wantMajorVersionChanges; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}
