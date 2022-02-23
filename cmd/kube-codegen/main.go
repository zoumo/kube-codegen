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

package main

import (
	"fmt"
	"os"

	"github.com/zoumo/golib/log"
	"github.com/zoumo/golib/log/consolog"

	"github.com/zoumo/kube-codegen/cmd/kube-codegen/app"
)

func init() {
	log.SetLogger(consolog.New())
}

func main() {
	command := app.NewRootCommand()
	if err := command.Execute(); err != nil {
		fmt.Printf("run command error: %v\n", err)
		os.Exit(1)
	}
}
