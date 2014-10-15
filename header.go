package ar

import (
	"os"
	"path"
	"time"
)

// Header represents a single file header in an ar archive. Some fields
// may not be populated.
type Header struct {
	Name    string    // Name of file.
	ModTime time.Time // Modification time.
	Uid     int       // User id of owner.
	Gid     int       // Group id of owner.
	Mode    int64     // Permission and mode bits.
	Size    int64     // Length in bytes.
}

// FileInfoHeader creates a populated Header from info. Because os.FileInfo's
// Name method returns only the base name of the file it describes, it may be
// necessary to modify the Name field of the returned header to provide the
// full path name of the file.
func FileInfoHeader(info os.FileInfo) *Header {
	header := &Header{
		Name:    info.Name(),
		ModTime: info.ModTime(),
		Mode:    int64(info.Mode()),
		Size:    info.Size(),
	}

	statHeader(info, header)
	return header
}

// FileInfo returns a os.FileInfo for the Header.
func (header *Header) FileInfo() os.FileInfo {
	return &fileInfoHeader{header}
}

// fileInfoHeader implements os.FileInfo.
type fileInfoHeader struct {
	header *Header
}

func (fi *fileInfoHeader) Name() string       { return path.Base(fi.header.Name) }
func (fi *fileInfoHeader) Size() int64        { return fi.header.Size }
func (fi *fileInfoHeader) ModTime() time.Time { return fi.header.ModTime }
func (fi *fileInfoHeader) IsDir() bool        { return fi.Mode().IsDir() }
func (fi *fileInfoHeader) Sys() interface{}   { return fi.header }

func (fi *fileInfoHeader) Mode() os.FileMode {
	return os.FileMode(fi.header.Mode)
}
