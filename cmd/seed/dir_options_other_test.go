//go:build !windows

package main

import (
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"
)

// testDirOptions is empty off Windows, where securedir already trusts the
// effective user.
func testDirOptions(testing.TB) securedir.Options { return securedir.Options{} }
