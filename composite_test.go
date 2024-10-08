package afero

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

var tempDirs []string

func NewTempOsBaseFs(t *testing.T) Fs {
	name, err := TempDir(NewOsFs(), "", "")
	if err != nil {
		t.Error("error creating tempDir", err)
	}

	tempDirs = append(tempDirs, name)

	return NewBasePathFs(NewOsFs(), name)
}

func CleanupTempDirs(t *testing.T) {
	osfs := NewOsFs()
	type ev struct {
		path string
		e    error
	}

	errs := []ev{}

	for _, x := range tempDirs {
		err := osfs.RemoveAll(x)
		if err != nil {
			errs = append(errs, ev{path: x, e: err})
		}
	}

	for _, e := range errs {
		fmt.Println("error removing tempDir", e.path, e.e)
	}

	if len(errs) > 0 {
		t.Error("error cleaning up tempDirs")
	}
	tempDirs = []string{}
}

func TestUnionCreateExisting(t *testing.T) {
	base := &MemMapFs{}
	roBase := &ReadOnlyFs{source: base}
	ufs := NewCopyOnWriteFs(roBase, &MemMapFs{})

	base.MkdirAll("/home/test", 0o777)
	fh, _ := base.Create("/home/test/file.txt")
	fh.WriteString("This is a test")
	fh.Close()

	fh, err := ufs.OpenFile("/home/test/file.txt", os.O_RDWR, 0o666)
	if err != nil {
		t.Errorf("Failed to open file r/w: %s", err)
	}

	_, err = fh.Write([]byte("####"))
	if err != nil {
		t.Errorf("Failed to write file: %s", err)
	}
	fh.Seek(0, 0)
	data, err := io.ReadAll(fh)
	if err != nil {
		t.Errorf("Failed to read file: %s", err)
	}
	if string(data) != "#### is a test" {
		t.Errorf("Got wrong data")
	}
	fh.Close()

	fh, _ = base.Open("/home/test/file.txt")
	data, _ = io.ReadAll(fh)
	if string(data) != "This is a test" {
		t.Errorf("Got wrong data in base file")
	}
	fh.Close()

	fh, err = ufs.Create("/home/test/file.txt")
	switch err {
	case nil:
		if fi, _ := fh.Stat(); fi.Size() != 0 {
			t.Errorf("Create did not truncate file")
		}
		fh.Close()
	default:
		t.Errorf("Create failed on existing file")
	}
}

func TestUnionMergeReaddir(t *testing.T) {
	base := &MemMapFs{}
	roBase := &ReadOnlyFs{source: base}

	ufs := &CopyOnWriteFs{base: roBase, layer: &MemMapFs{}}

	base.MkdirAll("/home/test", 0o777)
	fh, _ := base.Create("/home/test/file.txt")
	fh.WriteString("This is a test")
	fh.Close()

	fh, _ = ufs.Create("/home/test/file2.txt")
	fh.WriteString("This is a test")
	fh.Close()

	fh, _ = ufs.Open("/home/test")
	files, err := fh.Readdirnames(-1)
	if err != nil {
		t.Errorf("Readdirnames failed")
	}
	if len(files) != 2 {
		t.Errorf("Got wrong number of files: %v", files)
	}
}

func TestExistingDirectoryCollisionReaddir(t *testing.T) {
	base := &MemMapFs{}
	roBase := &ReadOnlyFs{source: base}
	overlay := &MemMapFs{}

	ufs := &CopyOnWriteFs{base: roBase, layer: overlay}

	base.MkdirAll("/home/test", 0o777)
	fh, _ := base.Create("/home/test/file.txt")
	fh.WriteString("This is a test")
	fh.Close()

	overlay.MkdirAll("home/test", 0o777)
	fh, _ = overlay.Create("/home/test/file2.txt")
	fh.WriteString("This is a test")
	fh.Close()

	fh, _ = ufs.Create("/home/test/file3.txt")
	fh.WriteString("This is a test")
	fh.Close()

	fh, _ = ufs.Open("/home/test")
	files, err := fh.Readdirnames(-1)
	if err != nil {
		t.Errorf("Readdirnames failed")
	}
	if len(files) != 3 {
		t.Errorf("Got wrong number of files in union: %v", files)
	}

	fh, _ = overlay.Open("/home/test")
	files, err = fh.Readdirnames(-1)
	if err != nil {
		t.Errorf("Readdirnames failed")
	}
	if len(files) != 2 {
		t.Errorf("Got wrong number of files in overlay: %v", files)
	}
}

