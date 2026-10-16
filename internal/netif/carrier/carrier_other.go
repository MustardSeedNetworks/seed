//go:build !darwin

package carrier

// mediaActive defers to IFF_RUNNING, which tracks carrier on this platform.
func mediaActive(string) (bool, bool) { return false, false }
