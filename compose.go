package gomemfs

import (
	"fmt"
	"io"
	"io/fs"
	"time"
)

// Compose adapts an existing fs.FS implementation to a Fulfiller. If ttl is
// not nil, objects from t will be set to expire at time.Now().Add(ttl).
func Compose(t fs.FS, ttl *time.Duration) Fulfiller {
	return func(path string) ([]byte, *time.Time, *time.Time, error) {
		f, err := t.Open(path)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("cannot open %q in composed %T: %w", path, t, err)
		}
		defer f.Close()

		buf, err := io.ReadAll(f)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("cannot read %q in composed %T: %w", path, t, err)
		}

		var mt time.Time
		if fi, err := f.Stat(); err != nil {
			mt = time.Now()
		} else {
			mt = fi.ModTime()
		}

		if ttl != nil {
			expire := time.Now().Add(*ttl)
			return buf, &mt, &expire, nil
		}

		return buf, &mt, nil, nil
	}
}
