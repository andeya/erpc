package color

// Stdout, Stderr are open Files pointing to
// colorful standard output and colorful standard error file descriptors.
var (
	Stdout = NewColorableStdout()
	Stderr = NewColorableStderr()
)
