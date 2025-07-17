package gomemfs

import (
	"time"
)

// A Fulfiller is a callback that receives a normalized path string and tries
// to obtain the byte contents for that path.
type Fulfiller func(path string) (content []byte, modtime *time.Time, expire *time.Time, err error)

func init() {

}
