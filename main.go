/* Copyright TACIXAT 2020 */
package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"syscall"
)

var seed *int64
var mountPoint *string

func init() {
	seed = flag.Int64("s", 0, "rand seed")
	mountPoint = flag.String("m", "", "/mnt/point")
}

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

type FFSDir struct {
	Name       string
	Interfaces map[string]*FFSInterface
	Children   map[string]*FFSWorm
	Index      uint64
}

func getInfo() ([]byte, error) {
	info := struct {
		Seed int64 `json:"seed"`
	}{}
	info.Seed = *seed

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
		a.Inode = ffsiInfo.Index
		a.Mode = 0o444
		out, _ := getInfo()
		a.Size = uint64(len(out))
		return nil
	}

	return ffsiInfo
}

func NewFFSDir(name string) *FFSDir {
	ffsd := &FFSDir{
		Name:       name,
		Interfaces: make(map[string]*FFSInterface),
		Children:   make(map[string]*FFSWorm),
		Index:      lidx.Next(),
	}

	ffsd.Interfaces["info"] = InfoInterface()
	return ffsd
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

	if f, ok := ffsd.Interfaces[name]; ok {
		return f, nil
	}

	return nil, syscall.ENOENT
}

func (ffsd *FFSDir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	ffsw := NewFFSWorm(req.Name)
	ffsd.Children[req.Name] = ffsw
	return ffsw, ffsw, nil
}

func (ffsd *FFSDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	l := make([]fuse.Dirent, 0)

	for n, c := range ffsd.Children {
		l = append(l, fuse.Dirent{c.Index, fuse.DT_Dir, n})
	}

	for n, i := range ffsd.Interfaces {
		l = append(l, fuse.Dirent{i.Index, fuse.DT_File, n})
	}

	return l, nil
}

var BatchSize int = 10

// Write once read many file-directory
type FFSWorm struct {
	Name     string
	Written  bool
	Data     []byte
	Index    uint64
	Children map[string]*FFSFile
	Flips    map[string]uint64
}

func NewFFSWorm(name string) *FFSWorm {
	ffsw := &FFSWorm{
		Name:     name,
		Written:  false,
		Data:     make([]byte, 0),
		Index:    lidx.Next(),
		Children: make(map[string]*FFSFile),
		Flips:    make(map[string]uint64),
	}

	ffsw.Children["0"] = NewFFSFile("0", ffsw)

	return ffsw
}

func (ffsw *FFSWorm) Mutate() {
	bitc := uint64(len(ffsw.Data) * 8)
	for i := len(ffsw.Children); i < BatchSize; i++ {
		name := fmt.Sprintf("%d", i)
		ffsw.Children[name] = NewFFSFile(name, ffsw)
		ffsw.Flips[name] = rand.Uint64() % bitc
	}
}

func (ffsw *FFSWorm) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	return nil
}

func (ffsw *FFSWorm) Attr(ctx context.Context, a *fuse.Attr) error {
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
	if f, ok := ffsw.Children[name]; ok {
		return f, nil
	}

	// if f, ok := ffsw.Interfaces[name]; ok {
	// 	return f, nil
	// }

	return nil, syscall.ENOENT
}

func (ffsw *FFSWorm) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	l := make([]fuse.Dirent, 0)

	for n, c := range ffsw.Children {
		l = append(l, fuse.Dirent{c.Index, fuse.DT_File, n})
	}

	return l, nil
}

func (ffsw *FFSWorm) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
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

// Mutation
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
	a.Inode = ffsf.Index
	a.Mode = 0o444
	a.Size = uint64(len(ffsf.Worm.Data))
	return nil
}

func (ffsf *FFSFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return ffsf, nil
}

type FFSIReadHandler func() ([]byte, error)
type FFSIAttrHandler func(a *fuse.Attr) error

// Mutation
type FFSInterface struct {
	Name        string
	Index       uint64
	ReadHandler FFSIReadHandler
	AttrHandler FFSIAttrHandler
}

func NewFFSInterface(name string) *FFSInterface {
	return &FFSInterface{
		Name:  name,
		Index: lidx.Next(),
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

func (ffsi *FFSInterface) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return ffsi, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	rand.Seed(*seed)

	if len(*mountPoint) == 0 {
		usage()
		os.Exit(2)
	}

	c, err := fuse.Mount(
		*mountPoint,
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
