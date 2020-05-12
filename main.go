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
	"strings"
)

type FFSRoot struct {
	fs.Inode
}

func (r *FFSRoot) OnAdd(ctx context.Context) {
	// ch := r.NewPersistentInode(
	// 	ctx, &FFSFile{
	// 		Data: []byte("hello world\n"),
	// 		Attr: fuse.Attr{
	// 			Mode: 0644,
	// 		},
	// 	}, fs.StableAttr{Ino: 2})

	// r.AddChild("file.txt", ch, false)
}

func (r *FFSRoot) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0755
	return 0
}

func (r *FFSRoot) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	fmt.Println(name)
	if !r.IsRoot() || strings.Contains(name, "/") {
		return nil, nil, 0, syscall.EPERM
	}

	d := r.NewInode(ctx, &fs.Inode{}, fs.StableAttr{Mode: fuse.S_IFDIR})

	f := d.NewInode(ctx, &FFSFile{
			Data: []byte{},
			Attr: fuse.Attr{
				Mode: 0644,
			},
		}, fs.StableAttr{Ino: 0})

	if ok := d.AddChild("0", f, true); !ok {
		return nil, nil, 0, syscall.EBADF
	}

	if ok := r.AddChild(name+"suffix", d, true); !ok {
		return nil, nil, 0, syscall.ENOTDIR
	} 

	for n, _ := range r.Children() {
		fmt.Println(n)
	}

	return f, nil, 0, 0
}

var _ = (fs.NodeGetattrer)((*FFSRoot)(nil))
var _ = (fs.NodeOnAdder)((*FFSRoot)(nil))
var _ = (fs.NodeCreater)((*FFSRoot)(nil))

func main() {
	debug := flag.Bool("d", false, "print debug data")
	mount := flag.String("m", "", "mountpoint")
	flag.Parse()
	fmt.Println(*debug, *mount)
	if len(*mount) == 0 {
		log.Fatal("Usage:\n  main /some/mount/point")
	}

	opts := &fs.Options{}
	opts.Debug = *debug
	opts.Options = []string{}
	// 	"rw",
	// }
	server, err := fs.Mount(*mount, &FFSRoot{}, opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}

	fmt.Println("Serving...")
	server.Wait()
}