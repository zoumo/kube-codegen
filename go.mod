module github.com/zoumo/kube-codegen

go 1.15

require (
	github.com/dave/jennifer v1.5.0
	github.com/go-logr/logr v0.4.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/otiai10/copy v1.5.0
	github.com/spf13/afero v1.9.5
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/zoumo/golib v0.0.0-20220223062151-794bff922af0
	github.com/zoumo/goset v0.2.0
	github.com/zoumo/make-rules v0.2.0
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	k8s.io/gengo v0.0.0-20210813121822-485abfe95c7c
	sigs.k8s.io/controller-tools v0.5.0
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr => github.com/go-logr/zapr v0.4.0
	github.com/spf13/cobra => github.com/spf13/cobra v1.1.1
	golang.org/x/tools => golang.org/x/tools v0.0.0-20200616133436-c1934b75d054
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.2
	k8s.io/apiserver => k8s.io/apiserver v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
	k8s.io/code-generator => k8s.io/code-generator v0.20.2
	k8s.io/component-base => k8s.io/component-base v0.20.2
	k8s.io/gengo => k8s.io/gengo v0.0.0-20210813121822-485abfe95c7c
	k8s.io/klog => k8s.io/klog v1.0.0
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.9.0
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	k8s.io/utils => k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.5.0
)
