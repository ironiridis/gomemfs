package gomemfs

import (
	"bytes"
	"io"
	"io/fs"
)

type File struct {
	r *bytes.Reader
	k *key
}

// Close releases the bytes.Reader for this object. It
// implements [fs.File].
func (f *File) Close() error {
	f.r = nil
	return nil
}

// Stat implements [fs.File].
func (f *File) Stat() (fs.FileInfo, error) {
	return &FileStat{k: f.k}, nil
}

// Read implements [fs.File].
func (f File) Read(b []byte) (int, error) {
	if f.r == nil {
		return 0, fs.ErrClosed
	}
	return f.r.Read(b)
}

// ReadAt implements [io.ReaderAt].
func (f File) ReadAt(b []byte, off int64) (int, error) {
	if f.r == nil {
		return 0, fs.ErrClosed
	}
	return f.r.ReadAt(b, off)
}

// ReadByte implements [io.ByteScanner].
func (f File) ReadByte() (byte, error) {
	if f.r == nil {
		return 0, fs.ErrClosed
	}
	return f.r.ReadByte()
}

// UnreadByte implements [io.ByteScanner].
func (f File) UnreadByte() error {
	if f.r == nil {
		return fs.ErrClosed
	}
	return f.r.UnreadByte()
}

// ReadRune implements [io.RuneScanner].
func (f File) ReadRune() (rune, int, error) {
	if f.r == nil {
		return 0, 0, fs.ErrClosed
	}
	return f.r.ReadRune()
}

// UnreadRune implements [io.RuneScanner].
func (f File) UnreadRune() error {
	if f.r == nil {
		return fs.ErrClosed
	}
	return f.r.UnreadRune()
}

// Seek implements [io.Seeker].
func (f File) Seek(offset int64, whence int) (int64, error) {
	if f.r == nil {
		return 0, fs.ErrClosed
	}
	return f.r.Seek(offset, whence)
}

// WriteTo implements [io.WriterTo].
func (f File) WriteTo(w io.Writer) (n int64, err error) {
	if f.r == nil {
		return 0, fs.ErrClosed
	}
	return f.r.WriteTo(w)
}
