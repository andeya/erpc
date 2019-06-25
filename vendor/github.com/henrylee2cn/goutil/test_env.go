package goutil

import (
	"flag"
	"os"
	"strings"
)

// IsGoTest check if the current process is a test.
func IsGoTest() bool {
	maybe := flag.Lookup("test.v") != nil ||
		flag.Lookup("test.run") != nil ||
		flag.Lookup("test.bench") != nil
	if !maybe {
		return false
	}
	if len(os.Args) == 0 {
		return false
	}
	return strings.HasSuffix(os.Args[0], ".test")
}
