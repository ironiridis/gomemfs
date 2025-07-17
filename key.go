package gomemfs

import (
	"bytes"
	"time"
)

type key struct {
	bytes []byte
	name  string
	fs    *FS

	// modtime may be returned by the Fulfiller, if eg the content
	// is stored on-disk. Otherwise modtime is set to the time that
	// the Fulfiller was run.
	modtime time.Time

	// expire may be nil if the object never expires.
	expire *time.Time
}

func (k *key) open() *File {
	return &File{r: bytes.NewReader(k.bytes), k: k}
}
