package gomemfs

import "errors"

// An FSOption represents a value that can be passed to gomemfs.New() or FS.Set to
// modify the behavior of an FS.
type FSOption interface {
	applyTo(*FS) error
}

// CaseInsensitive causes an FS to lowercase all keys before evaluating, retrieving,
// or fulfilling them when set to true. This option can only be set if the FS is
// empty.
type CaseInsensitive bool

func (fso CaseInsensitive) applyTo(fs *FS) error {
	if len(fs.keys) > 0 {
		return errors.New("cannot update case sensitivity with existing keys")
	}
	fs.caseInsensitive = bool(fso)
	return nil
}

// StatFulfills, if true, will cause an FS to fulfill a missing key on a call to
// FS.Stat. By default this is disabled. Usually this is undesirable as the
// content will be discarded.
type StatFulfills bool

func (fso StatFulfills) applyTo(fs *FS) error {
	fs.statFulfills = bool(fso)
	return nil
}

// FUTURE:
// * IncludeFolders
