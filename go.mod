module github.com/libbusinessid/spec

go 1.25.0

// The build toolchain is pinned so that every environment compiles with the
// same standard library. A patched toolchain is a security requirement: an
// unpatched one reaches this module through filepath.WalkDir (GO-2026-4602).
toolchain go1.26.5

require (
	github.com/hashicorp/hcl/v2 v2.24.0
	github.com/zclconf/go-cty v1.19.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/apparentlymart/go-textseg/v17 v17.0.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/text v0.39.0 // indirect
	golang.org/x/tools v0.47.0 // indirect
)
