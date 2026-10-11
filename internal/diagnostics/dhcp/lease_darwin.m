// SPDX-License-Identifier: BUSL-1.1

#include "lease_darwin.h"

#include <stdlib.h>

#include <CoreFoundation/CoreFoundation.h>
#include <SystemConfiguration/SystemConfiguration.h>

static CFStringRef option_prefix = CFSTR("Option_");

static CFTypeRef typed_value(CFDictionaryRef dict, CFStringRef key, CFTypeID type)
{
    CFTypeRef value = CFDictionaryGetValue(dict, key);
    if (value == NULL || CFGetTypeID(value) != type) {
        return NULL;
    }
    return value;
}

static void append_date(CFMutableStringRef out, CFStringRef name, CFDictionaryRef dhcp,
                        CFStringRef key)
{
    CFDateRef date = typed_value(dhcp, key, CFDateGetTypeID());
    if (date != NULL) {
        CFStringAppendFormat(out, NULL, CFSTR("%@\t%.0f\n"), name,
                             CFDateGetAbsoluteTime(date) + kCFAbsoluteTimeIntervalSince1970);
    }
}

// append_option writes one Option_<code> entry. configd stores each option of
// the ACK under that key with the option's raw bytes as its value.
static void append_option(const void *key, const void *value, void *context)
{
    CFMutableStringRef out = context;
    if (CFGetTypeID(key) != CFStringGetTypeID() || CFGetTypeID(value) != CFDataGetTypeID() ||
        !CFStringHasPrefix(key, option_prefix)) {
        return;
    }
    CFStringRef code = CFStringCreateWithSubstring(
        NULL, key,
        CFRangeMake(CFStringGetLength(option_prefix),
                    CFStringGetLength(key) - CFStringGetLength(option_prefix)));
    CFStringAppendFormat(out, NULL, CFSTR("option\t%@\t"), code);
    CFRelease(code);

    const UInt8 *bytes = CFDataGetBytePtr(value);
    for (CFIndex i = 0; i < CFDataGetLength(value); i++) {
        CFStringAppendFormat(out, NULL, CFSTR("%02x"), bytes[i]);
    }
    CFStringAppend(out, CFSTR("\n"));
}

// append_lease writes the lease of the service whose IPv4 entity is bound to
// ifname, if that service has a DHCP entity. The two entities share the
// service's key prefix.
static void append_lease(CFDictionaryRef snapshot, CFStringRef ipv4_key, CFDictionaryRef ipv4,
                         CFStringRef ifname, CFMutableStringRef out)
{
    CFStringRef name = typed_value(ipv4, kSCPropInterfaceName, CFStringGetTypeID());
    if (name == NULL || CFStringCompare(name, ifname, 0) != kCFCompareEqualTo) {
        return;
    }

    CFRange entity = CFStringFind(ipv4_key, kSCEntNetIPv4, kCFCompareBackwards);
    if (entity.location == kCFNotFound) {
        return;
    }
    CFStringRef service =
        CFStringCreateWithSubstring(NULL, ipv4_key, CFRangeMake(0, entity.location));
    CFStringRef dhcp_key =
        CFStringCreateWithFormat(NULL, NULL, CFSTR("%@%@"), service, kSCEntNetDHCP);
    CFRelease(service);
    CFDictionaryRef dhcp = typed_value(snapshot, dhcp_key, CFDictionaryGetTypeID());
    CFRelease(dhcp_key);
    if (dhcp == NULL) {
        return;
    }

    CFArrayRef addresses = typed_value(ipv4, kSCPropNetIPv4Addresses, CFArrayGetTypeID());
    if (addresses != NULL && CFArrayGetCount(addresses) > 0) {
        CFTypeRef address = CFArrayGetValueAtIndex(addresses, 0);
        if (CFGetTypeID(address) == CFStringGetTypeID()) {
            CFStringAppendFormat(out, NULL, CFSTR("address\t%@\n"), address);
        }
    }
    append_date(out, CFSTR("start"), dhcp, CFSTR("LeaseStartTime"));
    append_date(out, CFSTR("expiry"), dhcp, CFSTR("LeaseExpirationTime"));
    CFDictionaryApplyFunction(dhcp, append_option, out);
}

static char *copy_utf8(CFStringRef str)
{
    CFIndex size =
        CFStringGetMaximumSizeForEncoding(CFStringGetLength(str), kCFStringEncodingUTF8) + 1;
    char *result = malloc((size_t)size);
    if (result != NULL && !CFStringGetCString(str, result, size, kCFStringEncodingUTF8)) {
        result[0] = '\0';
    }
    return result;
}

char *dhcp_lease(const char *ifname)
{
    SCDynamicStoreRef store = SCDynamicStoreCreate(NULL, CFSTR("seed"), NULL, NULL);
    if (store == NULL) {
        return NULL;
    }

    CFStringRef entities[] = {kSCEntNetIPv4, kSCEntNetDHCP};
    CFStringRef patterns[2];
    for (size_t i = 0; i < 2; i++) {
        patterns[i] = SCDynamicStoreKeyCreateNetworkServiceEntity(NULL, kSCDynamicStoreDomainState,
                                                                  kSCCompAnyRegex, entities[i]);
    }
    CFArrayRef pattern_list =
        CFArrayCreate(NULL, (const void **)patterns, 2, &kCFTypeArrayCallBacks);
    for (size_t i = 0; i < 2; i++) {
        CFRelease(patterns[i]);
    }

    // One snapshot, so a service's address and its lease are read together.
    CFDictionaryRef snapshot = SCDynamicStoreCopyMultiple(store, NULL, pattern_list);
    CFRelease(pattern_list);
    CFRelease(store);
    if (snapshot == NULL) {
        return NULL;
    }

    CFStringRef name = CFStringCreateWithCString(NULL, ifname, kCFStringEncodingUTF8);

    CFIndex      count  = CFDictionaryGetCount(snapshot);
    const void **keys   = malloc(sizeof(void *) * (size_t)count);
    const void **values = malloc(sizeof(void *) * (size_t)count);
    CFDictionaryGetKeysAndValues(snapshot, keys, values);

    CFMutableStringRef out = CFStringCreateMutable(NULL, 0);
    for (CFIndex i = 0; i < count && name != NULL; i++) {
        if (CFStringHasSuffix(keys[i], kSCEntNetIPv4) &&
            CFGetTypeID(values[i]) == CFDictionaryGetTypeID()) {
            append_lease(snapshot, keys[i], values[i], name, out);
        }
    }
    free(keys);
    free(values);
    if (name != NULL) {
        CFRelease(name);
    }
    CFRelease(snapshot);

    char *result = copy_utf8(out);
    CFRelease(out);
    return result;
}
