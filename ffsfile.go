/* Copyright TACIXAT 2020 */
package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"context"
)

// Mutant
type FFSFile struct {
	Name  string
	Worm  *FFSWorm
	Index uint64
}

func NewFFSFile(name string, worm *FFSWorm) *FFSFile {
	return &FFSFile{
		Name:  name,
		Worm:  worm,
		Index: lidx.Next(),
	}
}

func (ffsf *FFSFile) ReadAll(ctx context.Context) ([]byte, error) {
	bitoff, ok := ffsf.Worm.Flips[ffsf.Name]
	if !ok {
		return ffsf.Worm.Data, nil
	}

	// Flip a bit!
	off := bitoff / 8
	bit := bitoff % 8
	sz := len(ffsf.Worm.Data)
	data := make([]byte, sz, sz)
	copy(data, ffsf.Worm.Data)
	data[off] = ffsf.Worm.Data[off] ^ (1 << bit)
	return data, nil
}

func (ffsf *FFSFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Valid = 0
	a.Inode = ffsf.Index
	a.Mode = 0o444
	a.Size = uint64(len(ffsf.Worm.Data))
	return nil
}

func (ffsf *FFSFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return ffsf, nil
}
