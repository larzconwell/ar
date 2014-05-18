package ar

import (
  "bytes"
  "io"
  "os"
  "path/filepath"
  "testing"
)

var (
  stdHeader *Header
  stringsHeader *Header
)

func TestWrite(t *testing.T) {
  out, err := os.Create(filepath.Join("testdata", "out", "writer_test.a"))
  if err != nil {
    t.Fatal(err)
  }
  defer out.Close()
  arWriter := NewWriter(out)

  in, err := os.Open(filepath.Join("testdata", "exit.o"))
  if err != nil {
    t.Fatal(err)
  }
  defer in.Close()

  stat, err := in.Stat()
  if err != nil {
    t.Fatal(err)
  }
  header := FileInfoHeader(stat)
  stdHeader = header

  err = arWriter.WriteHeader(header)
  if err != nil {
    t.Fatal(err)
  }

  _, err = io.Copy(arWriter, in)
  if err != nil {
    t.Fatal(err)
  }

  err = arWriter.Close()
  if err != nil {
    t.Fatal(err)
  }
}

func TestVerify(t *testing.T) {
  in, err := os.Open(filepath.Join("testdata", "out", "writer_test.a"))
  if err != nil {
    t.Fatal(err)
  }
  defer in.Close()
  arReader := NewReader(in)

  // Get first header.
  header, err := arReader.Next()
  if err != nil {
    t.Fatal(err)
  }
  if header == nil {
    t.Error("Reader should find at least one entry.")
  }

  if header.Name != stdHeader.Name {
    t.Error("Header name isn't what it should be.")
  }

  if header.Size != stdHeader.Size {
    t.Error("Header size isn't what it should be.")
  }
}

func TestLongNameWrite(t *testing.T) {
  out, err := os.Create(filepath.Join("testdata", "out", "writer_test_strings.a"))
  if err != nil {
    t.Fatal(err)
  }
  defer out.Close()
  arWriter := NewWriter(out)

  in, err := os.Open(filepath.Join("testdata", "exit.o"))
  if err != nil {
    t.Fatal(err)
  }
  defer in.Close()

  stat, err := in.Stat()
  if err != nil {
    t.Fatal(err)
  }
  header := FileInfoHeader(stat)
  header.Name = "extremelysuperlongnamesomewhereshouldcreatestringstable.o"
  stringsHeader = header

  err = arWriter.WriteHeader(header)
  if err != nil {
    t.Fatal(err)
  }

  _, err = io.Copy(arWriter, in)
  if err != nil {
    t.Fatal(err)
  }

  err = arWriter.Close()
  if err != nil {
    t.Fatal(err)
  }
}

func TestLongNameVerify(t *testing.T) {
  in, err := os.Open(filepath.Join("testdata", "out", "writer_test_strings.a"))
  if err != nil {
    t.Fatal(err)
  }
  defer in.Close()
  arReader := NewReader(in)

  // Get first header.
  header, err := arReader.Next()
  if err != nil {
    t.Fatal(err)
  }
  if header == nil {
    t.Error("Reader should find at least one entry.")
  }

  if header.Name != stringsHeader.Name {
    t.Error("Header name isn't what it should be.")
  }

  if header.Size != stringsHeader.Size {
    t.Error("Header size isn't what it should be.")
  }
}

func TestWriteAfterClose(t *testing.T) {
  out, err := os.Create(filepath.Join("testdata", "out", "writer_test.a"))
  if err != nil {
    t.Fatal(err)
  }
  defer out.Close()
  arWriter := NewWriter(out)

  in, err := os.Open(filepath.Join("testdata", "exit.o"))
  if err != nil {
    t.Fatal(err)
  }
  defer in.Close()

  stat, err := in.Stat()
  if err != nil {
    t.Fatal(err)
  }
  header := FileInfoHeader(stat)

  err = arWriter.WriteHeader(header)
  if err != nil {
    t.Fatal(err)
  }

  _, err = io.Copy(arWriter, in)
  if err != nil {
    t.Fatal(err)
  }

  err = arWriter.Close()
  if err != nil {
    t.Fatal(err)
  }

  err = arWriter.WriteHeader(header)
  if err != nil && err != ErrWriteAfterClose {
    t.Fatal(err)
  }
  if err == nil {
    t.Error("WriteHeader should've returned ErrWriteAfterClose but didn't.")
  }

  _, err = arWriter.Write([]byte(""))
  if err != nil && err != ErrWriteAfterClose {
    t.Fatal(err)
  }
  if err == nil {
    t.Error("Write should've returned ErrWriteAfterClose but didn't.")
  }
}

func TestWriteTooLong(t *testing.T) {
  out, err := os.Create(filepath.Join("testdata", "out", "writer_test.a"))
  if err != nil {
    t.Fatal(err)
  }
  defer out.Close()
  arWriter := NewWriter(out)

  in, err := os.Open(filepath.Join("testdata", "exit.o"))
  if err != nil {
    t.Fatal(err)
  }
  defer in.Close()

  var buf bytes.Buffer
  _, err = io.Copy(&buf, in)
  if err != nil {
    t.Fatal(err)
  }

  _, err = buf.Write([]byte("sdfsdfsdf"))
  if err != nil {
    t.Fatal(err)
  }

  stat, err := in.Stat()
  if err != nil {
    t.Fatal(err)
  }
  header := FileInfoHeader(stat)

  err = arWriter.WriteHeader(header)
  if err != nil {
    t.Fatal(err)
  }

  _, err = io.Copy(arWriter, &buf)
  if err != nil && err != ErrWriteTooLong {
    t.Fatal(err)
  }
  if err == nil {
    t.Error("Write should've returned ErrWriteTooLong but didn't.")
  }
}

func TestHeaderTooLong(t *testing.T) {
  out, err := os.Create(filepath.Join("testdata", "out", "writer_test.a"))
  if err != nil {
    t.Fatal(err)
  }
  defer out.Close()
  arWriter := NewWriter(out)

  in, err := os.Open(filepath.Join("testdata", "exit.o"))
  if err != nil {
    t.Fatal(err)
  }
  defer in.Close()

  stat, err := in.Stat()
  if err != nil {
    t.Fatal(err)
  }
  header := FileInfoHeader(stat)
  header.Uid = 9999999

  err = arWriter.WriteHeader(header)
  if err != nil && err != ErrHeaderTooLong {
    t.Fatal(err)
  }
  if err == nil {
    t.Error("WriteHeader should've returned ErrHeaderTooLong but didn't.")
  }
}
