#ifndef SEED_WIFIHELPER_PEER_H
#define SEED_WIFIHELPER_PEER_H

// peer_verify checks that the process on the other end of a connected unix
// socket satisfies a code-signing requirement. Returns NULL when it does, or a
// malloc'd error message the caller owns and must release with peer_free.
//
// Identification uses the peer's audit token rather than its pid: a pid can be
// recycled between the check and its use, and the token cannot.
char *peer_verify(int fd, const char *requirement);

void peer_free(char *s);

#endif // SEED_WIFIHELPER_PEER_H
