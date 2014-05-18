package ar_test

import (
	"github.com/larzconwell/ar"
	"io"
	"os"
)

func ExampleWriter() {
  out, err := os.Create("libz.a")
  if err != nil {
    panic(err)
  }
  defer out.Close()
  arWriter := ar.NewWriter(out)

  object, err := os.Open("hasher.o")
  if err != nil {
    panic(err)
  }
  defer object.Close()

  stat, err := object.Stat()
  if err != nil {
    panic(err)
  }

  header := ar.FileInfoHeader(stat)
  err = arWriter.WriteHeader(header)
  if err != nil {
    panic(err)
  }

  _, err = io.Copy(arWriter, object)
  if err != nil {
    panic(err)
  }

  err = arWriter.Close()
  if err != nil {
    panic(err)
  }
}

func ExampleReader() {
	in, err := os.Open("libz.a")
	if err != nil {
		panic(err)
	}
	defer in.Close()
	arReader := ar.NewReader(in)

	for {
		header, err := arReader.Next()
		if err != nil {
			panic(err)
		}
		if header == nil {
			break
		}

		out, err := os.Create(header.Name)
		if err != nil {
			panic(err)
		}

		_, err = io.Copy(out, arReader)
		if err != nil {
			panic(err)
		}

		err = out.Close()
		if err != nil {
			panic(err)
		}
	}
}
