package afero

import (
	"testing"
)

func TestMkdirAllReadonly(t *testing.T) {
	base := &MemMapFs{}
	ro := &ReadOnlyFs{source: base}

	base.MkdirAll("/home/test", 0o777)
	if err := ro.MkdirAll("/home/test", 0o777); err != nil {
		t.Errorf("Failed to MkdirAll on existing path in ReadOnlyFs: %s", err)
	}

	if err := ro.MkdirAll("/home/test/newdir", 0o777); err == nil {
		t.Error("Creating new dir with MkdirAll on ReadOnlyFs should fail but returned nil")
	}

	base.Create("/home/test/file")
	if err := ro.MkdirAll("/home/test/file", 0o777); err == nil {
		t.Error("Creating new dir with MkdirAll on ReadOnlyFs where a file already exists should fail but returned nil")
	}
}
package afero

import (
	"os"
	"testing"
	"time"
)

func checkForErrPermission(t *testing.T, err error) {
	t.Helper()
	if err == nil || !os.IsPermission(err) {
		t.Errorf("Expected err !=nil && err == ErrPermission, got %[1]T (%[1]v)", err)
	}
}

// Make sure that the ReadOnlyFs filter returns errors that can be
// checked with os.IsPermission
func TestReadOnlyFsErrPermission(t *testing.T) {
	fs := NewReadOnlyFs(NewMemMapFs())

	_, err := fs.Create("test")
	checkForErrPermission(t, err)
	checkForErrPermission(t, fs.Chtimes("test", time.Now(), time.Now()))
	checkForErrPermission(t, fs.Chmod("test", os.ModePerm))
	checkForErrPermission(t, fs.Chown("test", 0, 0))
	checkForErrPermission(t, fs.Mkdir("test", os.ModePerm))
	checkForErrPermission(t, fs.MkdirAll("test", os.ModePerm))
	_, err = fs.OpenFile("test", os.O_CREATE, os.ModePerm)
	checkForErrPermission(t, err)
	checkForErrPermission(t, fs.Remove("test"))
	checkForErrPermission(t, fs.RemoveAll("test"))
	checkForErrPermission(t, fs.Rename("test", "test"))

}
