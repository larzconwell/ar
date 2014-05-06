package ar_test

import (
	"github.com/larzconwell/ar"
	"io"
	"os"
)

func Example() {
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
