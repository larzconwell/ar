package ar

import (
	"os"
)

// statHeader is a no-op on Windows.
func statHeader(info os.FileInfo, header *Header) {}
