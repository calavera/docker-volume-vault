package main

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/hashicorp/vault/api"
	"github.com/square/keywhiz-fs"
	"golang.org/x/sys/unix"
)

const (
	EISDIR = fuse.Status(unix.EISDIR)
)

type fs struct {
	pathfs.FileSystem
	client *api.Client
	owner  keywhizfs.Ownership
}

// NewKeywhizFs readies a KeywhizFs struct and its parent filesystem objects.
func NewFs(client *api.Client, owner keywhizfs.Ownership) (*fs, nodefs.Node) {
	defaultfs := pathfs.NewDefaultFileSystem()            // Returns ENOSYS by default
	readonlyfs := pathfs.NewReadonlyFileSystem(defaultfs) // R/W calls return EPERM

	kwfs := &fs{readonlyfs, client, owner}
	nfs := pathfs.NewPathNodeFs(kwfs, nil)
	nfs.SetDebug(true)
	return kwfs, nfs.Root()
}

// GetAttr is a FUSE function which tells FUSE which files and directories exist.
func (f *fs) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	var attr *fuse.Attr
	switch {
	case name == "": // Base directory
		attr = f.directoryAttr(1, 0755)
	case name == "secret":
		attr = f.directoryAttr(1, 0755)
	default:
		s, err := f.client.Logical().Read(name)
		if err != nil {
			return nil, fuse.ENOENT
		}

		if s == nil || s.Data == nil {
			return nil, fuse.ENOENT
		}

		attr = f.secretAttr(s.Data["value"].(string))
	}

	if attr != nil {
		return attr, fuse.OK
	}
	return nil, fuse.ENOENT
}

// Open is a FUSE function where an in-memory open file struct is constructed.
func (f *fs) Open(name string, flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	var file nodefs.File
	switch {
	case name == "":
		return nil, EISDIR
	case name == "secret":
		return nil, EISDIR
	default:
		s, err := f.client.Logical().Read(name)
		if err != nil {
			return nil, fuse.ENOENT
		}

		if s == nil || s.Data == nil {
			return nil, fuse.ENOENT
		}

		file = nodefs.NewDataFile([]byte(s.Data["value"].(string)))
	}

	if file != nil {
		file = nodefs.NewReadOnlyFile(file)
		return file, fuse.OK
	}
	return nil, fuse.ENOENT
}

// OpenDir is a FUSE function called when performing a directory listing.
func (f *fs) OpenDir(name string, context *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	var entries []fuse.DirEntry
	return entries, fuse.OK
}

// Unlink is a FUSE function called when an object is deleted.
func (f *fs) Unlink(name string, context *fuse.Context) fuse.Status {
	return fuse.EACCES
}

// secretAttr constructs a fuse.Attr based on a given Secret.
func (f *fs) secretAttr(s string) *fuse.Attr {
	size := uint64(len(s))
	attr := &fuse.Attr{
		Size: size,
		Mode: 0444 | unix.S_IFREG,
	}

	return attr
}

// directoryAttr constructs a generic directory fuse.Attr with the given parameters.
func (f *fs) directoryAttr(subdirCount, mode uint32) *fuse.Attr {
	attr := &fuse.Attr{
		Mode: fuse.S_IFDIR | mode,
	}
	return attr
}
