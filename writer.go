package ar

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strconv"
	"time"
)

var (
	ErrWriteAfterClose = errors.New("ar: write after close")
	ErrWriteTooLong    = errors.New("ar: write too long")
	ErrHeaderTooLong   = errors.New("ar: header too long")
)

// entry contains the entry name an the byte offset to its header in the
// file entries buffer.
type entry struct {
	Name   string
	Offset int64
}

// Writer provides sequential writing to an ar archive using the GNU format.
// WriteHeader triggers a new entry to be written, aftwards the writer can be
// used as an io.Writer.
type Writer struct {
	writer  io.Writer
	symbols []*entry      // Contains the list for the GNU symbol table.
	strings *bytes.Buffer // Contains the GNU strings table.
	buf     *bytes.Buffer // Contains standard file entries.
	uw      int64         // Unwritten bytes for the current entry.
	pad     bool          // If the entry should contain the padding byte.
	closed  bool
}

// NewWriter creates a Writer writing to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer:  w,
		symbols: make([]*entry, 0),
		strings: new(bytes.Buffer),
		buf:     new(bytes.Buffer),
	}
}

// Write writes b to the current file entry. It returns ErrWriteTooLong if more
// bytes are being written than the header allows.
func (arw *Writer) Write(b []byte) (int, error) {
	if arw.closed {
		return 0, ErrWriteAfterClose
	}
	overwrite := false

	// Trim so we don't overwrite.
	if int64(len(b)) > arw.uw {
		b = b[:arw.uw]
		overwrite = true
	}

	n, err := arw.buf.Write(b)
	arw.uw -= int64(n)
	if err == nil && overwrite {
		err = ErrWriteTooLong
	}

	return n, err
}

// WriteHeader creates a new file entry for header. Calling after it's closed
// will return ErrWriteAfterClose. ErrHeaderTooLong is returned if the header
// won't fit.
func (arw *Writer) WriteHeader(header *Header) error {
	if arw.closed {
		return ErrWriteAfterClose
	}

	err := arw.fillUnwritten(arw.buf)
	if err != nil {
		return err
	}

	hdr, err := arw.createHeader(true, header)
	if err != nil {
		return err
	}

	_, err = arw.buf.Write(hdr)
	return err
}

// Close closes the ar archive creating the symbol/string tables. All writing
// to the underlying writer is delayed until Close.
func (arw *Writer) Close() error {
	if arw.closed {
		return nil
	}
	arw.closed = true

	err := arw.fillUnwritten(arw.buf)
	if err != nil {
		return err
	}

	// Create strings header, only populated if there's any data in the entry.
	strHeader := make([]byte, 0)
	if arw.strings.Len() > 0 {
		strHeader, err = arw.createHeader(false, &Header{
			Name:    "//",
			ModTime: time.Now(),
			Uid:     0,
			Gid:     0,
			Mode:    0,
			Size:    int64(arw.strings.Len()),
		})
		if err != nil {
			return err
		}
	}

	// Calculate the size of the data before file entries, used to complete
	// the symbol offsets.
	size := int64(68)                         // Magic num + header size.
	size += int64(4 + (4 * len(arw.symbols))) // Size of symbol table.
	for _, entry := range arw.symbols {
		size += int64(len(entry.Name + "\u0000"))
	}
	if (size-68)%2 != 0 {
		size++
	}
	size += int64(len(strHeader) + arw.strings.Len()) // Strings header + table.
	if arw.strings.Len()%2 != 0 {
		size++
	}

	// Create the symbol table.
	var symTable bytes.Buffer
	err = binary.Write(&symTable, binary.BigEndian, int32(len(arw.symbols)))
	if err != nil {
		return err
	}
	for _, entry := range arw.symbols {
		// Also set the final offset.
		entry.Offset += size

		err = binary.Write(&symTable, binary.BigEndian, int32(entry.Offset))
		if err != nil {
			return err
		}
	}
	for _, entry := range arw.symbols {
		_, err = symTable.Write([]byte(entry.Name + "\u0000"))
		if err != nil {
			return err
		}
	}

	// Create symbol header.
	symHeader, err := arw.createHeader(false, &Header{
		Name:    "/",
		ModTime: time.Now(),
		Uid:     0,
		Gid:     0,
		Mode:    0,
		Size:    int64(symTable.Len()),
	})
	if err != nil {
		return err
	}

	_, err = arw.writer.Write([]byte("!<arch>\n"))
	if err != nil {
		return err
	}

	// Write the symbol table.
	_, err = arw.writer.Write(symHeader)
	if err != nil {
		return err
	}
	if symTable.Len()%2 != 0 {
		_, err = symTable.Write([]byte("\n"))
		if err != nil {
			return err
		}
	}
	_, err = io.Copy(arw.writer, &symTable)
	if err != nil {
		return err
	}

	// Write strings table if needed.
	if len(strHeader) > 0 {
		_, err = arw.writer.Write(strHeader)
		if err != nil {
			return err
		}

		if arw.strings.Len()%2 != 0 {
			_, err = arw.strings.Write([]byte("\n"))
			if err != nil {
				return err
			}
		}
		_, err = io.Copy(arw.writer, arw.strings)
		if err != nil {
			return err
		}
	}

	_, err = io.Copy(arw.writer, arw.buf)
	return err
}

