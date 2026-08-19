module github.com/libbusinessid/spec/tools/pinned

go 1.25.0

// The build toolchain is pinned so that every environment compiles with the
// same standard library. A patched toolchain is a security requirement: an
// unpatched one reaches this module through filepath.WalkDir (GO-2026-4602).
toolchain go1.26.5

require (
	golang.org/x/tools v0.47.0
	google.golang.org/protobuf v1.36.12
)

require (
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/telemetry v0.0.0-20260625142307-59b4966ccb57 // indirect
)
