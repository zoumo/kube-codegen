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

package crd

import (
	"go/ast"
	"path"
	"sort"
	"strings"

	"github.com/dave/jennifer/jen"
	"sigs.k8s.io/controller-tools/pkg/crd"
	crdmarkers "sigs.k8s.io/controller-tools/pkg/crd/markers"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

var _ genall.Generator = Generator{}

const (
	// KubeAPIApprovedAnnotation is an annotation that must be set to create a CRD for the k8s.io, *.k8s.io, kubernetes.io, or *.kubernetes.io namespaces.
	// The value should be a link to a URL where the current spec was approved, so updates to the spec should also update the URL.
	// If the API is unapproved, you may set the annotation to a string starting with `"unapproved"`.  For instance, `"unapproved, temporarily squatting"` or `"unapproved, experimental-only"`.  This is discouraged.
	KubeAPIApprovedAnnotation = "api-approved.kubernetes.io"
)

// +controllertools:marker:generateHelp

// Generator generates Install function or CustomResourceDefinition objects.
type Generator struct {
	// AllowDangerousTypes allows types which are usually omitted from CRD generation
	// because they are not recommended.
	//
	// Currently the following additional types are allowed when this is true:
	// float32
	// float64
	//
	// Left unspecified, the default is false
	AllowDangerousTypes *bool `marker:",optional"`

	// MaxDescLen specifies the maximum description length for fields in CRD's OpenAPI schema.
	//
	// 0 indicates drop the description for all fields completely.
	// n indicates limit the description to at most n characters and truncate the description to
	// closest sentence boundary if it exceeds n characters.
	MaxDescLen *int `marker:",optional"`

	// HeaderFile specifies the header text (e.g. license) to prepend to generated files.
	HeaderFile string `marker:",optional"`
	// Year specifies the year to substitute for " YEAR" in the header file.
	Year string `marker:",optional"`
	// genInstall let this generator generate install function.
	GenInstall bool
	// genCRD let this generator generate CustomResourceDefinition object.
	GenCRD bool
}

func (Generator) RegisterMarkers(into *markers.Registry) error {
	return crdmarkers.Register(into)
}

func (g Generator) Generate(ctx *genall.GenerationContext) error {
	var headerText string

	if g.HeaderFile != "" {
		headerBytes, err := ctx.ReadFile(g.HeaderFile)
		if err != nil {
			return err
		}
		headerText = string(headerBytes)
	}
	headerText = strings.ReplaceAll(headerText, " YEAR", " "+g.Year)

	parser := &crd.Parser{
		Collector: ctx.Collector,
		Checker:   ctx.Checker,
		// Perform defaulting here to avoid ambiguity later
		AllowDangerousTypes: g.AllowDangerousTypes != nil && *g.AllowDangerousTypes,
	}

	crd.AddKnownTypes(parser)
	for _, root := range ctx.Roots {
		parser.NeedPackage(root)
	}

	metav1Pkg := crd.FindMetav1(ctx.Roots)
	if metav1Pkg == nil {
		// no objects in the roots, since nothing imported metav1
		return nil
	}

	// TODO: allow selecting a specific object
	kubeKinds := crd.FindKubeKinds(parser, metav1Pkg)
	if len(kubeKinds) == 0 {
		// no objects in the roots
		return nil
	}

	groups := make([]string, 0)

	for groupKind := range kubeKinds {
		parser.NeedCRDFor(groupKind, g.MaxDescLen)
		groups = append(groups, groupKind.Group)
	}

	// protect kubernetes community owned API groups in CRDs
	// see https://github.com/kubernetes/enhancements/pull/1111
	for gk := range parser.CustomResourceDefinitions {
		crd := parser.CustomResourceDefinitions[gk]
		group := crd.Spec.Group
		if strings.HasSuffix(group, ".k8s.io") || strings.HasSuffix(group, ".kubernetes.io") {
			crd.Annotations = map[string]string{
				KubeAPIApprovedAnnotation: "https://github.com/kubernetes/enhancements/pull/1111",
			}
			parser.CustomResourceDefinitions[gk] = crd
		}
	}

	cw := &codeWriter{
		headerText: headerText,
		parser:     parser,
		ctx:        ctx,
	}
	for _, group := range groups {
		goPackageName := ""
		dirName := ""
		for pkg, gv := range parser.GroupVersions {
			if gv.Group == group {
				// use dir name as go package name
				// k8s.io/api/apps/v1 -> apps
				// k8s.io/api/a.b.c/v1 -> abc
				dirName = path.Base(path.Dir(pkg.PkgPath))
				goPackageName = strings.ReplaceAll(dirName, ".", "")
				break
			}
		}
		if goPackageName == "" {
			// use first part of group
			goPackageName = strings.Split(group, ".")[0]
			dirName = goPackageName
		}

		if g.GenInstall {
			if err := cw.GenerateGroupInstall(group, dirName); err != nil {
				return err
			}
		}

		if g.GenCRD {
			if err := cw.GenerateGroup(group, dirName, goPackageName); err != nil {
				return err
			}
		}
	}

	if g.GenInstall {
		if err := cw.GenerateScheme(metav1Pkg); err != nil {
			return err
		}
	}

	return nil
}

func (Generator) CheckFilter() loader.NodeFilter {
	return filterTypesForCRDs
}

// filterTypesForCRDs filters out all nodes that aren't used in CRD generation,
// like interfaces and struct fields without JSON tag.
func filterTypesForCRDs(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.InterfaceType:
		// skip interfaces, we never care about references in them
		return false
	case *ast.StructType:
		return true
	case *ast.Field:
		_, hasTag := loader.ParseAstTag(node.Tag).Lookup("json")
		// fields without JSON tags mean we have custom serialization,
		// so only visit fields with tags.
		return hasTag
	default:
		return true
	}
}

