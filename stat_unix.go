// +build linux darwin freebsd openbsd netbsd

package ar

import (
	"os"
	"syscall"
)

// statHeader fills header with the uid/gid from info.
func statHeader(info os.FileInfo, header *Header) {
	sys, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}

	header.Uid = int(sys.Uid)
	header.Gid = int(sys.Gid)
}
