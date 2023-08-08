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

package cmd

import (
	"fmt"
	"os"

	"github.com/openconfig/models-ci/openconfig-ci/ocdiff"
	"github.com/openconfig/models-ci/yangutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// diffCmd represents the diff command, which diffs two sets of OpenConfig YANG
// files.
var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Diff between two sets of OpenConfig YANG files",
	Long: `Use this command to find what's different between two commits of openconfig/public:

openconfig-ci diff --oldp public_old/third_party --newp public_new/third_party --oldroot public_old/release --newroot public_new/release
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		viper.BindPFlags(cmd.Flags())
		oldfiles, err := yangutil.GetAllYANGFiles(viper.GetString("oldroot"))
		if err != nil {
			return fmt.Errorf("error while finding YANG files from the old root: %v", err)
		}
		newfiles, err := yangutil.GetAllYANGFiles(viper.GetString("newroot"))
		if err != nil {
			return fmt.Errorf("error while finding YANG files from the new root: %v", err)
		}
		report, err := ocdiff.NewDiffReport(viper.GetStringSlice("oldp"), viper.GetStringSlice("newp"), oldfiles, newfiles)
		if err != nil {
			return err
		}

		var opts []ocdiff.Option
		if viper.GetBool("github-comment") {
			opts = append(opts, ocdiff.WithGithubCommentStyle())
		}

		if viper.GetBool("disallowed-incompats") {
			opts = append(opts, ocdiff.WithDisallowedIncompatsOnly())
			if out := report.Report(opts...); out != "" {
				fmt.Printf("-----------Breaking changes that need a major version increment (note that this check is not exhaustive)-----------\n%s", out)
				os.Exit(1)
			}
		} else {
			fmt.Printf(report.Report(opts...))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)

	diffCmd.Flags().StringSlice("oldp", []string{}, "search path for old set of YANG files")
	diffCmd.Flags().StringSlice("newp", []string{}, "search path for new set of YANG files")
	diffCmd.Flags().StringP("oldroot", "o", "", "Root directory of old OpenConfig YANG files")
	diffCmd.Flags().StringP("newroot", "n", "", "Root directory of new OpenConfig YANG files")
	diffCmd.Flags().Bool("disallowed-incompats", false, "only show disallowed (per semver.org) backward-incompatible changes. Note that the backward-incompatible checks are not exhausive.")
	diffCmd.Flags().Bool("github-comment", false, "Show output suitable for posting in a GitHub comment.")
}
