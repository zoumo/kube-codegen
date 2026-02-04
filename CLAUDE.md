# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

kube-codegen is a Go code generator for implementing Kubernetes-style API types. It generates deepcopy functions, clients, listers, informers, CRDs, and other Kubernetes-related code.

## Commands

```bash
# Build the binary
go build ./cmd/kube-codegen

# Build all packages
go build ./...

# Run all tests
go test ./...

# Run specific test
go test ./pkg/cli/... -v -run TestFindGoModulePath

# Update dependencies
go mod tidy

# Run with verbose output
go build -v ./cmd/kube-codegen
```

## CLI Architecture

The CLI uses `github.com/zoumo/golib/cli` package with a command pattern:

**Command interface:**
```go
type Command interface {
    Name() string
    Run(cmd *cobra.Command, args []string) error
}
```

**ComplexOptions interface (commands implement this):**
```go
type ComplexOptions interface {
    Options  // BindFlags(fs *pflag.FlagSet)
    Complete(cmd *cobra.Command, args []string) error
    Validate() error
}
```

Commands embed `*cli.CommonOptions` which provides `Workspace` and `Logger` fields. The `cli.NewCobraCommand()` function auto-handles the lifecycle: BindFlags → Complete → Validate → Run.

**Import pattern:** When both `github.com/zoumo/golib/cli` and local `github.com/zoumo/kube-codegen/pkg/cli` are needed, use aliases:
```go
import (
    gcli "github.com/zoumo/golib/cli"
    kubecli "github.com/zoumo/kube-codegen/pkg/cli"
)
```

## Key Packages

| Package | Purpose |
|---------|---------|
| `cmd/kube-codegen/` | CLI entry points and command assembly |
| `pkg/cli/` | code-gen and client-gen commands |
| `pkg/codegen/` | Core CodeGenerator implementation |
| `pkg/generator/crd/` | CRD generation utilities |
| `cmd/crd-gen/` | Standalone CRD generator |

## Dependencies

- Go 1.23.0+
- github.com/zoumo/golib v0.2.2 - CLI framework with CommonOptions
- github.com/zoumo/make-rules v0.3.0 - Runner commands and golang utilities
- github.com/zoumo/goset v0.2.0 - Set operations
- k8s.io/gengo v0.0.0-20210813121822-485abfe95c7c - Kubernetes code generation framework
- sigs.k8s.io/controller-tools v0.5.0 - CRD generation
- github.com/spf13/cobra - CLI framework
- github.com/otiai10/copy - File copy utilities
- github.com/spf13/afero - File system utilities

## Code Patterns

**Logger:** Uses `github.com/zoumo/golib/log.Logger`. Commands access via `c.Logger` from embedded CommonOptions. Do not use logr directly.

**Go module discovery:** Uses `FindGoModulePath()` in `pkg/cli/options.go` which runs `go mod edit -json` to get the module path.

**Generator defaults:** GenOptions uses `Complete()` to set defaults like module path, input packages based on workspace directory.

**Runner utility:** Uses `make-rules/pkg/runner` to execute go commands with proper environment handling (e.g., GOWORK=off).

## Adding New Commands

1. Create a new file in `pkg/cli/` with a struct implementing both `cli.Command` and `cli.ComplexOptions`
2. Embed `*cli.CommonOptions` in the struct
3. Implement `Name()`, `BindFlags()`, `Complete()`, `Validate()`, `Run()`
4. Register in `cmd/kube-codegen/app/codegen.go` using `gcli.NewCobraCommand()`