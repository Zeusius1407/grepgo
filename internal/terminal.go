package cmd

import "os"

// isTerminal reports whether f is attached to a terminal rather than a pipe,
// regular file, or other redirection. It is the standard-library equivalent of
// C's isatty(3): a terminal is a character device, so we inspect the file's
// mode bits instead of calling the platform ioctl directly. This works on both
// Unix and Windows without any external dependency.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