type codeWriter struct {
	headerText string
	parser     *crd.Parser
	ctx        *genall.GenerationContext
}

func (cw *codeWriter) setFileDefault(f *jen.File) {
	f.HeaderComment("// +build !ignore_autogenerated\n")
	f.HeaderComment(cw.headerText + "\n")
	f.HeaderComment("// Code generated by crd-gen. DO NOT EDIT.")

	f.ImportAlias("k8s.io/apimachinery/pkg/apis/meta/v1", "metav1")
	f.ImportAlias("k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1", "apiextensionsv1beta1")
	f.ImportAlias("k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1", "apiextensionsv1")
	f.ImportAlias("k8s.io/apimachinery/pkg/util/runtime", "utilruntime")
}

func (cw *codeWriter) GenerateScheme(metav1Pkg *loader.Package) error {
	schemefile := jen.NewFile("install")
	cw.setFileDefault(schemefile)

	schemefile.Line()
	schemefile.Func().Id("Install").Params(jen.Id("scheme").Op("*").Qual("k8s.io/apimachinery/pkg/runtime", "Scheme")).BlockFunc(func(g *jen.Group) {
		must := jen.Qual("k8s.io/apimachinery/pkg/util/runtime", "Must")
		pkgs := []string{}
		for pkg := range cw.parser.GroupVersions {
			if pkg == metav1Pkg {
				continue
			}
			pkgs = append(pkgs, loader.NonVendorPath(pkg.PkgPath))
		}
		sort.Strings(pkgs)
		for _, pkg := range pkgs {
			// add alias
			alias := path.Base(path.Dir(pkg)) + path.Base(pkg)
			alias = strings.ReplaceAll(alias, ".", "")
			schemefile.ImportAlias(pkg, alias)

			g.Add(must.Clone().Call(jen.Qual(pkg, "AddToScheme").Call(jen.Id("scheme"))))
		}
	})

	w, err := cw.ctx.Open(nil, "install/zz.generated.scheme.go")
	if err != nil {
		return err
	}
	defer w.Close()
	if err := schemefile.Render(w); err != nil {
		return err
	}

	return nil
}

func (cw *codeWriter) GenerateGroupInstall(group string, dirName string) error {
	schemefile := jen.NewFile("install")
	cw.setFileDefault(schemefile)

	schemefile.Line()
	schemefile.Func().Id("Install").Params(jen.Id("scheme").Op("*").Qual("k8s.io/apimachinery/pkg/runtime", "Scheme")).BlockFunc(func(g *jen.Group) {
		must := jen.Qual("k8s.io/apimachinery/pkg/util/runtime", "Must")
		pkgs := []string{}
		for pkg, gv := range cw.parser.GroupVersions {
			if gv.Group != group {
				continue
			}
			pkgs = append(pkgs, loader.NonVendorPath(pkg.PkgPath))
		}
		sort.Strings(pkgs)
		for _, pkg := range pkgs {
			// add alias
			alias := path.Base(path.Dir(pkg)) + path.Base(pkg)
			alias = strings.ReplaceAll(alias, ".", "")
			schemefile.ImportAlias(pkg, alias)

			g.Add(must.Clone().Call(jen.Qual(pkg, "AddToScheme").Call(jen.Id("scheme"))))
		}
	})

	filename := path.Join(dirName, "install", "zz.generated.install.go")
	w, err := cw.ctx.Open(nil, filename)
	if err != nil {
		return err
	}
	defer w.Close()
	if err := schemefile.Render(w); err != nil {
		return err
	}

	return nil
}

func (cw *codeWriter) GenerateGroup(group string, dirName, goPackageName string) error {
	crdsfile := jen.NewFile(goPackageName)
	cw.setFileDefault(crdsfile)

	newCRDs := []jen.Code{}
	for groupKind := range cw.parser.CustomResourceDefinitions {
		if groupKind.Group != group {
			continue
		}
		crd := cw.parser.CustomResourceDefinitions[groupKind]
		value := GenerateValue(&crd)
		crdid := jen.Op("*").Qual("k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1", "CustomResourceDefinition")
		crdsfile.Comment("//nolint")
		crdsfile.Func().Id("New" + Capitalize(groupKind.Kind) + "CRD").Params().Add(crdid.Clone()).Block(
			jen.Return(value),
		)
		crdsfile.Line()
		newCRDs = append(newCRDs, jen.Id("New"+Capitalize(groupKind.Kind)+"CRD").Call())
	}

	slice := jen.Index().Op("*").Qual("k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1", "CustomResourceDefinition")
	crdsfile.Comment("//nolint")
	crdsfile.Func().Id("NewCustomResourceDefinitions").Params().Add(slice.Clone()).Block(
		jen.Return(
			slice.Clone().Values(newCRDs...),
		),
	)

	filename := path.Join(dirName, "zz.generated.crd.go")
	writer, err := cw.ctx.Open(nil, filename)
	if err != nil {
		return err
	}

	defer writer.Close()
	return crdsfile.Render(writer)
}
