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
	"fmt"

	"github.com/spf13/pflag"
	"github.com/zoumo/golib/cli/injection"
	"github.com/zoumo/golib/cli/plugin"
	"github.com/zoumo/goset"

	"github.com/zoumo/kube-codegen/pkg/codegen"
)

func NewCodeGenSubcommand() plugin.Subcommand {
	c := &codegenSubcommand{
		DefaultInjectionMixin: &injection.DefaultInjectionMixin{},
		generatorsOpt:         make([]string, 0),
		genOptions:            &genOptions{},
	}

	c.enabledGenerators = []string{
		"deepcopy",
		"defaulter",
		"conversion",
		"register",
		"install",
	}
	c.disabledGenerators = []string{
		"openapi",
		"client",
		"lister",
		"informer",
		"crd",
		"protobuf",
	}

	return c
}

type codegenSubcommand struct {
	*injection.DefaultInjectionMixin

	enabledGenerators  []string
	disabledGenerators []string

	genOptions    *genOptions
	generatorsOpt []string
}

func (c *codegenSubcommand) Name() string {
	return "code-gen"
}

func (c *codegenSubcommand) BindFlags(fs *pflag.FlagSet) {
	// project args
	c.genOptions.BindFlags(fs)
	fs.StringSliceVar(&c.generatorsOpt, "generators", nil, fmt.Sprintf("comma-separated list of generators. generater prefixed with '-' are not generated, generator prefixed with '+' will be generated additionally. e.g. -crd will disable crd generator.  (default generators, enabled: %v, disabled: %v)", c.enabledGenerators, c.disabledGenerators))
}

func (c *codegenSubcommand) PreRun(args []string) error {
	if err := c.genOptions.SetDefault(c.Workspace); err != nil {
		return err
	}

	sorted := codegen.EnabledGenerators(c.enabledGenerators, c.disabledGenerators, c.generatorsOpt)
	enabled := goset.NewSetFromStrings(sorted)
	if enabled.ContainsAny("client", "lister", "informer") && len(c.genOptions.clientPath) == 0 {
		if len(c.genOptions.clientPath) == 0 {
			c.Logger.Info("You are about to generate clients,listers,informers without specifying --client-path")
		}
	}

	if err := c.genOptions.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *codegenSubcommand) Run(args []string) error {
	generator := codegen.NewCodeGenerator(
		c.Workspace,
		c.genOptions.module,
		c.Logger,
		c.genOptions.codeGeneratorVersion,
		c.enabledGenerators,
		c.disabledGenerators,
		c.genOptions.boilerplatePath,
		c.genOptions.apisPath,
		c.genOptions.clientPath,
		c.genOptions.inputPackages,
		c.genOptions.clientsetDirName,
		c.genOptions.informersDirName,
		c.genOptions.listersDirName,
	)

	return generator.Run(c.generatorsOpt)
}
