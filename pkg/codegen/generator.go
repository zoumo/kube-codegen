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

package codegen

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/otiai10/copy"
	"github.com/zoumo/goset"
	"github.com/zoumo/make-rules/pkg/golang"
	"github.com/zoumo/make-rules/pkg/runner"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/parser"

	"github.com/zoumo/kube-codegen/cmd/crd-gen/app"
)

var (
	ClientGenerators = []string{
		"client",
		"lister",
		"informer",
	}
	sortedValidGenerators = []string{
		"deepcopy",
		"defaulter",
		"conversion",
		"register",
		"install",
		"crd",
		"openapi",
		"protobuf",
		"client",
		"lister",
		"informer",
	}
	validGenerators = goset.NewSetFromStrings(sortedValidGenerators)
)

type CodeGenerator struct {
	workspace       string
	workspaceModule string
	logger          logr.Logger

	goCmd                *runner.Runner
	gomodHelper          *golang.GomodHelper
	enabledGenerators    []string
	disabledGenerators   []string
	codeGeneratorVersion string

	inputPackages []string

	boilerplatePath  string
	apisPath         string
	clientPath       string
	clientsetDirName string
	listerDirName    string
	informerDirName  string

	outputBase string
}

func NewCodeGenerator(
	workspace string,
	workspaceModule string,
	logger logr.Logger,
	codeGeneratorVersion string,
	enabledGenerators, disabledGenerators []string,
	boilerplatePath, apisPath, clientPath string,
	inputPackages []string,
) *CodeGenerator {
	c := &CodeGenerator{
		workspace:            workspace,
		workspaceModule:      workspaceModule,
		logger:               logger,
		goCmd:                runner.NewRunner("go"),
		gomodHelper:          golang.NewGomodHelper(path.Join(workspace, "go.mod"), logger),
		enabledGenerators:    make([]string, 0),
		disabledGenerators:   make([]string, 0),
		codeGeneratorVersion: codeGeneratorVersion,
		inputPackages:        inputPackages,

		boilerplatePath:  boilerplatePath,
		apisPath:         apisPath,
		clientPath:       clientPath,
		outputBase:       path.Join(workspace, "generated"),
		clientsetDirName: "kubernetes",
		listerDirName:    "listers",
		informerDirName:  "informers",
	}

	enabled, disabled := goset.NewSet(), goset.NewSet()

	for _, g := range enabledGenerators {
		if validGenerators.Contains(g) {
			enabled.Add(g) //nolint
		}
	}
	for _, g := range disabledGenerators {
		if validGenerators.Contains(g) {
			disabled.Add(g) //nolint
		}
	}
	c.enabledGenerators = enabled.ToStrings()
	c.disabledGenerators = disabled.ToStrings()
	return c
}

func (c *CodeGenerator) Run(generators []string) error {
	// clean up generated dir
	os.RemoveAll(c.outputBase)

	// detect code-generator version
	if c.codeGeneratorVersion == "" {
		bytes, err := c.goCmd.RunOutput("list", "-f", "{{if .Replace}}{{.Replace.Version}}{{else}}{{.Version}}{{end}}", "-m", "k8s.io/code-generator")
		if err != nil {
			return err
		}
		c.codeGeneratorVersion = strings.TrimSpace(string(bytes))
	}

	// do generation
	if err := c.doGenerate(generators); err != nil {
		return err
	}

	// copy and clean
	if err := c.postRun(); err != nil {
		return err
	}
	return nil
}

func (c *CodeGenerator) postRun() error {
	// copy generated files
	_, err := os.Stat(c.outputBase)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if os.IsNotExist(err) {
		// not generated
		return nil
	}

	// generated
	src := path.Join(c.outputBase, c.workspaceModule)
	dst := c.workspace
	c.logger.Info("copying", "src", src, "dst", dst)
	if err := copy.Copy(src, dst); err != nil {
		return err
	}

	// clean up generated
	os.RemoveAll(c.outputBase)
	return nil
}

func (c *CodeGenerator) doGenerate(generators []string) error {
	sorted := EnabledGenerators(c.enabledGenerators, c.disabledGenerators, generators)

	c.logger.Info("before generating",
		"generators", sorted,
		"inputPackages", c.inputPackages,
		"codeGeneratorVersion", c.codeGeneratorVersion,
	)

	for _, g := range sorted {
		if err := c.doGen(g); err != nil {
			return err
		}
	}
	return nil
}

