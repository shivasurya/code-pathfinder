//go:build noupdatecheck

// Package updatecheck provides a no-op implementation when the binary is
// compiled with -tags noupdatecheck. All calls to Check return nil immediately.
package updatecheck

import "context"

// DefaultManifestURL is the CDN URL for the latest.json manifest.
const DefaultManifestURL = "https://assets.codepathfinder.dev/pathfinder/latest.json"

// Check is a no-op stub compiled when the noupdatecheck build tag is set.
// Distros and packagers that want to compile the update-check feature out
// entirely should pass -tags noupdatecheck to the Go toolchain.
func Check(_ context.Context, _, _ string, _ Options) *Result { return nil }
