package afero

import (
	"os"
	"time"
)

var _ Lstater = (*ReadOnlyFs)(nil)
var _ Fs = (*ReadOnlyFs)(nil)

type ReadOnlyFs struct {
	source Fs
}

func NewReadOnlyFs(source Fs) *ReadOnlyFs {
	return &ReadOnlyFs{source: source}
}

func (r *ReadOnlyFs) ReadDir(name string) ([]os.FileInfo, error) {
	return ReadDir(r.source, name)
}

func (r *ReadOnlyFs) Chtimes(n string, a, m time.Time) error {
	return os.ErrPermission
}

func (r *ReadOnlyFs) Chmod(n string, m os.FileMode) error {
	return os.ErrPermission
}

func (r *ReadOnlyFs) Chown(n string, uid, gid int) error {
	return os.ErrPermission
}

func (r *ReadOnlyFs) Name() string {
	return "ReadOnlyFilter"
}

func (r *ReadOnlyFs) Stat(name string) (os.FileInfo, error) {
	return r.source.Stat(name)
}

func (r *ReadOnlyFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	if lsf, ok := r.source.(Lstater); ok {
		return lsf.LstatIfPossible(name)
	}
	fi, err := r.Stat(name)
	return fi, false, err
}

func (r *ReadOnlyFs) SymlinkIfPossible(oldname, newname string) error {
	return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: ErrNoSymlink}
}

func (r *ReadOnlyFs) ReadlinkIfPossible(name string) (string, error) {
	if srdr, ok := r.source.(LinkReader); ok {
		return srdr.ReadlinkIfPossible(name)
	}

	return "", &os.PathError{Op: "readlink", Path: name, Err: ErrNoReadlink}
}

func (r *ReadOnlyFs) Rename(o, n string) error {
	return os.ErrPermission
}

func (r *ReadOnlyFs) RemoveAll(p string) error {
	return os.ErrPermission
}

func (r *ReadOnlyFs) Remove(n string) error {
	return os.ErrPermission
}

func (r *ReadOnlyFs) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, os.ErrPermission
	}
	return r.source.OpenFile(name, flag, perm)
}

func (r *ReadOnlyFs) Open(n string) (File, error) {
	return r.source.Open(n)
}

func (r *ReadOnlyFs) Mkdir(n string, p os.FileMode) error {
	return os.ErrPermission
}

func (r *ReadOnlyFs) MkdirAll(n string, p os.FileMode) error {
	fi, err := r.source.Stat(n)
	if err == nil && fi.IsDir() {
		return nil
	}
	return os.ErrPermission
}

func (r *ReadOnlyFs) Create(n string) (File, error) {
	return nil, os.ErrPermission
}
