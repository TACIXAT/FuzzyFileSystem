# FFS

The goal of this project is to create a format-aware file system that is good at storing a lot of mostly similar copies of files.

## About

### Tech

The project is built on top of Fuse and the [bazil/fuse](https://github.com/bazil/fuse) Go library.

### Status

This is a brand new project. Do not expect the interface to be very stable!!!

### Goals

* Guidable fuzzing
  - Touch a mutation to use it as a seed
  - Remove uninteresting cases
* Format aware fuzzing
  - Target or ignore specfic fields
  - Intelligent mutations based on type
* Light memory footprint
* Serializable and sharable