/* TACIXAT 2020 */
package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
)

var seed *int64
var mountPoint *string
var batchSize *uint

func init() {
	seed = flag.Int64("s", 0, "rand seed")
	mountPoint = flag.String("mp", "", "/mnt/point")
	batchSize = flag.Uint("bs", 10, "mutate batch size")
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -mp /some/mount/point\n", os.Args[0])
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
