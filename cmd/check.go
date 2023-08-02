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

	"github.com/openconfig/models-ci/ocdiff"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// checkCmd represents the check command
// FIXME(wenbli): Update comments.
var checkCmd = &cobra.Command{
	Use:   "diff",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		viper.BindPFlags(cmd.Flags())
		report, err := ocdiff.ReportDiff(viper.GetStringSlice("oldp"), viper.GetStringSlice("newp"), viper.GetStringSlice("oldfiles"), viper.GetStringSlice("newfiles"))
		if err != nil {
			return err
		}

		if viper.GetBool("disallowed-incompats") {
			if out := report.ReportDisallowedIncompats(); out != "" {
				fmt.Printf("Backward-incompatible changes not covered by version increments per semver.org:\n%s", out)
				os.Exit(1)
			}
		} else {
			fmt.Printf(report.ReportAll())
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Flags().StringSlice("oldp", []string{}, "search path for old set of YANG files")
	checkCmd.Flags().StringSlice("newp", []string{}, "search path for new set of YANG files")
	checkCmd.Flags().StringSlice("oldfiles", []string{}, "comma-separated list of old YANG files")
	checkCmd.Flags().StringSlice("newfiles", []string{}, "comma-separated list of new YANG files")
	checkCmd.Flags().Bool("disallowed-incompats", false, "only show disallowed (per semver.org) backwards-incompatible changes. Note that the backwards-incompatible checks are not exhausive.")
}
