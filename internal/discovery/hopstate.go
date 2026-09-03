package discovery

// A traceroute hop's state is part of the wire shape, so both the Unix and
// Windows tracers produce the same strings. They previously declared them
// twice -- unexported here, exported there -- which is one rename away from
// the two platforms disagreeing about what "unreachable" is spelled like.
// Anything that reasons about hop state across platforms, such as the
// continuous path monitor, needs the single declaration.
const (
	hopStateReply       = "reply"
	hopStateTimeout     = "timeout"
	hopStateError       = "error"
	hopStateUnreachable = "unreachable"
)
