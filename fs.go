package gomemfs

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"
	"time"
)

// An FS refers to a dynamically generated file structure. The file paths are always
// delimited by forward slashes (Linux/macOS/POSIX style). An FS always starts empty
// and the contents are generated on-demand when an object is opened by calling one
// or more callback functions to fulfill generation.
type FS struct {
	mu        sync.Mutex
	keys      map[string]*key
	callbacks []Fulfiller
}

func New() *FS {
	fs := FS{
		keys: make(map[string]*key),
	}
	return &fs
}

func (d *FS) lookup(name string) *key {
	// must be called with fs.mu Locked
	k, ok := d.keys[name]
	if !ok {
		return nil
	}
	if k == nil {
		// we're somehow storing a nil pointer
		delete(d.keys, name)
		return nil
	}
	if k.expire != nil && time.Now().After(*k.expire) {
		// we found a key but it's expired
		delete(d.keys, name)
		return nil
	}
	return k
}

func (d *FS) fulfill(name string) (*key, error) {
	// must be called with fs.mu Locked
	var content []byte
	var modtime *time.Time
	var expire *time.Time
	var err error
	for i := range d.callbacks {
		content, modtime, expire, err = d.callbacks[i](name)
		if err != nil {
			return nil, err
		}
		if content != nil {
			break
		}
	}
	if content == nil {
		return nil, fs.ErrNotExist
	}
	if modtime == nil {
		var n time.Time = time.Now()
		modtime = &n
	}
	k := &key{
		bytes:   content,
		name:    name,
		modtime: *modtime,
		expire:  expire,
		fs:      d,
	}
	// if the Fulfiller returns a zero expire time, do not cache
	if expire != nil && !expire.IsZero() {
		d.keys[name] = k
	}
	return k, nil
}

// TODO: implement case-insensitive
func (d *FS) normalize(name string) (string, error) {
	return strings.TrimPrefix(path.Clean(name), "/"), nil
}

// Open implements [fs.FS].
func (d *FS) Open(name string) (fs.File, error) {
	n, err := d.normalize(name)
	if err != nil {
		return nil, fmt.Errorf("cannot normalize key %q: %w", name, err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	if k := d.lookup(n); k != nil {
		return k.open(), nil
	}

	if k, err := d.fulfill(n); err != nil {
		return nil, err
	} else {
		return k.open(), nil
	}
}

// ReadFile implements [fs.ReadFileFS]. Note that, because ReadFile returns
// a copy of the byte data, this is not a very efficient method; if one is
// eg reading from a ZIP file and then using this to obtain a buffer to send
// over the network, the data will be copied at least 3 times. In that case
// a more efficient route would be getting the File and using [io.WriterTo]
// via [io.Copy] to (paradoxically) reduce intermediate copies.
func (d *FS) ReadFile(name string) ([]byte, error) {
	n, err := d.normalize(name)
	if err != nil {
		return nil, fmt.Errorf("cannot normalize key %q: %w", name, err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	if k := d.lookup(n); k != nil {
		return bytes.Clone(k.bytes), nil
	}

	if k, err := d.fulfill(n); err != nil {
		return nil, err
	} else {
		return bytes.Clone(k.bytes), nil
	}
}

// Stat implements [fs.StatFS].
func (d *FS) Stat(name string) (fs.FileInfo, error) {
	n, err := d.normalize(name)
	if err != nil {
		return nil, fmt.Errorf("cannot normalize key %q: %w", name, err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	if k := d.lookup(n); k != nil {
		return &FileStat{k: k}, nil
	}

	// TODO: fulfill-on-stat should be configurable! eg if a resource
	// is always uncached, permitting Stat may not be reasonable
	if k, err := d.fulfill(n); err != nil {
		return nil, err
	} else {
		return &FileStat{k: k}, nil
	}
}

// Sub implements [fs.SubFS].
func (d *FS) Sub(dir string) (fs.FS, error) {
	return &SubFS{p: d, d: dir}, nil
}

// Expire removes an item from the FS.
func (d *FS) Expire(name string) error {
	n, err := d.normalize(name)
	if err != nil {
		return fmt.Errorf("cannot normalize key %q: %w", name, err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.keys, n)
	return nil
}

// FlushExpired scans all items in the FS and removes any that have
// expired.
func (d *FS) FlushExpired() error {
	e := make(map[string]bool, len(d.keys))
	n := time.Now()
	d.mu.Lock()
	for k, kp := range d.keys {
		if kp != nil && kp.expire != nil && n.After(*kp.expire) {
			e[k] = true
		}
	}
	for k := range e {
		delete(d.keys, k)
	}
	d.mu.Unlock()
	return nil
}
