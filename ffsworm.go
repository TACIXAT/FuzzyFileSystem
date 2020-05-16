/* Copyright TACIXAT 2020 */
package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"syscall"
)

func MutationInterface(ffsw *FFSWorm) *FFSInterface {
	ffsiMutate := NewFFSInterface("mutate")
	ffsiMutate.AttrHandler = func(a *fuse.Attr) error {
		a.Valid = 0
		a.Inode = ffsiMutate.Index
		a.Mode = 0o444
		out, _ := getInfo()
		a.Size = uint64(len(out))
		return nil
	}

	ffsiMutate.SetAttrHandler = func(req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
		if req.Valid.MtimeNow() {
			ffsw.Mutex.Lock()
			defer ffsw.Mutex.Unlock()
			ffsw.Mutate()
		}
		return nil
	}

	return ffsiMutate
}

// Write once read many file-directory
type FFSWorm struct {
	Name       string
	Written    bool
	Data       []byte
	Index      uint64
	Interfaces map[string]*FFSInterface
	Children   map[string]*FFSFile
	Flips      map[string]uint64
	Mutex      *sync.Mutex
}

func NewFFSWorm(name string) *FFSWorm {
	ffsw := &FFSWorm{
		Name:       name,
		Written:    false,
		Data:       make([]byte, 0),
		Index:      lidx.Next(),
		Interfaces: make(map[string]*FFSInterface),
		Children:   make(map[string]*FFSFile),
		Flips:      make(map[string]uint64),
		Mutex:      new(sync.Mutex),
	}

	ffsw.Children["0"] = NewFFSFile("0", ffsw)

	ffsw.Interfaces["mutate"] = MutationInterface(ffsw)

	return ffsw
}

func (ffsw *FFSWorm) Mutate() {
	bitc := uint64(len(ffsw.Data) * 8)
	start := uint(len(ffsw.Children))
	for i := start; i < start+*batchSize; i++ {
		name := fmt.Sprintf("%d", i)
		ffsw.Children[name] = NewFFSFile(name, ffsw)
		ffsw.Flips[name] = rand.Uint64() % bitc
	}
}

func (ffsw *FFSWorm) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	return nil
}

func (ffsw *FFSWorm) Attr(ctx context.Context, a *fuse.Attr) error {
	ffsw.Mutex.Lock()
	defer ffsw.Mutex.Unlock()

	a.Valid = 0
	a.Inode = ffsw.Index
	if ffsw.Written {
		a.Mode = os.ModeDir | 0o444
	} else {
		a.Mode = 0o644
	}

	a.Size = uint64(len(ffsw.Data))
	return nil
}

func (ffsw *FFSWorm) Lookup(ctx context.Context, name string) (fs.Node, error) {
	ffsw.Mutex.Lock()
	defer ffsw.Mutex.Unlock()

	if f, ok := ffsw.Children[name]; ok {
		return f, nil
	}

	if f, ok := ffsw.Interfaces[name]; ok {
		return f, nil
	}

	return nil, syscall.ENOENT
}

func (ffsw *FFSWorm) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	ffsw.Mutex.Lock()
	defer ffsw.Mutex.Unlock()

	l := make([]fuse.Dirent, 0)

	for n, i := range ffsw.Interfaces {
		l = append(l, fuse.Dirent{i.Index, fuse.DT_File, n})
	}

	for n, c := range ffsw.Children {
		l = append(l, fuse.Dirent{c.Index, fuse.DT_File, n})
	}

	return l, nil
}

func (ffsw *FFSWorm) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	ffsw.Mutex.Lock()
	defer ffsw.Mutex.Unlock()

	if ffsw.Written {
		return syscall.EPERM
	}
	ffsw.Written = true

	end := req.Offset + int64(len(req.Data))
	if int64(len(ffsw.Data)) < end {
		n := make([]byte, end)
		copy(n, ffsw.Data)
		ffsw.Data = n
	}

	start := req.Offset
	copy(ffsw.Data[start:end], req.Data)
	resp.Size = len(req.Data)

	ffsw.Mutate()

	return nil
}

func (ffsw *FFSWorm) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	return nil
}
