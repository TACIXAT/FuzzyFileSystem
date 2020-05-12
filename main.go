// Hellofs implements a simple "hello world" file system.
package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"os"
	"syscall"
)

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

// FS implements the hello world file system.
type FFS struct{
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

// Dir implements both Node and Handle for the root directory.
type FFSDir struct{
	Name string
	Children map[string]*FFSFile
	Index uint64
}

func NewFFSDir(name string) *FFSDir {
	ffsd := &FFSDir{
		Name: name,
		Children: make(map[string]*FFSFile),
		Index: lidx.Next(),
	}
	return ffsd 
}

func (ffsd *FFSDir) HashCode() string {
	return fmt.Sprintf("%d", ffsd.Index)
}

func (ffsd *FFSDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = ffsd.Index
	a.Mode = os.ModeDir | 0o644
	return nil
}

func (ffsd *FFSDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if f, ok := ffsd.Children[name]; ok {
		return f, nil
	}
	return nil, syscall.ENOENT
}

func (ffsd *FFSDir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	ffsf := NewFFSFile(req.Name)
	ffsd.Children[req.Name] = ffsf
	return ffsf, ffsf, nil
}

func (ffsd *FFSDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	l := make([]fuse.Dirent, 0)

	for n, c := range ffsd.Children {
		l = append(l, fuse.Dirent{c.Index, fuse.DT_Dir, n})
	}

	return l, nil
}

// func (ffsd *FFSDir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
// 	fmt.Println("mkdir", req.Name)
// 	c :=  NewFFSDir(req.Name)
// 	ffsd.Children[req.Name] = c
// 	return c, nil
// }

// File implements both Node and Handle for the hello file.
type FFSFile struct{
	Name string
	Written bool
	Data []byte
	Index uint64
}

func NewFFSFile(name string) *FFSFile {
	ffsf := &FFSFile{
		Name: name,
		Written: false,
		Data: make([]byte, 0),
		Index: lidx.Next(),
	}

	return ffsf
}

func (ffsf *FFSFile) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	return nil
}

func (ffsf *FFSFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 2
	a.Mode = 0o644
	a.Size = uint64(len(ffsf.Data))
	return nil
}

func (ffsf *FFSFile) ReadAll(ctx context.Context) ([]byte, error) {
	return ffsf.Data, nil
}

func (ffsf *FFSFile)  Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	if ffsf.Written {
		return syscall.EPERM
	}

	end := req.Offset + int64(len(req.Data))
	if int64(len(ffsf.Data)) < end {
		n := make([]byte, end)
		copy(n, ffsf.Data)
		ffsf.Data = n
	}

	start := req.Offset
	copy(ffsf.Data[start:end], req.Data)
	resp.Size = len(req.Data)
	return nil
}

func (ffsf *FFSFile) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	ffsf.Written = true
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("FuzzFileSystem"),
		fuse.Subtype("ffs"),
	)

	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	fmt.Println("Serving...")
	err = fs.Serve(c, NewFFS())
	if err != nil {
		log.Fatal(err)
	}
}