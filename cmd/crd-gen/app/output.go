// Copyright 2022 jim.zoumo@gmail.com
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

package app

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"sigs.k8s.io/controller-tools/pkg/loader"
)

// OutputToDirectory outputs each artifact to the given directory, regardless
// of if it's package-associated or not.
type OutputToDirectory string

func (o OutputToDirectory) Open(_ *loader.Package, itemPath string) (io.WriteCloser, error) {
	// ensure the directory exists
	dir := path.Dir(filepath.Join(string(o), itemPath))
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, err
	}
	path := filepath.Join(string(o), itemPath)
	return os.Create(path)
}

// OutputArtifacts outputs artifacts to different locations, depending on
// whether they're package-associated or not.
//
// Non-package associated artifacts
// are output to the Config directory, while package-associated ones are output
// to their package's source files' directory, unless an alternate path is
// specified in Code.
type OutputArtifacts struct {
	// Config points to the directory to which to write configuration.
	Config OutputToDirectory
	// Code overrides the directory in which to write new code (defaults to where the existing code lives).
	Code OutputToDirectory `marker:",optional"`
}

func (o OutputArtifacts) Open(pkg *loader.Package, itemPath string) (io.WriteCloser, error) {
	if pkg == nil {
		return o.Config.Open(pkg, itemPath)
	}

	if o.Code != "" {
		return o.Code.Open(pkg, itemPath)
	}

	if len(pkg.CompiledGoFiles) == 0 {
		return nil, fmt.Errorf("cannot output to a package with no path on disk")
	}
	outDir := filepath.Dir(pkg.CompiledGoFiles[0])
	outPath := filepath.Join(outDir, itemPath)
	return os.Create(outPath)
}
