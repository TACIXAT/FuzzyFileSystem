# FFS

A format-aware file mutator implemented as an in-memory file system. With a goal of being good at storing a lot of mostly similar copies of files.

![DEMO](demo.gif)

## Status

This is a brand new project. Do not expect the interface to be very stable!!!

### Goals

* Guidable fuzzing
  - Touch a mutation to use it as a seed
  - Remove uninteresting cases
* Format aware fuzzing
  - Target or ignore specfic regions
  - Intelligent mutations based on type
* Light memory footprint
* Serializable and sharable
* Integrate with AFL++
* Generate new files via touch

## Practical

### Install

Requires [Golang](https://golang.org/dl/). Install fuse.

```bash
git clone git@github.com:tacixat/FuzzyFileSystem
cd FuzzyFileSystem
go get # ?? idfk I'll look into this
```

### Run

```bash
go run *.go -mp /mnt/ffs
```

### Usage

```bash
cp file.ext /mnt/ffs/
ls -v /mnt/ffs/file.ext/
cat /mnt/ffs/file.ext/1
touch /mnt/ffs/file.ext/mutate
cat /mnt/ffs/file.ext/11
```

### Cleanup

```bash
umount /mnt/ffs
```

## About

### Tech

The project is built on top of Fuse and the [bazil/fuse](https://github.com/bazil/fuse) Go library.

### License

None for now. Only steel-hearted pirates may use it.