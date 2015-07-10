package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/calavera/dkvolume"
	"golang.org/x/sys/unix"
)

const (
	id = "_vault"
)

var (
	defaultPath = filepath.Join(dkvolume.DefaultDockerRootDirectory, id)
	root        = flag.String("root", defaultPath, "Docker volumes root directory")
)

func main() {
	var Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] url\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() != 1 {
		Usage()
		os.Exit(1)
	}

	lockMemory()

	d := newDriver(*root, flag.Args()[0])
	h := dkvolume.NewHandler(d)
	fmt.Println(h.ServeUnix("root", "vault"))
}

// Locks memory, preventing memory from being written to disk as swap
func lockMemory() {
	err := unix.Mlockall(unix.MCL_FUTURE | unix.MCL_CURRENT)
	switch err {
	case nil:
	case unix.ENOSYS:
		log.Println("mlockall() not implemented on this system")
	case unix.ENOMEM:
		log.Println("mlockall() failed with ENOMEM")
	default:
		log.Fatalf("Could not perform mlockall and prevent swapping memory: %v", err)
	}
}