func (c *CodeGenerator) installCodeGenerator(name string) error {
	_, err := c.goCmd.WithEnvs("GOBIN", path.Join(c.workspace, "bin")).RunCombinedOutput("get", "-v", fmt.Sprintf("k8s.io/code-generator/cmd/%s@%s", name, c.codeGeneratorVersion))
	if err != nil {
		return err
	}
	return nil
}

func (c *CodeGenerator) installProtocGenGoGo() error {
	_, err := c.goCmd.WithEnvs("GOBIN", path.Join(c.workspace, "bin")).RunCombinedOutput("get", "-v", fmt.Sprintf("k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo@%s", c.codeGeneratorVersion))
	if err != nil {
		return err
	}
	return nil
}

func (c *CodeGenerator) doGen(generator string) error {
	if !validGenerators.Contains(generator) {
		return nil
	}
	runner, err := c.prepareRunner(generator)
	if err != nil {
		return err
	}
	switch generator {
	case "deepcopy":
		return c.genDeepcopy(runner)
	case "defaulter":
		return c.genDefaulter(runner)
	case "conversion":
		return c.genConversion(runner)
	case "register":
		return c.genRegister(runner)
	case "openapi":
		return c.genOpenapi(runner)
	case "crd":
		return c.genCRD(runner)
	case "install":
		return c.genInstall(runner)
	case "protobuf":
		return c.genProtobuf(runner)
	case "lister":
		return c.genLister(runner)
	case "client":
		return c.genClient(runner)
	case "informer":
		return c.genInformer(runner)
	}
	return nil
}

func (c *CodeGenerator) prepareRunner(generator string) (*runner.Runner, error) {
	switch generator {
	case "crd", "install":
		return nil, nil
	case "protobuf":
		generator = "go-to-protobuf"
		if err := c.installProtocGenGoGo(); err != nil {
			return nil, err
		}
	default:
		generator += "-gen"
	}

	if err := c.installCodeGenerator(generator); err != nil {
		return nil, err
	}
	newPath := fmt.Sprintf("%s:%s", path.Join(c.workspace, "bin"), os.Getenv("PATH"))
	run := runner.NewRunner(path.Join(c.workspace, "bin", generator)).WithEnvs("PATH", newPath)
	return run, nil
}

func (c *CodeGenerator) genDeepcopy(run *runner.Runner) error {
	generatorName := "deepcopy-gen"
	inputDirs := strings.Join(c.inputPackages, ",")
	outputPackage := path.Join(c.workspaceModule, c.apisPath)

	args := []string{
		"--go-header-file", c.boilerplatePath,
		"--input-dirs", inputDirs,
		"--output-base", c.outputBase,
		"--output-package", outputPackage,
		"--output-file-base", "zz_generated.deepcopy",
		"--bounding-dirs", path.Join(c.workspaceModule, c.apisPath),
	}
	c.logger.Info(generatorName, "args", strings.Join(args, " "))
	_, err := run.RunCombinedOutput(args...)
	if err != nil {
		c.logger.Error(err, "failed to run generator", "generator", generatorName)
		return err
	}
	return nil
}

func (c *CodeGenerator) genDefaulter(run *runner.Runner) error {
	generatorName := "defaulter-gen"

	inputDirs := strings.Join(c.inputPackages, ",")
	outputPackage := path.Join(c.workspaceModule, c.apisPath)
	args := []string{
		"--go-header-file", c.boilerplatePath,
		"--input-dirs", inputDirs,
		"--output-base", c.outputBase,
		"--output-package", outputPackage,
		"--output-file-base", "zz_generated.defaults",
	}

	c.logger.Info(generatorName, "args", strings.Join(args, " "))
	_, err := run.RunCombinedOutput(args...)
	if err != nil {
		c.logger.Error(err, "failed to run generator", "generator", generatorName)
		return err
	}
	return nil
}

