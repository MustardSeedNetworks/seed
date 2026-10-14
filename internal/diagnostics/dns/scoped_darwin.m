// SPDX-License-Identifier: BUSL-1.1

#include "scoped_darwin.h"

#include <stdlib.h>

#include <CoreFoundation/CoreFoundation.h>
#include <SystemConfiguration/SystemConfiguration.h>

static CFStringRef string_value(CFDictionaryRef dict, CFStringRef key)
{
    CFTypeRef value = CFDictionaryGetValue(dict, key);
    if (value == NULL || CFGetTypeID(value) != CFStringGetTypeID()) {
        return NULL;
    }
    return (CFStringRef)value;
}

// interface_of names the interface a service's DNS entity is scoped to, the
// way configd does for scutil's scoped section: the entity's own
// InterfaceName first, then the service's IPv4 and IPv6 entities.
static CFStringRef interface_of(CFDictionaryRef snapshot, CFStringRef dns_key, CFDictionaryRef dns)
{
    CFStringRef name = string_value(dns, kSCPropInterfaceName);
    if (name != NULL) {
        return name;
    }

    CFRange entity = CFStringFind(dns_key, kSCEntNetDNS, kCFCompareBackwards);
    if (entity.location == kCFNotFound) {
        return NULL;
    }
    CFStringRef service =
        CFStringCreateWithSubstring(NULL, dns_key, CFRangeMake(0, entity.location));

    CFStringRef ip_entities[] = {kSCEntNetIPv4, kSCEntNetIPv6};
    for (size_t i = 0; i < sizeof ip_entities / sizeof ip_entities[0] && name == NULL; i++) {
        CFStringRef key =
            CFStringCreateWithFormat(NULL, NULL, CFSTR("%@%@"), service, ip_entities[i]);
        CFTypeRef ip = CFDictionaryGetValue(snapshot, key);
        CFRelease(key);
        if (ip != NULL && CFGetTypeID(ip) == CFDictionaryGetTypeID()) {
            name = string_value((CFDictionaryRef)ip, kSCPropInterfaceName);
        }
    }
    CFRelease(service);
    return name;
}

static void append_service(CFDictionaryRef snapshot, CFStringRef key, CFTypeRef value,
                           CFMutableStringRef out)
{
    if (!CFStringHasSuffix(key, kSCEntNetDNS) || CFGetTypeID(value) != CFDictionaryGetTypeID()) {
        return;
    }
    CFDictionaryRef dns = (CFDictionaryRef)value;

    CFTypeRef servers = CFDictionaryGetValue(dns, kSCPropNetDNSServerAddresses);
    if (servers == NULL || CFGetTypeID(servers) != CFArrayGetTypeID()) {
        return;
    }
    CFStringRef name = interface_of(snapshot, key, dns);
    if (name == NULL) {
        return;
    }

    for (CFIndex i = 0; i < CFArrayGetCount(servers); i++) {
        CFTypeRef server = CFArrayGetValueAtIndex(servers, i);
        if (CFGetTypeID(server) == CFStringGetTypeID()) {
            CFStringAppendFormat(out, NULL, CFSTR("%@\t%@\n"), name, server);
        }
    }
}

char *scoped_resolvers(void)
{
    SCDynamicStoreRef store = SCDynamicStoreCreate(NULL, CFSTR("seed"), NULL, NULL);
    if (store == NULL) {
        return NULL;
    }

    CFStringRef entities[] = {kSCEntNetDNS, kSCEntNetIPv4, kSCEntNetIPv6};
    CFStringRef patterns[3];
    for (size_t i = 0; i < 3; i++) {
        patterns[i] = SCDynamicStoreKeyCreateNetworkServiceEntity(NULL, kSCDynamicStoreDomainState,
                                                                  kSCCompAnyRegex, entities[i]);
    }
    CFArrayRef pattern_list =
        CFArrayCreate(NULL, (const void **)patterns, 3, &kCFTypeArrayCallBacks);
    for (size_t i = 0; i < 3; i++) {
        CFRelease(patterns[i]);
    }

    // One snapshot, so a service's DNS and IP entities are read together.
    CFDictionaryRef snapshot = SCDynamicStoreCopyMultiple(store, NULL, pattern_list);
    CFRelease(pattern_list);
    CFRelease(store);
    if (snapshot == NULL) {
        return NULL;
    }

    CFIndex      count  = CFDictionaryGetCount(snapshot);
    const void **keys   = malloc(sizeof(void *) * (size_t)count);
    const void **values = malloc(sizeof(void *) * (size_t)count);
    CFDictionaryGetKeysAndValues(snapshot, keys, values);

    CFMutableStringRef out = CFStringCreateMutable(NULL, 0);
    for (CFIndex i = 0; i < count; i++) {
        append_service(snapshot, (CFStringRef)keys[i], (CFTypeRef)values[i], out);
    }
    free(keys);
    free(values);
    CFRelease(snapshot);

    CFIndex size =
        CFStringGetMaximumSizeForEncoding(CFStringGetLength(out), kCFStringEncodingUTF8) + 1;
    char *result = malloc((size_t)size);
    if (result != NULL && !CFStringGetCString(out, result, size, kCFStringEncodingUTF8)) {
        result[0] = '\0';
    }
    CFRelease(out);
    return result;
}
