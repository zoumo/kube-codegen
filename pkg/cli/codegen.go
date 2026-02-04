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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zoumo/golib/cli"
	"github.com/zoumo/goset"

	"github.com/zoumo/kube-codegen/pkg/codegen"
)

func NewCodeGenSubcommand() cli.Command {
	c := &codegenSubcommand{
		CommonOptions:   &cli.CommonOptions{},
		generatorsOpt:   make([]string, 0),
		genOptions:      &genOptions{},
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
	*cli.CommonOptions

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

func (c *codegenSubcommand) Complete(cmd *cobra.Command, args []string) error {
	if err := c.CommonOptions.Complete(cmd, args); err != nil {
		return err
	}

	c.genOptions.Workdir = c.Workspace
	if err := c.genOptions.Complete(cmd, args); err != nil {
		return err
	}

	sorted := codegen.EnabledGenerators(c.enabledGenerators, c.disabledGenerators, c.generatorsOpt)
	enabled := goset.NewSetFromStrings(sorted)
	if enabled.ContainsAny("client", "lister", "informer") && len(c.genOptions.clientPath) == 0 {
		if len(c.genOptions.clientPath) == 0 {
			c.Logger.Info("You are about to generate clients,listers,informers without specifying --client-path")
		}
	}

	return nil
}

func (c *codegenSubcommand) Validate() error {
	return c.genOptions.Validate()
}

func (c *codegenSubcommand) Run(cmd *cobra.Command, args []string) error {
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
		c.genOptions.inputInternalPackages,
		c.genOptions.clientsetDirName,
		c.genOptions.informersDirName,
		c.genOptions.listersDirName,
		c.genOptions.verbose,
	)

	return generator.Run(c.generatorsOpt)
}