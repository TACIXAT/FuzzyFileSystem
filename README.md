# FFS

The goal of this project is to create a format-aware file system that is good at storing a lot of mostly similar copies of files.

## Practical

### Install

Requires [Golang](https://golang.org/dl/).

```bash
git clone git@github.com:tacixat/FuzzyFileSystem
cd FuzzyFileSystem
go get # ?? idfk I'll look into this
```

### Run

```bash
go run main.go /mnt/point
```

### Usage

```bash
cp file.ext /mnt/point/
cat /mnt/point/file.ext/1
```

## About

### Tech

The project is built on top of Fuse and the [bazil/fuse](https://github.com/bazil/fuse) Go library.

### Status

This is a brand new project. Do not expect the interface to be very stable!!!

## Goals

* Guidable fuzzing
  - Touch a mutation to use it as a seed
  - Remove uninteresting cases
* Format aware fuzzing
  - Target or ignore specfic fields
  - Intelligent mutations based on type
* Light memory footprint
* Serializable and sharable