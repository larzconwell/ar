package ar

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

var (
	ErrHeader       = errors.New("ar: invalid ar header")
	ErrStringsEntry = errors.New("ar: entry name not in strings table")
)

// Reader provides sequential access to an ar archive. The Next method
// advances to the next file entry, which afterwards can be treated as an
// io.Reader.
type Reader struct {
	reader  io.Reader
	strings map[int64]string // Contains the GNU strings table(key=offset).
	ur      int64            // Unread bytes for the current entry.
	pad     bool             // If the entry contains the padding byte.
	magic   bool             // Indicates if magic number has been read.
}

// NewReader creates a Reader reading from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{reader: r, strings: make(map[int64]string)}
}

// Next advances to the next file entry. A nil, nil return indicates there
// are no entries left to read.
func (arr *Reader) Next() (*Header, error) {
	var err error

	if !arr.magic {
		err = arr.readMagic()
		if err != nil {
			return nil, err
		}
	}

	err = arr.skipUnread()
	if err != nil {
		return nil, err
	}

	header := new(Header)
	hdr := make([]byte, 60)

	_, err = io.ReadFull(arr.reader, hdr)
	if err != nil {
		if err == io.EOF {
			err = nil
		}

		return nil, err
	}

	nameField := arr.trimPad(hdr[:16])
	timeField := arr.trimPad(hdr[16:28])
	uidField := arr.trimPad(hdr[28:34])
	gidField := arr.trimPad(hdr[34:40])
	modeField := arr.trimPad(hdr[40:48])
	sizeField := arr.trimPad(hdr[48:58])
	trailerField := arr.trimPad(hdr[58:60])
	if trailerField != "`\n" {
		return nil, ErrHeader
	}

	// Convert timestamp.
	timeInt, err := strconv.ParseInt(timeField, 10, 64)
	if err != nil && nameField != "//" {
		return nil, err
	}
	header.ModTime = time.Unix(timeInt, 0)

	// Convert uid/gid.
	uidInt, err := strconv.Atoi(uidField)
	if err != nil && nameField != "//" {
		return nil, err
	}
	gidInt, err := strconv.Atoi(gidField)
	if err != nil && nameField != "//" {
		return nil, err
	}
	header.Uid = uidInt
	header.Gid = gidInt

	// Convert mode.
	modeInt, err := strconv.ParseInt(modeField, 8, 64)
	if err != nil && nameField != "//" {
		return nil, err
	}
	header.Mode = modeInt

	// Detect if we need to parse the extended file names and for which format.
	extendedFormat := ""
	nameSize := int64(-1)
	if len(nameField) > 3 && nameField[:3] == "#1/" {
		extendedFormat = "bsd"
		nameSize, err = strconv.ParseInt(nameField[3:], 10, 64)
		if err != nil {
			return nil, err
		}
	}
	if len(nameField) > 1 && nameField[0] == '/' && nameField != "//" {
		extendedFormat = "gnu"
		nameSize, err = strconv.ParseInt(nameField[1:], 10, 64)
		if err != nil {
			return nil, err
		}
	}

	// Convert and retrieve the entry size.
	sizeInt, err := strconv.ParseInt(sizeField, 10, 64)
	if err != nil {
		return nil, err
	}
	header.Size = sizeInt
	if extendedFormat == "bsd" {
		header.Size -= nameSize
	}

	// Retrieve the name.
	header.Name = nameField
	if extendedFormat == "bsd" {
		name := make([]byte, nameSize)
		_, err = io.ReadFull(arr.reader, name)
		if err != nil {
			return nil, err
		}

		header.Name = arr.trimPad(name)
	}

	if extendedFormat == "gnu" {
		name, ok := arr.strings[nameSize]
		if !ok {
			return nil, ErrStringsEntry
		}

		header.Name = name
	}

	// Set unread and padding.
	arr.ur = header.Size
	if header.Size%2 == 0 {
		arr.pad = false
	} else {
		arr.pad = true
	}

	// Parse and store the strings table.
	if header.Name == "//" {
		err = arr.parseStringsTable(header)
		if err != nil {
			return nil, err
		}

		return arr.Next()
	}

	// Skip symbols table.
	if header.Name == "/" || strings.Contains(header.Name, "__.SYMDEF") ||
		header.Name == "__.PKGDEF" || header.Name == "__.GOSYMDEF" {
		return arr.Next()
	}

	// Clean up GNU name.
	if header.Name[len(header.Name)-1] == '/' {
		header.Name = header.Name[:len(header.Name)-1]
	}

	return header, nil
}

// Read reads from the current entry. It returns 0, io.EOF when the end is
// reached until Next is called.
func (arr *Reader) Read(b []byte) (int, error) {
	if arr.ur == 0 {
		return 0, io.EOF
	}

	if int64(len(b)) > arr.ur {
		b = b[:arr.ur]
	}

	n, err := arr.reader.Read(b)
	arr.ur -= int64(n)

	if err == io.EOF && arr.ur > 0 {
		err = io.ErrUnexpectedEOF
	}

	return n, err
}

// skipUnread skips unread bytes and any padding.
func (arr *Reader) skipUnread() error {
	unread := arr.ur
	if arr.pad {
		unread++
	}

	_, err := io.CopyN(ioutil.Discard, arr.reader, unread)
	return err
}

// readMagic reads the magic number.
func (arr *Reader) readMagic() error {
	magic := make([]byte, 8)

	_, err := io.ReadFull(arr.reader, magic)
	if err != nil {
		return err
	}

	if string(magic) != "!<arch>\n" {
		return ErrHeader
	}

	arr.magic = true
	return nil
}

// trimPad trims field padding.
func (arr *Reader) trimPad(field []byte) string {
	return string(bytes.TrimRight(field, " \u0000"))
}

// parseStringsTable gets the GNU strings table from a file entry.
func (arr *Reader) parseStringsTable(header *Header) error {
	strings := make([]byte, header.Size)
	_, err := io.ReadFull(arr, strings)
	if err != nil {
		return err
	}
	offset := 0
	name := make([]byte, 0)

	for i, c := range strings {
		if c == '\n' {
			arr.strings[int64(offset)] = string(bytes.TrimRight(name, "/"))
			name = make([]byte, 0)
			offset = i + 1
			continue
		}

		name = append(name, c)
	}

	return nil
}
