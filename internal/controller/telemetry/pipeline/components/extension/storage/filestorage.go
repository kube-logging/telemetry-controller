// Copyright © 2024 Kube logging authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	defaultFileStorageDirectory = "/var/lib/otelcol/file_storage"
)

var (
	defaultFileStorageDirectoryWindows = filepath.Join(os.Getenv("ProgramData"), "Otelcol", "FileStorage")
)

func GenerateFileStorageExtensionForTenant(persistDirPath string) map[string]any {
	return map[string]any{
		"create_directory": true,
		"directory":        DetermineFileStorageDirectory(persistDirPath),
	}
}

func DetermineFileStorageDirectory(persistDirPath string) string {
	if persistDirPath != "" {
		info, err := os.Stat(persistDirPath)
		if err == nil && info.IsDir() {
			return persistDirPath
		} else {
			return defaultFileStorageDirectory
		}
	}

	if runtime.GOOS == "windows" {
		return defaultFileStorageDirectoryWindows
	}

	return defaultFileStorageDirectory
}
