// Package version holds nerv-task's own binary version, set at build
// time by goreleaser's ldflags (-X .../internal/version.Binary=<tag>),
// mirroring nerv's own versioning.
package version

// Binary is the released tag this binary was built from (for example
// "v1.0.0"). It stays "dev" for a local, non-release build.
var Binary = "dev"
