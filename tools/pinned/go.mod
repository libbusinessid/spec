module github.com/libbusinessid/spec/tools/pinned

go 1.25.0

// The build toolchain is pinned so that every environment compiles with the
// same standard library. A patched toolchain is a security requirement: an
// unpatched one reaches this module through filepath.WalkDir (GO-2026-4602).
toolchain go1.26.5

require (
	golang.org/x/tools v0.49.0
	google.golang.org/protobuf v1.36.12
)

require (
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/telemetry v0.0.0-20260811182544-a038080d80e5 // indirect
)
