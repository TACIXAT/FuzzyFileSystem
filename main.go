// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This program is the analogon of libfuse's hello.c, a a program that
// exposes a single file "file.txt" in the root directory.
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/hanwen/go-fuse/fs"
	"github.com/hanwen/go-fuse/fuse"
	"log"
	"syscall"
)

type FFSRoot struct {
	fs.Inode
}

func (r *FFSRoot) OnAdd(ctx context.Context) {
	ch := r.NewPersistentInode(
		ctx, &FFSFile{
			Data: []byte("hello world\n"),
			Attr: fuse.Attr{
				Mode: 0644,
			},
		}, fs.StableAttr{Ino: 2})

	r.AddChild("file.txt", ch, false)
}

func (r *FFSRoot) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0755
	return 0
}

func (r *FFSRoot) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (node *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	fmt.Println(name)
	// if n.IsRoot() && name == delDir {
	// }

	var st syscall.Stat_t
	dirName, idx := r.getBranch(&st)
	fmt.Println(dirName)
	// if idx > 0 {
	// 	if errno := n.promote(); errno != 0 {
	// 		return nil, nil, 0, errno
	// 	}
	// 	idx = 0
	return nil, nil, 0, syscall.EPERM
	// }
	// fullPath := filepath.Join(dirName, name)
	// r := n.root()
	// if errno := r.rmMarker(fullPath); errno != 0 && errno != syscall.ENOENT {
	// 	return nil, nil, 0, errno
	// }

	// abs := filepath.Join(n.root().roots[0], fullPath)
	// fd, err := syscall.Creat(abs, mode)
	// if err != nil {
	// 	return nil, nil, 0, err.(syscall.Errno)
	// }

	// if err := syscall.Fstat(fd, &st); err != nil {
	// 	// now what?
	// 	syscall.Close(fd)
	// 	syscall.Unlink(abs)
	// 	return nil, nil, 0, err.(syscall.Errno)
	// }

	// ch := n.NewInode(ctx, &unionFSNode{}, fs.StableAttr{Mode: st.Mode, Ino: st.Ino})
	// out.FromStat(&st)

	// return ch, fs.NewLoopbackFile(fd), 0, 0
}

var _ = (fs.NodeGetattrer)((*FFSRoot)(nil))
var _ = (fs.NodeOnAdder)((*FFSRoot)(nil))
var _ = (fs.NodeCreater)((*FFSRoot)(nil))

func main() {
	debug := flag.Bool("debug", false, "print debug data")
	flag.Parse()
	if len(flag.Args()) < 1 {
		log.Fatal("Usage:\n  main /some/mount/point")
	}

	opts := &fs.Options{}
	opts.Debug = *debug
	opts.Options = []string{
		"rw",
	}
	server, err := fs.Mount(flag.Arg(0), &FFSRoot{}, opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}

	fmt.Println("Serving...")
	server.Wait()
}