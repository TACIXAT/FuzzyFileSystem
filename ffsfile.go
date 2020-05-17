/* TACIXAT 2020 */
package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"context"
	// "fmt"
)

// Mutant
type FFSFile struct {
	Name       string
	Worm       *FFSWorm
	Index      uint64
	Underlying uint
}

func NewFFSFile(name string, worm *FFSWorm) *FFSFile {
	return &FFSFile{
		Name:       name,
		Worm:       worm,
		Index:      lidx.Next(),
		Underlying: worm.Current,
	}
}

func (ffsf *FFSFile) getBytes() []byte {
	bitoff, ok := ffsf.Worm.Flips[ffsf.Name]
	if !ok {
		return ffsf.Worm.Data[ffsf.Underlying]
	}

	// Flip a bit!
	off := bitoff / 8
	bit := bitoff % 8
	sz := len(ffsf.Worm.Data[ffsf.Underlying])
	data := make([]byte, sz, sz)
	copy(data, ffsf.Worm.Data[ffsf.Underlying])
	data[off] = ffsf.Worm.Data[ffsf.Underlying][off] ^ (1 << bit)

	return data
}

func (ffsf *FFSFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Valid = 0
	a.Inode = ffsf.Index
	a.Mode = 0o444
	a.Size = uint64(len(ffsf.Worm.Data[ffsf.Underlying]))
	return nil
}

func (ffsf *FFSFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return ffsf, nil
}

func (ffsf *FFSFile) ReadAll(ctx context.Context) ([]byte, error) {
	return ffsf.getBytes(), nil
}

func (ffsf *FFSFile) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	if req.Valid.MtimeNow() {
		ffsf.Worm.Mutex.Lock()
		defer ffsf.Worm.Mutex.Unlock()
		ffsf.Worm.Data = append(ffsf.Worm.Data, ffsf.getBytes())
		ffsf.Worm.Current++
	}

	return nil
}
