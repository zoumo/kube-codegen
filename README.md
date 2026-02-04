# kube-codegen

Golang code-generators used to implement Kubernetes-style API types.

## Features

- **code-gen**: Runs golang code-generators for APIs (deepcopy, defaulter, conversion, register, install)
- **client-gen**: Generates Kubernetes clients, listers, and informers
- **crd-gen**: Generates CRD manifests

## Prerequisites

- Go 1.23.0+

## Usage

```bash
# Build
go build ./cmd/kube-codegen

# Code generation
./kube-codegen code-gen --go-header-file= hack/boilerplate.go.txt \
  --apis-module=github.com/example/api \
  --apis-path=pkg/apis \
  --group-versions=fleet.example.com:v1

# Client generation
./kube-codegen client-gen --go-header-file= hack/boilerplate.go.txt \
  --apis-module=github.com/example/api \
  --apis-path=pkg/apis \
  --client-path=pkg/client

# Generate CRDs
./kube-codegen crd-gen

# Show version
./kube-codegen version
```

## Commands

| Command | Description |
|---------|-------------|
| code-gen | Generate deepcopy, defaulter, conversion, register, install code |
| client-gen | Generate clients, listers, informers |
| crd-gen | Generate CRD manifests |

## Architecture

Uses `github.com/zoumo/golib/cli` for CLI framework with Command and ComplexOptions patterns.

**Key packages:**
- `cmd/kube-codegen/` - CLI entry points
- `pkg/cli/` - Command implementations
- `pkg/codegen/` - Core generator logic
- `pkg/generator/crd/` - CRD generation

## Dependencies

- github.com/zoumo/golib v0.2.2
- github.com/zoumo/make-rules v0.3.0
- k8s.io/gengo
- sigs.k8s.io/controller-tools