func (c *CodeGenerator) genConversion(run *runner.Runner) error {
	generatorName := "conversion-gen"
	inputDirs := strings.Join(c.inputPackages, ",")
	outputPackage := path.Join(c.workspaceModule, c.apisPath)

	args := []string{
		"--go-header-file", c.boilerplatePath,
		"--input-dirs", inputDirs,
		"--output-base", c.outputBase,
		"--output-package", outputPackage,
		"--output-file-base", "zz_generated.conversion",
	}
	c.logger.Info(generatorName, "args", strings.Join(args, " "))
	_, err := run.RunCombinedOutput(args...)
	if err != nil {
		c.logger.Error(err, "failed to run generator", "generator", generatorName)
		return err
	}
	return nil
}

func (c *CodeGenerator) genRegister(run *runner.Runner) error {
	generatorName := "register-gen"
	inputDirs := strings.Join(c.inputPackages, ",")
	outputPackage := path.Join(c.workspaceModule, c.apisPath)

	args := []string{
		"--go-header-file", c.boilerplatePath,
		"--input-dirs", inputDirs,
		"--output-base", c.outputBase,
		"--output-package", outputPackage,
	}
	c.logger.Info(generatorName, "args", strings.Join(args, " "))
	_, err := run.RunCombinedOutput(args...)
	if err != nil {
		c.logger.Error(err, "failed to run generator", "generator", generatorName)
		return err
	}
	return nil
}

func (c *CodeGenerator) genOpenapi(run *runner.Runner) error {
	generatorName := "openapi-gen"
	inputs := []string{
		"k8s.io/apimachinery/pkg/apis/meta/v1",
		"k8s.io/apimachinery/pkg/api/resource",
		"k8s.io/apimachinery/pkg/version",
		"k8s.io/apimachinery/pkg/runtime",
		"k8s.io/apimachinery/pkg/util/intstr",
	}
	inputDirs := strings.Join(append(inputs, c.inputPackages...), ",")
	outputPackage := path.Join(c.workspaceModule, c.apisPath, "generated/openapi")
	violations := path.Join(c.workspace, c.apisPath, "generated/openapi/violations.report")
	if err := os.MkdirAll(path.Dir(violations), 0755); err != nil {
		return err
	}

	args := []string{
		"--go-header-file", c.boilerplatePath,
		"--input-dirs", inputDirs,
		"--output-base", c.outputBase,
		"--output-package", outputPackage,
		"--report-filename", violations,
	}
	c.logger.Info(generatorName, "args", strings.Join(args, " "))
	_, err := run.RunCombinedOutput(args...)
	if err != nil {
		c.logger.Error(err, "failed to run generator", "generator", generatorName)
		return err
	}
	return nil
}

func (c *CodeGenerator) genCRD(_ *runner.Runner) error {
	generatorName := "crd-gen"
	cmd := app.NewRootCommand()
	args := []string{
		"crd:headerFile=" + c.boilerplatePath + ",genCRD=true,genInstall=false",
		"paths=" + path.Join(c.workspace, c.apisPath, "..."),
		"output:crd:dir=" + path.Join(c.workspace, c.apisPath),
	}
	c.logger.Info(generatorName, "args", strings.Join(args, " "))
	cmd.SetArgs(args)
	return cmd.Execute()
}

func (c *CodeGenerator) genInstall(_ *runner.Runner) error {
	generatorName := "install-gen"
	cmd := app.NewRootCommand()
	args := []string{
		"crd:headerFile=" + c.boilerplatePath + ",genCRD=false,genInstall=true",
		"paths=" + path.Join(c.workspace, c.apisPath, "..."),
		"output:crd:dir=" + path.Join(c.workspace, c.apisPath),
	}
	c.logger.Info(generatorName, "args", strings.Join(args, " "))
	cmd.SetArgs(args)
	return cmd.Execute()
}

// create all modules symlinks in temp dir for protobuf generator
func (c *CodeGenerator) linkAllModulesInTempDir() (string, error) {
	tempDir, _ := ioutil.TempDir("", "proto-gen.*")

	_, err := c.goCmd.RunCombinedOutput("mod", "download")
	if err != nil {
		return "", err
	}

	mods, err := c.gomodHelper.ParseListMod()
	if err != nil {
		return "", err
	}

	// Get all the modules we use and create required directory structure
	allDirs := goset.NewSet()
	for _, m := range mods {
		dir := path.Join(tempDir, path.Dir(m.Path))
		allDirs.Add(dir) //nolint
	}
	uniqDirs := allDirs.ToStrings()
	sort.Strings(uniqDirs)
	for _, dir := range uniqDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
	}
	// create symlinks
	for _, m := range mods {
		if err := os.Symlink(m.Dir, path.Join(tempDir, m.Path)); err != nil {
			nerr := errors.Unwrap(err)
			if os.IsExist(nerr) {
				// ignore exists error
				continue
			}
			return "", err
		}
	}
	return tempDir, nil
}