func TestNestedDirBaseReaddir(t *testing.T) {
	base := &MemMapFs{}
	roBase := &ReadOnlyFs{source: base}
	overlay := &MemMapFs{}

	ufs := &CopyOnWriteFs{base: roBase, layer: overlay}

	base.MkdirAll("/home/test/foo/bar", 0o777)
	fh, _ := base.Create("/home/test/file.txt")
	fh.WriteString("This is a test")
	fh.Close()

	fh, _ = base.Create("/home/test/foo/file2.txt")
	fh.WriteString("This is a test")
	fh.Close()
	fh, _ = base.Create("/home/test/foo/bar/file3.txt")
	fh.WriteString("This is a test")
	fh.Close()

	overlay.MkdirAll("/", 0o777)

	// Opening something only in the base
	fh, _ = ufs.Open("/home/test/foo")
	list, err := fh.Readdir(-1)
	if err != nil {
		t.Errorf("Readdir failed %s", err)
	}
	if len(list) != 2 {
		for _, x := range list {
			fmt.Println(x.Name())
		}
		t.Errorf("Got wrong number of files in union: %v", len(list))
	}
}

func TestNestedDirOverlayReaddir(t *testing.T) {
	base := &MemMapFs{}
	roBase := &ReadOnlyFs{source: base}
	overlay := &MemMapFs{}

	ufs := &CopyOnWriteFs{base: roBase, layer: overlay}

	base.MkdirAll("/", 0o777)
	overlay.MkdirAll("/home/test/foo/bar", 0o777)
	fh, _ := overlay.Create("/home/test/file.txt")
	fh.WriteString("This is a test")
	fh.Close()
	fh, _ = overlay.Create("/home/test/foo/file2.txt")
	fh.WriteString("This is a test")
	fh.Close()
	fh, _ = overlay.Create("/home/test/foo/bar/file3.txt")
	fh.WriteString("This is a test")
	fh.Close()

	// Opening nested dir only in the overlay
	fh, _ = ufs.Open("/home/test/foo")
	list, err := fh.Readdir(-1)
	if err != nil {
		t.Errorf("Readdir failed %s", err)
	}
	if len(list) != 2 {
		for _, x := range list {
			fmt.Println(x.Name())
		}
		t.Errorf("Got wrong number of files in union: %v", len(list))
	}
}

func TestNestedDirOverlayOsFsReaddir(t *testing.T) {
	defer CleanupTempDirs(t)
	base := NewTempOsBaseFs(t)
	roBase := &ReadOnlyFs{source: base}
	overlay := NewTempOsBaseFs(t)

	ufs := &CopyOnWriteFs{base: roBase, layer: overlay}

	base.MkdirAll("/", 0o777)
	overlay.MkdirAll("/home/test/foo/bar", 0o777)
	fh, _ := overlay.Create("/home/test/file.txt")
	fh.WriteString("This is a test")
	fh.Close()
	fh, _ = overlay.Create("/home/test/foo/file2.txt")
	fh.WriteString("This is a test")
	fh.Close()
	fh, _ = overlay.Create("/home/test/foo/bar/file3.txt")
	fh.WriteString("This is a test")
	fh.Close()

	// Opening nested dir only in the overlay
	fh, _ = ufs.Open("/home/test/foo")
	list, err := fh.Readdir(-1)
	fh.Close()
	if err != nil {
		t.Errorf("Readdir failed %s", err)
	}
	if len(list) != 2 {
		for _, x := range list {
			fmt.Println(x.Name())
		}
		t.Errorf("Got wrong number of files in union: %v", len(list))
	}
}

func TestCopyOnWriteFsWithOsFs(t *testing.T) {
	defer CleanupTempDirs(t)
	base := NewTempOsBaseFs(t)
	roBase := &ReadOnlyFs{source: base}
	overlay := NewTempOsBaseFs(t)

	ufs := &CopyOnWriteFs{base: roBase, layer: overlay}

	base.MkdirAll("/home/test", 0o777)
	fh, _ := base.Create("/home/test/file.txt")
	fh.WriteString("This is a test")
	fh.Close()

	overlay.MkdirAll("home/test", 0o777)
	fh, _ = overlay.Create("/home/test/file2.txt")
	fh.WriteString("This is a test")
	fh.Close()

	fh, _ = ufs.Create("/home/test/file3.txt")
	fh.WriteString("This is a test")
	fh.Close()

	fh, _ = ufs.Open("/home/test")
	files, err := fh.Readdirnames(-1)
	fh.Close()
	if err != nil {
		t.Errorf("Readdirnames failed")
	}
	if len(files) != 3 {
		t.Errorf("Got wrong number of files in union: %v", files)
	}

	fh, _ = overlay.Open("/home/test")
	files, err = fh.Readdirnames(-1)
	fh.Close()
	if err != nil {
		t.Errorf("Readdirnames failed")
	}
	if len(files) != 2 {
		t.Errorf("Got wrong number of files in overlay: %v", files)
	}
}

