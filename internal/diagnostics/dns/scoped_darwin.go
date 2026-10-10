// SPDX-License-Identifier: BUSL-1.1

//go:build darwin

package dns

/*
#cgo LDFLAGS: -framework CoreFoundation -framework SystemConfiguration
#include <stdlib.h>
#include "scoped_darwin.h"
*/
import "C"

import (
	"strings"
	"unsafe"
)

// interfaceResolversPlatform reads the resolvers macOS has scoped to iface.
//
// The scoped configuration lives in the dynamic store, not in a file:
// /etc/resolv.conf is the unscoped answer, which is precisely the one that
// must not be attributed to an interface. It is read in process, from the
// same per-service entities configd builds `scutil --dns` from: any exec
// from the daemon can wedge its child on macOS (#2530).
func interfaceResolversPlatform(iface string) ([]string, bool) {
	table := C.scoped_resolvers()
	if table == nil {
		return nil, false
	}
	defer C.free(unsafe.Pointer(table))
	return scopedResolversFor(C.GoString(table), iface)
}

// scopedResolversFor returns the servers the store's scoped table lists for
// iface, and whether the table can attribute resolvers to an interface at all.
// Each line is "<interface>\t<server>".
//
// An interface with no scoped resolver returns none: answering with the
// host's default resolvers would be the same defect as naming another link's
// gateway (#2690). A table with no scoped resolver for any interface cannot
// say which interface any resolver belongs to, which is a different answer
// from "this interface has none": reporting it as an absence would put every
// interface, the one carrying the route included, behind "no resolvers on
// this interface".
func scopedResolversFor(table, iface string) ([]string, bool) {
	var servers []string
	attributable := false
	for line := range strings.SplitSeq(table, "\n") {
		name, server, found := strings.Cut(line, "\t")
		if !found {
			continue
		}
		attributable = true
		if name == iface {
			servers = append(servers, server)
		}
	}
	return servers, attributable
}
