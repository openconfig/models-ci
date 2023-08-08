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

// yangutil contains utilities related to YANG.
package yangutil

import (
	"os"
	"path/filepath"
)

func GetAllYANGFiles(path string) ([]string, error) {
	var files []string
	if err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(info.Name()) == ".yang" {
				files = append(files, path)
			}
			return nil
		},
	); err != nil {
		return nil, err
	}
	return files, nil
}
