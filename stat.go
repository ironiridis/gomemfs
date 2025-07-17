package gomemfs

import (
	"io/fs"
	"path"
	"time"
)

type FileStat struct {
	k *key
}

func (s FileStat) Name() string {
	return path.Base(s.k.name)
}

func (s FileStat) Size() int64 {
	return int64(len(s.k.bytes))
}

func (s FileStat) Mode() fs.FileMode {
	return fs.FileMode(0) // "regular"
}

func (s FileStat) ModTime() time.Time {
	return s.k.modtime
}

func (s FileStat) IsDir() bool {
	return s.Mode().IsDir()
}

func (s FileStat) Sys() any {
	return nil
}
