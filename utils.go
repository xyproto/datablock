package datablock

import (
	"io/ioutil"
)

// MustString returns the contents of the given filename as a string.
// Does not use the cache.  Returns an empty string if there were errors.
func MustString(filename string) string {
	data, err := ioutil.ReadFile(filename)
	if err == nil {
		// No error, return the file as a string
		return string(data)
	}
	// There were errors, return an empty string
	return ""
}
