package datablock

import (
	"github.com/bmizerany/assert"
	"testing"
)

func TestInterface(t *testing.T) {
	fs := NewFileStat(false, 0)
	if !fs.isDir(".") {
		t.Error("isDir failed to recognize .")
	}
	if !fs.isDir("/") {
		t.Error("isDir failed to recognize /")
	}
}

func TestExists(t *testing.T) {
	fs := NewFileStat(false, 0)
	assert.Equal(t, fs.exists("LICENSE"), true)
	assert.NotEqual(t, fs.exists("LICENSES-SCHMISENCES"), true)
}