func (c *CodeGenerator) genProtobuf(run *runner.Runner) error {
	generatorName := "go-to-protobuf"

	inputDirs := strings.Join(c.inputPackages, ",")

	// create protobuf generator import environment
	tempDir, err := c.linkAllModulesInTempDir()
	if err != nil {
		return err
	}

	// copy types to output path, let generator to overwrite protobuf struct tag
	for _, pkg := range c.inputPackages {
		rel, _ := filepath.Rel(c.workspaceModule, pkg)
		localPath := path.Join(c.workspace, rel)
		if err := copy.Copy(localPath, path.Join(c.outputBase, pkg)); err != nil {
			return err
		}
	}

	// detect apimachinery packages
	b := parser.New()
	b.AddBuildTags("proto")
	for _, pkg := range c.inputPackages {
		if err := b.AddDir(pkg); err != nil {
			return err
		}
	}
	ctx, err := generator.NewContext(
		b,
		namer.NameSystems{
			"public": namer.NewPublicNamer(3),
		},
		"public",
	)
	if err != nil {
		return err
	}
	apimachineries := []string{
		`k8s.io/apimachinery/pkg/util/intstr`,
		`k8s.io/apimachinery/pkg/api/resource`,
		`k8s.io/apimachinery/pkg/runtime/schema`,
		`k8s.io/apimachinery/pkg/runtime`,
		`k8s.io/apimachinery/pkg/apis/meta/v1`,
		`k8s.io/apimachinery/pkg/apis/meta/v1beta1`,
		`k8s.io/apimachinery/pkg/apis/testapigroup/v1`,
	}
	for _, pkg := range c.inputPackages {
		for _, d := range ctx.Universe[pkg].Imports {
			if strings.HasPrefix(d.Path, "k8s.io/api") {
				apimachineries = append(apimachineries, d.Path)
			}
		}
	}

	for i := range apimachineries {
		api := apimachineries[i]
		api = fmt.Sprintf("-%s=%s", api, protoSafeOutermostPackage(api))
		apimachineries[i] = api
	}

	args := []string{
		"--go-header-file", c.boilerplatePath,
		"--proto-import", tempDir,
		"--proto-import", path.Join(tempDir, "github.com/gogo/protobuf/protobuf"),
		"--packages", inputDirs,
		"--output-base", c.outputBase,
		"--apimachinery-packages", strings.Join(apimachineries, ","),
	}
	c.logger.Info(generatorName, "args", strings.Join(args, " "))
	_, err = run.RunCombinedOutput(args...)
	if err != nil {
		c.logger.Error(err, "failed to run generator", "generator", generatorName)
		return err
	}
	return nil
}

func (c *CodeGenerator) genClient(run *runner.Runner) error {
	generatorName := "client-gen"

	input := strings.Join(c.inputPackages, ",")
	outputPackage := path.Join(c.workspaceModule, c.clientPath)

	localClientsetPath := path.Join(c.workspace, c.clientPath, c.clientsetDirName)
	outputClientsetPath := path.Join(c.outputBase, outputPackage, c.clientsetDirName)
	err := copyExpansions(c.logger, localClientsetPath, outputClientsetPath)
	if err != nil {
		return err
	}
	args := []string{
		"--go-header-file", c.boilerplatePath,
		"--input-base", "",
		"--input", input,
		"--clientset-name", c.clientsetDirName,
		"--output-base", c.outputBase,
		"--output-package", outputPackage,
	}
	c.logger.Info(generatorName, "args", strings.Join(args, " "))
	_, err = run.RunCombinedOutput(args...)
	if err != nil {
		c.logger.Error(err, "failed to run generator", "generator", generatorName)
		return err
	}
	return nil
}

