// SPDX-License-Identifier: BUSL-1.1

#ifndef SEED_DNS_SCOPED_DARWIN_H
#define SEED_DNS_SCOPED_DARWIN_H

// scoped_resolvers returns one "<interface>\t<server>\n" line per resolver a
// network service has scoped to an interface, read from the dynamic store. It
// returns NULL when the store cannot be read. The caller frees the result.
char *scoped_resolvers(void);

#endif
