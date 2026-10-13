// SPDX-License-Identifier: BUSL-1.1

#include "pci_darwin.h"

#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>

// read_id reads a PCI ID property. IOPCIDevice publishes vendor-id and
// device-id as little-endian data whose low two bytes hold the ID.
static int read_id(io_registry_entry_t entry, CFStringRef key, uint16_t *out)
{
    CFTypeRef value = IORegistryEntryCreateCFProperty(entry, key, kCFAllocatorDefault, 0);
    if (value == NULL) {
        return 0;
    }
    int ok = 0;
    if (CFGetTypeID(value) == CFDataGetTypeID() && CFDataGetLength((CFDataRef)value) >= 2) {
        const UInt8 *bytes = CFDataGetBytePtr((CFDataRef)value);
        *out               = (uint16_t)(bytes[0] | bytes[1] << 8);
        ok                 = 1;
    }
    CFRelease(value);
    return ok;
}

int netif_pci_ids(const char *ifname, uint16_t *vendor, uint16_t *device)
{
    io_service_t entry = IOServiceGetMatchingService(
        kIOMainPortDefault, IOBSDNameMatching(kIOMainPortDefault, 0, ifname));
    if (entry == IO_OBJECT_NULL) {
        return 0;
    }
    int found = 0;
    while (!found) {
        if (IOObjectConformsTo(entry, "IOPCIDevice")) {
            found = read_id(entry, CFSTR("vendor-id"), vendor) &&
                    read_id(entry, CFSTR("device-id"), device);
            break;
        }
        io_registry_entry_t parent;
        kern_return_t       kr = IORegistryEntryGetParentEntry(entry, kIOServicePlane, &parent);
        IOObjectRelease(entry);
        if (kr != KERN_SUCCESS) {
            return 0;
        }
        entry = parent;
    }
    IOObjectRelease(entry);
    return found;
}
