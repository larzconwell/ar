package ar

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var (
	gnuHeader *Header
	bsdHeader *Header
)

func init() {
	err := os.MkdirAll(filepath.Join("testdata", "out"), os.ModePerm|os.ModeDir)
	if err != nil {
		panic(err)
	}
}

func TestGNURead(t *testing.T) {
	in, err := os.Open(filepath.Join("testdata", "gnu_test.a"))
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	arReader := NewReader(in)

	// Get first header.
	header, err := arReader.Next()
	gnuHeader = header
	if err != nil {
		t.Fatal(err)
	}
	if header == nil {
		t.Error("Reader should find at least one entry.")
	}

	if header.Name != "exit.o" {
		t.Error("Header name isn't what it should be.")
	}

	out, err := os.OpenFile(filepath.Join("testdata", "out", "gnu_"+header.Name),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	_, err = io.Copy(out, arReader)
	if err != nil {
		t.Fatal(err)
	}

	// Attempt another header.
	header, err = arReader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if header != nil {
		t.Error("Reader should only get one entry.")
	}
}

func TestGNUVerify(t *testing.T) {
	info, err := os.Stat(filepath.Join("testdata", "out", "gnu_exit.o"))
	if err != nil {
		t.Fatal(err)
	}

	if info.Size() != gnuHeader.Size {
		t.Error("Info size doesn't match header.")
	}
}

func TestGNUInvalidStrings(t *testing.T) {
	in, err := os.Open(filepath.Join("testdata", "gnu_invalid_strings.a"))
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	arReader := NewReader(in)

	_, err = arReader.Next()
	if err != nil && err != ErrStringsEntry {
		t.Fatal(err)
	}

	if err == nil {
		t.Error("Next should have returned ErrHeader but didn't.")
	}
}

func TestBSDRead(t *testing.T) {
	in, err := os.Open(filepath.Join("testdata", "bsd_test.a"))
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	arReader := NewReader(in)

	// Get first header.
	header, err := arReader.Next()
	bsdHeader = header
	if err != nil {
		t.Fatal(err)
	}
	if header == nil {
		t.Error("Reader should find at least one entry.")
	}

	if header.Name != "exit.o" {
		t.Error("Header name isn't what it should be.")
	}

	out, err := os.OpenFile(filepath.Join("testdata", "out", "bsd_"+header.Name),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	_, err = io.Copy(out, arReader)
	if err != nil {
		t.Fatal(err)
	}

	// Attempt another header.
	header, err = arReader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if header != nil {
		t.Error("Reader should only get one entry.")
	}
}

func TestBSDVerify(t *testing.T) {
	info, err := os.Stat(filepath.Join("testdata", "out", "bsd_exit.o"))
	if err != nil {
		t.Fatal(err)
	}

	if info.Size() != bsdHeader.Size {
		t.Error("Info size doesn't match header.")
	}
}

func TestInvalidSize(t *testing.T) {
	in, err := os.Open(filepath.Join("testdata", "invalid_size.a"))
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	arReader := NewReader(in)

	header, err := arReader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if header == nil {
		t.Error("Reader should find at least one entry.")
	}

	_, err = io.Copy(ioutil.Discard, arReader)
	if err != nil && err != io.ErrUnexpectedEOF {
		t.Fatal(err)
	}

	if err == nil {
		t.Error("Read should return ErrUnexpectedEOF if the size isn't correct.")
	}
}

func TestInvalidFormat(t *testing.T) {
	in, err := os.Open(filepath.Join("testdata", "out", "gnu_exit.o"))
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	arReader := NewReader(in)

	_, err = arReader.Next()
	if err != nil && err != ErrHeader {
		t.Fatal(err)
	}

	if err == nil {
		t.Error("Next should have returned ErrHeader but didn't.")
	}
}

func TestInvalidHeader(t *testing.T) {
	in, err := os.Open(filepath.Join("testdata", "invalid_header.a"))
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	arReader := NewReader(in)

	_, err = arReader.Next()
	if err != nil && err != ErrHeader {
		t.Fatal(err)
	}

	if err == nil {
		t.Error("Next should have returned ErrHeader but didn't.")
	}
}
