/* Copyright TACIXAT 2020 */
package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"syscall"
)

// Write once read many file-directory
type FFSWorm struct {
	Name       string
	Written    bool
	Data       [][]byte
	Current    uint
	Index      uint64
	Interfaces map[string]*FFSInterface
	Children   map[string]*FFSFile
	NextChild  uint
	Mapping    map[int]int
	Flips      map[string]uint64
	Mutex      *sync.Mutex
}

func MutationInterface(ffsw *FFSWorm) *FFSInterface {
	ffsiMutate := NewFFSInterface("mutate")
	ffsiMutate.AttrHandler = func(a *fuse.Attr) error {
		a.Valid = 0
		a.Inode = ffsiMutate.Index
		a.Mode = 0o444
		return nil
	}

	ffsiMutate.SetAttrHandler = func(req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
		if req.Valid.MtimeNow() {
			ffsw.Mutex.Lock()
			defer ffsw.Mutex.Unlock()
			return ffsw.Mutate()
		}
		return nil
	}

	return ffsiMutate
}

type Range struct {
	Offset int `json:"offset"`
	Size   int `json:"size"`
}

type Mask struct {
	Include bool    `json:"include"`
	Ranges  []Range `json:"ranges"`
}

func (ffsw *FFSWorm) include(ranges []Range) error {
	old := ffsw.Mapping
	ffsw.Mapping = make(map[int]int)

	mapping := 0
	for _, r := range ranges {
		offset := r.Offset

		end := offset + r.Size
		if end > len(ffsw.Data[ffsw.Current]) {
			end = len(ffsw.Data[ffsw.Current])
		}

		for i := offset; i < end; i++ {
			if _, ok := ffsw.Mapping[mapping]; ok {
				ffsw.Mapping = old
				return syscall.EPERM
			}

			ffsw.Mapping[mapping] = i
			mapping++
		}
	}

	return nil
}

func (ffsw *FFSWorm) exclude(ranges []Range) error {
	old := ffsw.Mapping
	ffsw.Mapping = make(map[int]int)

	mapping := 0
	// space before
	start := 0
	for _, r := range ranges {
		for i := start; i < r.Offset; i++ {
			ffsw.Mapping[mapping] = i
			mapping++
		}

		start = r.Offset + r.Size
	}

	// space after
	if start < len(ffsw.Data[ffsw.Current]) {
		for i := start; i < len(ffsw.Data[ffsw.Current]); i++ {
			if _, ok := ffsw.Mapping[mapping]; ok {
				ffsw.Mapping = old
				return syscall.EPERM
			}

			ffsw.Mapping[mapping] = i
			mapping++
		}
	}

	return nil
}

func MaskInterface(ffsw *FFSWorm) *FFSInterface {
	ffsiMask := NewFFSInterface("mask")
	ffsiMask.AttrHandler = func(a *fuse.Attr) error {
		a.Valid = 0
		a.Inode = ffsiMask.Index
		a.Mode = 0o644
		// out, _ := getMask()
		// a.Size = uint64(len(out))
		a.Size = 0
		return nil
	}

	ffsiMask.WriteHandler = func(req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
		ffsw.Mutex.Lock()
		defer ffsw.Mutex.Unlock()

		m := Mask{}
		err := json.Unmarshal(req.Data, &m)
		if err != nil {
			return syscall.EPERM
		}

		resp.Size = len(req.Data)

		if len(m.Ranges) == 0 {
			ffsw.Mapping = make(map[int]int)
			return nil
		}

		if m.Include {
			return ffsw.include(m.Ranges)
		} else {
			return ffsw.exclude(m.Ranges)
		}

		return nil
	}

	ffsiMask.SetAttrHandler = func(req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
		return nil
	}

	return ffsiMask
}

func NewFFSWorm(name string) *FFSWorm {
	ffsw := &FFSWorm{
		Name:       name,
		Written:    false,
		Data:       make([][]byte, 1),
		Current:    0,
		Index:      lidx.Next(),
		Interfaces: make(map[string]*FFSInterface),
		Children:   make(map[string]*FFSFile),
		Mapping:    make(map[int]int),
		Flips:      make(map[string]uint64),
		Mutex:      new(sync.Mutex),
	}

	ffsw.Children["0"] = NewFFSFile("0", ffsw)
	ffsw.NextChild = 1

	ffsw.Interfaces["mutate"] = MutationInterface(ffsw)
	ffsw.Interfaces["mask"] = MaskInterface(ffsw)

	return ffsw
}

// Caller must lock
func (ffsw *FFSWorm) Mutate() error {
	d := ffsw.Data[ffsw.Current]
	bitc := uint64(len(d) * 8)
	if len(ffsw.Mapping) > 0 {
		bitc = uint64(len(ffsw.Mapping) * 8)
	}

	if bitc == 0 {
		return syscall.ENOENT
	}

	end := ffsw.NextChild + *batchSize
	for i := ffsw.NextChild; i < end; i++ {
		name := fmt.Sprintf("%d", i)
		ffsw.Children[name] = NewFFSFile(name, ffsw)

		flip := rand.Uint64() % bitc
		if len(ffsw.Mapping) > 0 {
			byte := flip / 8
			bit := flip % 8
			byte = uint64(ffsw.Mapping[int(byte)])
			flip = byte*8 + bit
		}

		ffsw.Flips[name] = flip
	}

	ffsw.NextChild = end

	return nil
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
		a.Mode = os.ModeDir | 0o644
		a.Size = 0
	} else {
		a.Mode = 0o644
		a.Size = uint64(len(ffsw.Data[ffsw.Current]))
	}

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

func (ffsw *FFSWorm) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	if req.Name == "0" {
		return syscall.EPERM
	}

	if _, ok := ffsw.Children[req.Name]; ok {
		delete(ffsw.Children, req.Name)
		delete(ffsw.Flips, req.Name)
		return nil
	}

	if _, ok := ffsw.Interfaces[req.Name]; ok {
		return syscall.EPERM
	}

	return syscall.ENOENT
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

func (ffsw *FFSWorm) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	ffsw.Mutex.Lock()
	defer ffsw.Mutex.Unlock()
	return nil, nil, syscall.EPERM
}

func (ffsw *FFSWorm) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	ffsw.Mutex.Lock()
	defer ffsw.Mutex.Unlock()

	if ffsw.Written {
		return syscall.EPERM
	}

	end := req.Offset + int64(len(req.Data))
	if int64(len(ffsw.Data[ffsw.Current])) < end {
		n := make([]byte, end)
		copy(n, ffsw.Data[ffsw.Current])
		ffsw.Data[ffsw.Current] = n
	}

	start := req.Offset
	copy(ffsw.Data[ffsw.Current][start:end], req.Data)
	resp.Size = len(req.Data)
	return nil
}

func (ffsw *FFSWorm) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	ffsw.Mutex.Lock()
	defer ffsw.Mutex.Unlock()
	if len(ffsw.Data[0]) > 0 {
		ffsw.Written = true
	}
	return nil
}
