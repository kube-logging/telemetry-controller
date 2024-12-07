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
	"strings"
)

const (
	defaultFileStorageDirectory = "/var/lib/otelcol/file_storage"
)

var (
	DefaultFileStorageName             = mustNewFileStorageExtensionName("file_storage/default")
	defaultFileStorageDirectoryWindows = filepath.Join(os.Getenv("ProgramData"), "Otelcol", "FileStorage")
)

type FileStorageExtensionName struct {
	typ  string
	name string
}

func (f FileStorageExtensionName) Type() string {
	return f.typ
}

func (f FileStorageExtensionName) Name() string {
	return f.name
}

func (f FileStorageExtensionName) String() string {
	return fmt.Sprintf("%s/%s", f.typ, f.name)
}

func NewFileStorageExtensionName(fullName string) (FileStorageExtensionName, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return FileStorageExtensionName{}, fmt.Errorf("invalid file storage extension name format: %s", fullName)
	}
	return FileStorageExtensionName{
		typ:  parts[0],
		name: parts[1],
	}, nil
}

func mustNewFileStorageExtensionName(s string) FileStorageExtensionName {
	ext, err := NewFileStorageExtensionName(s)
	if err != nil {
		panic(fmt.Sprintf("Invalid file storage extension name: %v", err))
	}
	return ext
}

func GenerateFileStorageExtension(persistDirPath string) map[string]any {
	return map[string]any{
		"create_directory": true,
		"directory":        determineFileStorageDirectory(persistDirPath),
	}
}

func determineFileStorageDirectory(persistDirPath string) string {
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
