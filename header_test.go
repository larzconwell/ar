package ar

import (
	"os"
	"path"
	"testing"
	"time"
)

func TestFileInfoHeader(t *testing.T) {
	info, err := os.Stat(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}

	header := FileInfoHeader(info)
	if header.Name != info.Name() {
		t.Error("Header name doesn't match file info.")
	}

	if header.ModTime != info.ModTime() {
		t.Error("Header modtime doesn't match file info.")
	}

	if header.Mode != int64(info.Mode()) {
		t.Error("Header mode doesn't match file info.")
	}

	if header.Size != info.Size() {
		t.Error("Header size doesn't match file info.")
	}
}

func TestFileInfo(t *testing.T) {
	now := time.Now()
	header := &Header{
		Name:    "testdata/test.o",
		ModTime: now,
		Mode:    int64(os.ModePerm | os.ModeDir),
		Size:    5,
	}
	info := header.FileInfo()

	if info.Name() != path.Base(header.Name) {
		t.Error("Info name doesn't match header.")
	}

	if info.ModTime() != header.ModTime {
		t.Error("Info modtime doesn't match header.")
	}

	if int64(info.Mode()) != header.Mode {
		t.Error("Header mode doesn't match header.")
	}

	if info.Size() != header.Size {
		t.Error("Size size doesn't match header.")
	}
}
