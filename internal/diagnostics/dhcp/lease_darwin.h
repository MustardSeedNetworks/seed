// SPDX-License-Identifier: BUSL-1.1

#ifndef SEED_DHCP_LEASE_DARWIN_H
#define SEED_DHCP_LEASE_DARWIN_H

// dhcp_lease returns the DHCP lease configd holds for the interface named
// ifname, read from the dynamic store, as "<key>\t<value>\n" lines:
//
//   address\t<the IPv4 address the service bound>
//   start\t<lease start, Unix seconds>
//   expiry\t<lease expiry, Unix seconds>
//   option\t<code>\t<the option's bytes in hex>
//
// It returns an empty string when the interface holds no lease and NULL when
// the store cannot be read. The caller frees the result.
char *dhcp_lease(const char *ifname);

#endif