func (c *CodeGenerator) genLister(run *runner.Runner) error {
	generatorName := "lister-gen"

	inputDirs := strings.Join(c.inputPackages, ",")
	outputPackage := path.Join(c.workspaceModule, c.clientPath, c.listerDirName)

	localListersPath := path.Join(c.workspace, c.clientPath, c.listerDirName)
	outputListersPath := path.Join(c.outputBase, outputPackage)
	err := copyExpansions(c.logger, localListersPath, outputListersPath)
	if err != nil {
		return err
	}
	args := []string{
		"--go-header-file", c.boilerplatePath,
		"--input-dirs", inputDirs,
		"--output-base", c.outputBase,
		"--output-package", outputPackage,
	}
	c.logger.Info(generatorName, "args", strings.Join(args, " "))
	_, err = run.RunCombinedOutput(args...)
	if err != nil {
		c.logger.Error(err, "failed to run generator", "generator", generatorName)
		return err
	}
	return nil
}

func (c *CodeGenerator) genInformer(run *runner.Runner) error {
	generatorName := "informer-gen"
	inputDirs := strings.Join(c.inputPackages, ",")
	outputPackage := path.Join(c.workspaceModule, c.clientPath, c.informerDirName)

	versionedClientsetPackage := path.Join(c.workspaceModule, c.clientPath, c.clientsetDirName)
	listersPacakge := path.Join(c.workspaceModule, c.clientPath, c.listerDirName)
	args := []string{
		"--go-header-file", c.boilerplatePath,
		"--input-dirs", inputDirs,
		"--output-base", c.outputBase,
		"--output-package", outputPackage,
		"--single-directory",
		"--versioned-clientset-package", versionedClientsetPackage,
		"--listers-package", listersPacakge,
	}
	c.logger.Info(generatorName, "args", strings.Join(args, " "))
	_, err := run.RunCombinedOutput(args...)
	if err != nil {
		c.logger.Error(err, "failed to run generator", "generator", generatorName)
		return err
	}
	return nil
}

func copyExpansions(logger logr.Logger, srcPrefix, dstPrefix string) error {
	_, err := os.Stat(srcPrefix)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	expansions, err := findExpansions(srcPrefix)
	if err != nil {
		return err
	}
	for _, expansion := range expansions {
		rel, err := filepath.Rel(srcPrefix, expansion)
		if err != nil {
			return err
		}
		target := filepath.Join(dstPrefix, rel)

		logger.Info("copying expansions", "src", expansion, "dst", target)
		if err = copy.Copy(expansion, target); err != nil {
			return err
		}
	}
	return nil
}

func findExpansions(root string) ([]string, error) {
	expansions := []string{}
	err := filepath.Walk(root, func(fpath string, info os.FileInfo, ierr error) error {
		if ierr != nil {
			return ierr
		}
		if fpath == root {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == "generated_expansion.go" || info.Name() == "expansion_generated.go" {
			return nil
		}
		// *_expansion.go excludes generated_expansion.go
		if strings.HasSuffix(info.Name(), "_expansion.go") {
			expansions = append(expansions, fpath)
		}
		return nil
	})
	return expansions, err
}

func protoSafeOutermostPackage(name string) string {
	pkg := strings.Replace(name, "/", ".", -1)
	pkg = strings.Replace(pkg, "-", "_", -1)
	pkg = "." + pkg
	return pkg
}

func EnabledGenerators(defaultEnabledGenerators, defaultDisabledGenerators, generatorsOptions []string) []string {
	all := goset.NewSet()
	disabled := goset.NewSet()
	enabled := goset.NewSet()
	target := goset.NewSet()

	for _, g := range defaultDisabledGenerators {
		// disabled by default
		disabled.Add(g) //nolint
	}
	for _, g := range defaultEnabledGenerators {
		all.Add(g) //nolint
		all = all.Unite(disabled)
	}

	for _, opt := range generatorsOptions {
		if len(opt) == 0 {
			continue
		}
		if opt[0] == '-' {
			disabled.Add(opt[1:]) //nolint
		} else if opt[0] == '+' {
			enabled.Add(opt[1:]) //nolint
		} else {
			target.Add(opt) //nolint
		}
	}

	if target.Len() == 0 {
		// all - disable + enabled
		target = all.Diff(disabled).Unite(enabled)
	}

	sorted := []string{}
	for _, g := range sortedValidGenerators {
		if target.Contains(g) {
			sorted = append(sorted, g)
		}
	}
	return sorted
}