func TestUnionCacheWrite(t *testing.T) {
	base := &MemMapFs{}
	layer := &MemMapFs{}

	ufs := NewCacheOnReadFs(base, layer, 0)

	base.Mkdir("/data", 0o777)

	fh, err := ufs.Create("/data/file.txt")
	if err != nil {
		t.Errorf("Failed to create file")
	}
	_, err = fh.Write([]byte("This is a test"))
	if err != nil {
		t.Errorf("Failed to write file")
	}

	fh.Seek(0, io.SeekStart)
	buf := make([]byte, 4)
	_, _ = fh.Read(buf)
	fh.Write([]byte(" IS A"))
	fh.Close()

	baseData, _ := ReadFile(base, "/data/file.txt")
	layerData, _ := ReadFile(layer, "/data/file.txt")
	if string(baseData) != string(layerData) {
		t.Errorf("Different data: %s <=> %s", baseData, layerData)
	}
}

func TestUnionCacheExpire(t *testing.T) {
	base := &MemMapFs{}
	layer := &MemMapFs{}
	ufs := &CacheOnReadFs{base: base, layer: layer, cacheTime: 1 * time.Second}

	base.Mkdir("/data", 0o777)

	fh, err := ufs.Create("/data/file.txt")
	if err != nil {
		t.Errorf("Failed to create file")
	}
	_, err = fh.Write([]byte("This is a test"))
	if err != nil {
		t.Errorf("Failed to write file")
	}
	fh.Close()

	fh, _ = base.Create("/data/file.txt")
	// sleep some time, so we really get a different time.Now() on write...
	time.Sleep(2 * time.Second)
	fh.WriteString("Another test")
	fh.Close()

	data, _ := ReadFile(ufs, "/data/file.txt")
	if string(data) != "Another test" {
		t.Errorf("cache time failed: <%s>", data)
	}
}

func TestCacheOnReadFsNotInLayer(t *testing.T) {
	base := NewMemMapFs()
	layer := NewMemMapFs()
	fs := NewCacheOnReadFs(base, layer, 0)

	fh, err := base.Create("/file.txt")
	if err != nil {
		t.Fatal("unable to create file: ", err)
	}

	txt := []byte("This is a test")
	fh.Write(txt)
	fh.Close()

	fh, err = fs.Open("/file.txt")
	if err != nil {
		t.Fatal("could not open file: ", err)
	}

	b, err := ReadAll(fh)
	fh.Close()

	if err != nil {
		t.Fatal("could not read file: ", err)
	} else if !bytes.Equal(txt, b) {
		t.Fatalf("wanted file text %q, got %q", txt, b)
	}

	fh, err = layer.Open("/file.txt")
	if err != nil {
		t.Fatal("could not open file from layer: ", err)
	}
	fh.Close()
}

// #194
func TestUnionFileReaddirEmpty(t *testing.T) {
	osFs := NewOsFs()

	base := NewMemMapFs()
	overlay := NewMemMapFs()
	ufs := &CopyOnWriteFs{base: base, layer: overlay}
	mem := NewMemMapFs()

	// The OS file will return io.EOF on end of directory.
	for _, fs := range []Fs{osFs, ufs, mem} {
		baseDir, err := TempDir(fs, "", "empty-dir")
		if err != nil {
			t.Fatal(err)
		}

		f, err := fs.Open(baseDir)
		if err != nil {
			t.Fatal(err)
		}

		names, err := f.Readdirnames(1)
		if err != io.EOF {
			t.Fatal(err)
		}

		if len(names) != 0 {
			t.Fatal("should be empty")
		}

		f.Close()

		fs.RemoveAll(baseDir)
	}
}

// #197
func TestUnionFileReaddirDuplicateEmpty(t *testing.T) {
	base := NewMemMapFs()
	dir, err := TempDir(base, "", "empty-dir")
	if err != nil {
		t.Fatal(err)
	}

	// Overlay shares same empty directory as base
	overlay := NewMemMapFs()
	err = overlay.Mkdir(dir, 0o700)
	if err != nil {
		t.Fatal(err)
	}

	ufs := &CopyOnWriteFs{base: base, layer: overlay}

	f, err := ufs.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	names, err := f.Readdirnames(0)

	if err == io.EOF {
		t.Errorf("unexpected io.EOF error")
	}

	if len(names) != 0 {
		t.Fatal("should be empty")
	}
}

func TestUnionFileReaddirAskForTooMany(t *testing.T) {
	base := &MemMapFs{}
	overlay := &MemMapFs{}

	const testFiles = 5
	for i := 0; i < testFiles; i++ {
		WriteFile(base, fmt.Sprintf("file%d.txt", i), []byte("afero"), 0o777)
	}

	ufs := &CopyOnWriteFs{base: base, layer: overlay}

	f, err := ufs.Open("/")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	// Read part of all files
	wantNames := 3
	names, err := f.Readdirnames(wantNames)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != wantNames {
		t.Fatalf("got %d names %v, want %d", len(names), names, wantNames)
	}

	// Try to read more files than remaining
	wantNames = testFiles - len(names)
	names, err = f.Readdirnames(wantNames + 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != wantNames {
		t.Fatalf("got %d names %v, want %d", len(names), names, wantNames)
	}

	// End of directory
	_, err = f.Readdirnames(3)
	if err != io.EOF {
		t.Fatal(err)
	}
}
