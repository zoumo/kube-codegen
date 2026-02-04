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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zoumo/golib/cli"
	"github.com/zoumo/make-rules/pkg/runner"

	"github.com/zoumo/kube-codegen/pkg/codegen"
)

func NewClientGenSubcommand() cli.Command {
	return &clientgenSubCommand{
		CommonOptions: &cli.CommonOptions{},
		goCmd:         runner.NewRunner("go"),
		genOptions:    &genOptions{},
	}
}

type clientgenSubCommand struct {
	*cli.CommonOptions

	goCmd *runner.Runner

	genOptions *genOptions
}

func (c *clientgenSubCommand) Name() string {
	return "client-gen"
}

func (c *clientgenSubCommand) BindFlags(fs *pflag.FlagSet) {
	c.genOptions.BindFlags(fs)
}

func (c *clientgenSubCommand) Complete(cmd *cobra.Command, args []string) error {
	if err := c.CommonOptions.Complete(cmd, args); err != nil {
		return err
	}

	c.genOptions.Workdir = c.Workspace
	if err := c.genOptions.Complete(cmd, args); err != nil {
		return err
	}

	if len(c.genOptions.clientPath) == 0 {
		c.Logger.Info("You are about to generate clients,listers,informers without specifying --client-path")
	}

	return nil
}

func (c *clientgenSubCommand) Validate() error {
	return c.genOptions.Validate()
}

func (c *clientgenSubCommand) Run(cmd *cobra.Command, args []string) error {
	generator := codegen.NewCodeGenerator(
		c.Workspace,
		c.genOptions.module,
		c.Logger,
		c.genOptions.codeGeneratorVersion,
		codegen.ClientGenerators,
		nil,
		c.genOptions.boilerplatePath,
		"",
		c.genOptions.clientPath,
		c.genOptions.inputPackages,
		c.genOptions.inputInternalPackages,
		c.genOptions.clientsetDirName,
		c.genOptions.informersDirName,
		c.genOptions.listersDirName,
		c.genOptions.verbose,
	)

	// run all generators
	return generator.Run(nil)
}