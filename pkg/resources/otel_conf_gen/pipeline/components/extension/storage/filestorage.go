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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	defaultFileStorageDirectory = "/var/lib/otelcol/file_storage"
)

var defaultFileStorageDirectoryWindows = filepath.Join(os.Getenv("ProgramData"), "Otelcol", "FileStorage")

func GenerateFileStorageExtensionForTenant(persistDirPath string, tenantName string) map[string]any {
	return map[string]any{
		"create_directory": true,
		"directory":        DetermineFileStorageDirectory(persistDirPath, tenantName),
	}
}

func DetermineFileStorageDirectory(persistDirPath string, tenantName string) string {
	if persistDirPath != "" {
		return persistDirPath
	}

	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("%s/%s", defaultFileStorageDirectoryWindows, tenantName)
	default:
		return fmt.Sprintf("%s/%s", defaultFileStorageDirectory, tenantName)
	}
}
