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
	"github.com/spf13/cobra"
	gcli "github.com/zoumo/golib/cli"
	"github.com/zoumo/make-rules/version"

	kubecli "github.com/zoumo/kube-codegen/pkg/cli"
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "kube-codegen",
		SilenceUsage: true,
	}

	root.AddCommand(NewCodegenCommand())
	root.AddCommand(NewClientGenCommand())
	root.AddCommand(version.NewCommand())
	return root
}

func NewCodegenCommand() *cobra.Command {
	cmd := gcli.NewCobraCommand(kubecli.NewCodeGenSubcommand())
	cmd.Short = "code-gen runs golang code-generators for apis in local repository, used to implement Kubernetes-style API types."
	return cmd
}

func NewClientGenCommand() *cobra.Command {
	cmd := gcli.NewCobraCommand(kubecli.NewClientGenSubcommand())
	cmd.Short = "client-gen runs client-gen,lister-gen,informer-gen code-generators for apis in local or remote repository, used to implement Kubernetes-style clients sdk."
	return cmd
}