//go:build tools

// Package tools pins the exact versions of the code generation and quality
// tools used by this repository. It is never compiled into any binary.
package tools

import (
	_ "golang.org/x/tools/cmd/goimports"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
