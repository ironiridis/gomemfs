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

	caseInsensitive bool
	statFulfills    bool
}

func New(o ...FSOption) (*FS, error) {
	fs := &FS{
		keys: make(map[string]*key),
	}
	for i := range o {
		if err := o[i].applyTo(fs); err != nil {
			return nil, fmt.Errorf("failed to apply %T: %w", o[i], err)
		}
	}
	return fs, nil
}

// Set applies an FSOption to an existing FS, if possible.
func (d *FS) Set(o FSOption) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return o.applyTo(d)
}

// Len reports the number of keys currently stored in FS.
func (d *FS) Len() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.keys)
}

// FulfillWith adds one or more Fulfiller callbacks to this FS. Fulfillers are
// run in LIFO order.
func (d *FS) FulfillWith(f ...Fulfiller) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.callbacks = append(d.callbacks, f...)
	return nil
}

// Put sets the contents of key name in the FS. If the key already exists, it is
// replaced. The []byte buffer must not be modified after calling Put; if needed
// you may use [bytes.Clone] to create a private copy for Put.
func (d *FS) Put(name string, content []byte, modtime time.Time, expire *time.Time) error {
	n, err := d.normalize(name)
	if err != nil {
		return fmt.Errorf("cannot put key %q: %w", name, err)
	}
	d.mu.Lock()
	k := &key{
		bytes:   content,
		name:    n,
		modtime: modtime,
		expire:  expire,
		fs:      d,
	}
	d.keys[n] = k
	d.mu.Unlock()
	return nil
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

	// we scan in reverse order! the last added callback is called
	// first, until we encounter an error or get non-nil content
	for i := range d.callbacks {
		idx := len(d.callbacks) - (i + 1)
		content, modtime, expire, err = d.callbacks[idx](name)
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

func (d *FS) normalize(name string) (string, error) {
	if d.caseInsensitive {
		name = strings.ToLower(name)
	}
	return strings.TrimPrefix(path.Clean(name), "/"), nil
}

// Open implements [fs.FS].
func (d *FS) Open(name string) (fs.File, error) {
	n, err := d.normalize(name)
	if err != nil {
		return nil, fmt.Errorf("cannot open key %q: %w", name, err)
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
		return nil, fmt.Errorf("cannot retrieve key %q: %w", name, err)
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
		return nil, fmt.Errorf("cannot stat key %q: %w", name, err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	if k := d.lookup(n); k != nil {
		return &FileStat{k: k}, nil
	}

	if !d.statFulfills {
		return nil, fs.ErrNotExist
	}

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
		return fmt.Errorf("cannot expire key %q: %w", name, err)
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
