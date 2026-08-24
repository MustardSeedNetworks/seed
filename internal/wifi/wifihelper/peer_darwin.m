#import <Foundation/Foundation.h>
#import <Security/SecCode.h>
#import <Security/SecRequirement.h>

#include <bsm/libbsm.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <sys/un.h>

#include "peer_darwin.h"

static char *peer_err(NSString *message) {
  const char *utf8 = message.UTF8String ?: "peer verification failed";
  size_t len = strlen(utf8);
  char *out = malloc(len + 1);
  if (!out) {
    return NULL;
  }
  memcpy(out, utf8, len + 1);
  return out;
}

char *peer_verify(int fd, const char *requirement) {
  @autoreleasepool {
    audit_token_t token;
    socklen_t len = sizeof(token);
    if (getsockopt(fd, SOL_LOCAL, LOCAL_PEERTOKEN, &token, &len) != 0 || len != sizeof(token)) {
      return peer_err(@"could not read peer audit token");
    }

    NSData *tokenData = [NSData dataWithBytes:&token length:sizeof(token)];
    NSDictionary *attrs = @{(__bridge NSString *)kSecGuestAttributeAudit : tokenData};

    SecCodeRef code = NULL;
    OSStatus status = SecCodeCopyGuestWithAttributes(NULL, (__bridge CFDictionaryRef)attrs,
                                                     kSecCSDefaultFlags, &code);
    if (status != errSecSuccess || !code) {
      return peer_err([NSString stringWithFormat:@"could not identify peer code (OSStatus %d)",
                                                 (int)status]);
    }

    SecRequirementRef req = NULL;
    NSString *reqText = [NSString stringWithUTF8String:requirement];
    status = SecRequirementCreateWithString((__bridge CFStringRef)reqText, kSecCSDefaultFlags, &req);
    if (status != errSecSuccess || !req) {
      CFRelease(code);
      return peer_err([NSString stringWithFormat:@"invalid code requirement (OSStatus %d)",
                                                 (int)status]);
    }

    status = SecCodeCheckValidity(code, kSecCSDefaultFlags, req);
    CFRelease(req);
    CFRelease(code);

    if (status != errSecSuccess) {
      return peer_err([NSString stringWithFormat:@"peer does not satisfy the required signature "
                                                 @"(OSStatus %d)",
                                                 (int)status]);
    }
    return NULL;
  }
}

void peer_free(char *s) { free(s); }
