package goutil

import (
	"errors"
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

// GetGopaths returns the list of Go path directories.
func GetGopaths() []string {
	var all []string
	for _, p := range filepath.SplitList(build.Default.GOPATH) {
		if p == "" || p == build.Default.GOROOT {
			// Empty paths are uninteresting.
			// If the path is the GOROOT, ignore it.
			// People sometimes set GOPATH=$GOROOT.
			// Do not get confused by this common mistake.
			continue
		}
		if strings.HasPrefix(p, "~") {
			// Path segments starting with ~ on Unix are almost always
			// users who have incorrectly quoted ~ while setting GOPATH,
			// preventing it from expanding to $HOME.
			// The situation is made more confusing by the fact that
			// bash allows quoted ~ in $PATH (most shells do not).
			// Do not get confused by this, and do not try to use the path.
			// It does not exist, and printing errors about it confuses
			// those users even more, because they think "sure ~ exists!".
			// The go command diagnoses this situation and prints a
			// useful error.
			// On Windows, ~ is used in short names, such as c:\progra~1
			// for c:\program files.
			continue
		}
		all = append(all, p)
	}
	for k, v := range all {
		// GOPATH should end with / or \
		if strings.HasSuffix(v, "/") || strings.HasSuffix(v, string(os.PathSeparator)) {
			continue
		}
		v += string(os.PathSeparator)
		all[k] = v
	}
	return all
}

// GetFirstGopath gets the first $GOPATH value.
func GetFirstGopath(allowAutomaticGuessing bool) (gopath string, err error) {
	a := GetGopaths()
	if len(a) > 0 {
		gopath = a[0]
	}
	defer func() {
		gopath = strings.Replace(gopath, "/", string(os.PathSeparator), -1)
	}()
	if gopath != "" {
		return
	}
	if !allowAutomaticGuessing {
		err = errors.New("not found GOPATH")
		return
	}
	p, _ := os.Getwd()
	p = strings.Replace(p, "\\", "/", -1) + "/"
	i := strings.LastIndex(p, "/src/")
	if i == -1 {
		err = errors.New("not found GOPATH")
		return
	}
	gopath = p[:i+1]
	return
}
