// SPDX-License-Identifier: BUSL-1.1

#ifndef SEED_NETIF_PCI_DARWIN_H
#define SEED_NETIF_PCI_DARWIN_H

#include <stdint.h>

// netif_pci_ids walks the IORegistry from the network interface whose BSD
// name is ifname up to the nearest PCI device and stores that device's vendor
// and device IDs. It returns 1 on success and 0 when the interface does not
// exist or has no PCI device behind it (loopback, tunnels, bridges, USB
// adapters).
int netif_pci_ids(const char *ifname, uint16_t *vendor, uint16_t *device);

#endif
