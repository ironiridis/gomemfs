package gomemfs

import (
	"io/fs"
	"path"
)

type SubFS struct {
	p *FS
	d string
}

func (d *SubFS) Open(name string) (fs.File, error) {
	return d.p.Open(path.Join(d.d, name))
}

func (d *SubFS) ReadFile(name string) ([]byte, error) {
	return d.p.ReadFile(path.Join(d.d, name))
}

func (d *SubFS) Stat(name string) (fs.FileInfo, error) {
	return d.p.Stat(path.Join(d.d, name))
}

func (d *SubFS) Sub(dir string) (fs.FS, error) {
	return &SubFS{p: d.p, d: path.Join(d.d, dir)}, nil
}
