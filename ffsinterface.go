/* TACIXAT 2020 */
package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"context"
	"sync"
	"syscall"
)

type FFSIReadHandler func() ([]byte, error)
type FFSIAttrHandler func(*fuse.Attr) error
type FFSIWriteHandler func(*fuse.WriteRequest, *fuse.WriteResponse) error
type FFSISetAttrHandler func(*fuse.SetattrRequest, *fuse.SetattrResponse) error

// Mutation
type FFSInterface struct {
	Name           string
	Index          uint64
	ReadHandler    FFSIReadHandler
	AttrHandler    FFSIAttrHandler
	WriteHandler   FFSIWriteHandler
	SetAttrHandler FFSISetAttrHandler
	Mutex          *sync.Mutex
}

func NewFFSInterface(name string) *FFSInterface {
	return &FFSInterface{
		Name:  name,
		Index: lidx.Next(),
		Mutex: new(sync.Mutex),
	}
}

func (ffsi *FFSInterface) ReadAll(ctx context.Context) ([]byte, error) {
	if ffsi.ReadHandler != nil {
		return ffsi.ReadHandler()
	}

	return nil, syscall.ENOSYS
}

func (ffsi *FFSInterface) Attr(ctx context.Context, a *fuse.Attr) error {
	if ffsi.AttrHandler != nil {
		return ffsi.AttrHandler(a)
	}

	return syscall.ENOSYS
}

func (ffsi *FFSInterface) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	return nil
}

func (ffsi *FFSInterface) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	if ffsi.SetAttrHandler != nil {
		return ffsi.SetAttrHandler(req, resp)
	}

	return syscall.ENOSYS
}

func (ffsi *FFSInterface) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return ffsi, nil
}

func (ffsi *FFSInterface) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	ffsi.Mutex.Lock()
	defer ffsi.Mutex.Unlock()
	if ffsi.WriteHandler != nil {
		return ffsi.WriteHandler(req, resp)
	}

	return syscall.ENOSYS
}

func (ffsi *FFSInterface) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	return nil
}
