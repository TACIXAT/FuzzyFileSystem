/* TACIXAT 2020 */
package main

import (
	"bazil.org/fuse/fs"
	"sync"
)

// Root node
type FFS struct {
	Dir *FFSDir
}

func NewFFS() FFS {
	return FFS{
		Dir: NewFFSDir(""),
	}
}

func (ffs FFS) Root() (fs.Node, error) {
	return ffs.Dir, nil
}

// We can probably just set Inode to 0 and have it auto generate
type LockingIndex struct {
	Index uint64
	Mutex *sync.Mutex
}

var lidx *LockingIndex = &LockingIndex{
	Index: 2,
	Mutex: new(sync.Mutex),
}

func (lidx *LockingIndex) Next() uint64 {
	lidx.Mutex.Lock()
	defer lidx.Mutex.Unlock()
	next := lidx.Index
	lidx.Index += 1
	return next
}
