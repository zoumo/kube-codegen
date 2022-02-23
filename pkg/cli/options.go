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
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/pflag"
	"github.com/zoumo/goset"
	"github.com/zoumo/make-rules/pkg/runner"
)

var (
	versionRegexp = regexp.MustCompile("^v(0|[1-9][0-9]*)((alpha|beta)(0|[1-9][0-9]*))?$")
)

type genOptions struct {
	module               string
	boilerplatePath      string
	apisPath             string
	clientPath           string
	groupVersionsOpt     []string
	codeGeneratorVersion string

	apisModule    string
	inputPackages []string
}

func (c *genOptions) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.module, "module", c.module, "generated files go module. If it is empty. kube-codegen will read it from go.mod")
	fs.StringVar(&c.boilerplatePath, "go-header-file", c.boilerplatePath, "go header file path")
	fs.StringVar(&c.codeGeneratorVersion, "code-generator-version", "", "k8s.io/code-generator version. If it is empty, kube-codegen will find the version from go mod")
	fs.StringVar(&c.apisModule, "apis-module", c.apisModule, "the module of api types (e.g. github.com/example/api and k8s.io/api), if it is empty, kube-codgen use module in go.mod")
	fs.StringVar(&c.apisPath, "apis-path", c.apisPath, "apis path relative to group-versions in apis-module, (e.g. pkg/apis). The whole api path will be '<apis-module>/<apis-path>/<group>/<version>'.")
	fs.StringSliceVar(&c.groupVersionsOpt, "group-versions", c.groupVersionsOpt, "the groups and their versions in the format groupA:v1,groupA:v1,groupB:v1,groupC:v2 relative to '<apis-package>/<apis-path>'. Empty means all group versions")
	fs.StringVar(&c.clientPath, "client-path", c.clientPath, "the relative generated client output path, (e.g. pkg/clients). If you want generate client,lister,informer, it should be set")
}

func (c *genOptions) SetDefault(workdir string) error {
	// Try to guess repository if flag is not set.
	if len(c.module) == 0 {
		// true to guess repo from go mod
		repoPath, err := FindGoModulePath(true)
		if err != nil {
			return fmt.Errorf("failed to find go module from mod, you must provide repo name, please set the flag --repo, err: %v", err)
		}
		c.module = repoPath
	}

	if len(c.apisModule) == 0 {
		c.apisModule = c.module
	}

	inputPackages, err := c.inputAPIPackages(workdir)
	if err != nil {
		return err
	}

	c.inputPackages = inputPackages

	return nil
}

func (c *genOptions) Validate() error {
	if len(c.module) == 0 {
		return fmt.Errorf("--repo must be specified")
	}

	if len(c.boilerplatePath) == 0 {
		return fmt.Errorf("--go-header-file must be specified")
	}

	if len(c.inputPackages) == 0 {
		return fmt.Errorf("no apis package found in %v", path.Join(c.apisModule, c.apisPath))
	}
	return nil
}

func (c *genOptions) inputAPIPackages(workdir string) (inputPackages []string, err error) {
	var apiModuleDir string
	if c.apisModule == c.module {
		apiModuleDir = workdir
	} else {
		goCmd := runner.NewRunner("go")
		bytes, err := goCmd.RunOutput("list", "-f", "{{ .Dir }}", "-m", c.apisModule)
		if err != nil {
			return nil, err
		}
		apiModuleDir = strings.TrimSpace(string(bytes))
	}

	root := path.Join(apiModuleDir, c.apisPath)
	// find all apis group version package
	allGroupVersions, err := findGroupVersion(root)
	if err != nil {
		return nil, err
	}

	if len(c.groupVersionsOpt) == 0 {
		for _, gv := range allGroupVersions {
			inputPackages = append(inputPackages, path.Join(c.apisModule, c.apisPath, gv))
		}
		return inputPackages, nil
	}

	allGVSet := goset.NewSetFromStrings(allGroupVersions)
	// filter group version
	for _, gv := range c.groupVersionsOpt {
		if !allGVSet.Contains(gv) {
			continue
		}
		inputPackages = append(inputPackages, path.Join(c.apisModule, c.apisPath, gv))
	}
	return inputPackages, nil
}

// findGroupVersion walk into apis root dir, and find all group/version under this apis path
func findGroupVersion(root string) ([]string, error) {
	groupVersions := []string{}
	err := filepath.Walk(root, func(fpath string, info os.FileInfo, ierr error) error {
		if ierr != nil {
			return ierr
		}
		if fpath == root {
			return nil
		}
		if !info.IsDir() {
			return nil
		}

		sub := strings.TrimPrefix(fpath, root+"/")
		if len(strings.Split(sub, "/")) != 2 {
			return nil
		}
		// fpath = repo/apis/group/version
		// sub = group/version
		if versionRegexp.MatchString(path.Base(sub)) {
			groupVersions = append(groupVersions, sub)
		}
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}
	return groupVersions, nil
}

// module and goMod arg just enough of the output of `go mod edit -json` for our purposes
type goMod struct {
	Module module
}
type module struct {
	Path string
}

// FindGoModulePath finds the path of the current module, if present.
func FindGoModulePath(forceModules bool) (string, error) {
	cmd := exec.Command("go", "mod", "edit", "-json")
	cmd.Env = append(cmd.Env, os.Environ()...)
	if forceModules {
		cmd.Env = append(cmd.Env, "GO111MODULE=on" /* turn on modules just for these commands */)
	}
	out, err := cmd.Output()
	if err != nil {
		if exitErr, isExitErr := err.(*exec.ExitError); isExitErr {
			err = fmt.Errorf("%s", string(exitErr.Stderr))
		}
		return "", err
	}
	mod := goMod{}
	if err := json.Unmarshal(out, &mod); err != nil {
		return "", err
	}
	return mod.Module.Path, nil
}
