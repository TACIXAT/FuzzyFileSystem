/* TACIXAT 2020 */
package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"context"
	"encoding/json"
	"os"
	"sync"
	"syscall"
)

func getInfo() ([]byte, error) {
	info := struct {
		Seed      int64 `json:"seed"`
		BatchSize uint  `json:"batch_size"`
	}{}
	info.Seed = *seed
	info.BatchSize = *batchSize

	out, err := json.Marshal(info)
	if err != nil {
		return []byte{}, nil
	}

	out = append(out, '\n')

	return out, nil
}

func InfoInterface() *FFSInterface {
	ffsiInfo := NewFFSInterface("info")
	ffsiInfo.ReadHandler = func() ([]byte, error) {
		return getInfo()
	}

	ffsiInfo.AttrHandler = func(a *fuse.Attr) error {
		a.Valid = 0
		a.Inode = ffsiInfo.Index
		a.Mode = 0o444
		out, _ := getInfo()
		a.Size = uint64(len(out))
		return nil
	}

	return ffsiInfo
}

type FFSDir struct {
	Name       string
	Interfaces map[string]*FFSInterface
	Children   map[string]*FFSWorm
	Index      uint64
	Mutex      *sync.Mutex
}

func NewFFSDir(name string) *FFSDir {
	ffsd := &FFSDir{
		Name:       name,
		Interfaces: make(map[string]*FFSInterface),
		Children:   make(map[string]*FFSWorm),
		Index:      lidx.Next(),
		Mutex:      new(sync.Mutex),
	}

	ffsd.Interfaces["info"] = InfoInterface()
	return ffsd
}

func (ffsd *FFSDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Valid = 0
	a.Inode = ffsd.Index
	a.Mode = os.ModeDir | 0o644
	return nil
}

func (ffsd *FFSDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	ffsd.Mutex.Lock()
	defer ffsd.Mutex.Unlock()

	if f, ok := ffsd.Children[name]; ok {
		return f, nil
	}

	if f, ok := ffsd.Interfaces[name]; ok {
		return f, nil
	}

	return nil, syscall.ENOENT
}

func (ffsd *FFSDir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	ffsd.Mutex.Lock()
	defer ffsd.Mutex.Unlock()

	ffsw := NewFFSWorm(req.Name)
	ffsd.Children[req.Name] = ffsw

	resp.EntryValid = 0
	return ffsw, ffsw, nil
}

func (ffsd *FFSDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	ffsd.Mutex.Lock()
	defer ffsd.Mutex.Unlock()

	l := make([]fuse.Dirent, 0)

	for n, c := range ffsd.Children {
		l = append(l, fuse.Dirent{c.Index, fuse.DT_Dir, n})
	}

	for n, i := range ffsd.Interfaces {
		l = append(l, fuse.Dirent{i.Index, fuse.DT_File, n})
	}

	return l, nil
}