// createHeader creates the header entry, if standard / is added to names, and
// strings/symbol tables are written.
func (arw *Writer) createHeader(standard bool, header *Header) ([]byte, error) {
	// Get name and detect if extended.
	offset := ""
	name := toASCII(header.Name)
	if standard {
		name += "/"
	}
	if len(name) > 16 {
		if !standard {
			return nil, ErrHeaderTooLong
		}

		offset = name
		name = "/" + strconv.Itoa(arw.strings.Len())
	}

	// Get modtime, and ensure it fits.
	mod := strconv.FormatInt(header.ModTime.Unix(), 10)
	if len(mod) > 12 {
		return nil, ErrHeaderTooLong
	}

	// Get uid/gid, and ensure they'll fit.
	uid := strconv.Itoa(header.Uid)
	if len(uid) > 6 {
		return nil, ErrHeaderTooLong
	}
	gid := strconv.Itoa(header.Gid)
	if len(gid) > 6 {
		return nil, ErrHeaderTooLong
	}

	// Format the mode, and ensure it fits.
	mode := strconv.FormatInt(header.Mode, 8)
	if standard {
		mode = "100" + mode
	}
	if len(mode) > 8 {
		return nil, ErrHeaderTooLong
	}

	// Get size, and ensure it fits.
	size := strconv.FormatInt(header.Size, 10)
	if len(size) > 10 {
		return nil, ErrHeaderTooLong
	}

	// Set unwritten and padding.
	arw.uw = header.Size
	if header.Size%2 == 0 {
		arw.pad = false
	} else {
		arw.pad = true
	}

	// Write to strings buffer if extended.
	if offset != "" {
		_, err := arw.strings.Write([]byte(offset + "\n"))
		if err != nil {
			return nil, err
		}
	}

	// Add item to symbol table.
	if standard {
		entry := &entry{Name: name, Offset: int64(arw.buf.Len())}
		if offset != "" {
			entry.Name = offset
		}
		arw.symbols = append(arw.symbols, entry)
	}

	// Add content to fields.
	hdr := make([]byte, 60)
	arw.fillField(hdr[:16], name)
	arw.fillField(hdr[16:28], mod)
	arw.fillField(hdr[28:34], uid)
	arw.fillField(hdr[34:40], gid)
	arw.fillField(hdr[40:48], mode)
	arw.fillField(hdr[48:58], size)
	arw.fillField(hdr[58:], "`\n")

	return hdr, nil
}

// fillUnwritten writes any unwritten bytes and writes the padding byte to w.
func (arw *Writer) fillUnwritten(w io.Writer) error {
	fill := make([]byte, arw.uw)
	for i := range fill {
		fill[i] = ' '
	}

	if arw.pad {
		fill = append(fill, '\n')
	}

	_, err := w.Write(fill)
	return err
}

// fillField writes a string and any padding to a byte slice.
func (arw *Writer) fillField(field []byte, contents string) {
	clen := len(contents)

	for i := range field {
		if i >= clen {
			field[i] = ' '
			continue
		}

		field[i] = contents[i]
	}
}

// toASCII strips non ascii characters from s.
func toASCII(s string) string {
	n := make([]rune, 0)

	for _, c := range s {
		if c < 0x80 {
			n = append(n, c)
		}
	}

	return string(n)
}
