//go:build appengine
// +build appengine

package color

// IsTerminal returns true if the file descriptor is terminal which
// is always false on on appengine classic which is a sandboxed PaaS.
func IsTerminal(fd uintptr) bool {
	return false
}
