package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/calavera/dkvolume"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hashicorp/vault/api"
	"github.com/square/keywhiz-fs"
)

type driver struct {
	root   string
	addr   string
	server *fuse.Server
	m      sync.Mutex
}

func newDriver(root, addr string) *driver {
	return &driver{
		root: root,
		addr: addr,
	}
}

func (d *driver) Create(r dkvolume.Request) dkvolume.Response {
	return dkvolume.Response{}
}

func (d *driver) Remove(r dkvolume.Request) dkvolume.Response {
	return dkvolume.Response{}
}

func (d *driver) Path(r dkvolume.Request) dkvolume.Response {
	return dkvolume.Response{Mountpoint: d.mountpoint(r.Name)}
}

func (d *driver) Mount(r dkvolume.Request) dkvolume.Response {
	d.m.Lock()
	defer d.m.Unlock()
	m := d.mountpoint(r.Name)
	log.Printf("Mounting volume %s on %s\n", r.Name, m)

	fi, err := os.Lstat(m)

	if os.IsNotExist(err) {
		if err := os.MkdirAll(m, 0755); err != nil {
			return dkvolume.Response{Err: err.Error()}
		}
	} else if err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	if fi != nil && !fi.IsDir() {
		return dkvolume.Response{Err: fmt.Sprintf("%v already exist and it's not a directory", m)}
	}

	server, err := d.mountServer(m)
	if err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	d.server = server

	return dkvolume.Response{Mountpoint: m}
}

func (d driver) Unmount(r dkvolume.Request) dkvolume.Response {
	d.m.Lock()
	defer d.m.Unlock()
	m := d.mountpoint(r.Name)
	log.Printf("Unmounting volume %s from %s\n", r.Name, m)

	if d.server != nil {
		if err := d.server.Unmount(); err != nil {
			return dkvolume.Response{Err: err.Error()}
		}
	}

	return dkvolume.Response{}
}

func (d *driver) mountpoint(name string) string {
	return filepath.Join(d.root, name)
}

func (d *driver) mountServer(mountpoint string) (*fuse.Server, error) {
	if err := os.MkdirAll(filepath.Dir(mountpoint), 0755); err != nil {
		return nil, err
	}

	conf := api.DefaultConfig()
	client, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}

	ownership := keywhizfs.NewOwnership("root", "root")
	kwfs, root := NewFs(client, ownership)

	mountOptions := &fuse.MountOptions{
		AllowOther: true,
		Name:       kwfs.String(),
		Options:    []string{"default_permissions"},
	}

	// Empty Options struct avoids setting a global uid/gid override.
	conn := nodefs.NewFileSystemConnector(root, &nodefs.Options{})
	server, err := fuse.NewServer(conn.RawFS(), mountpoint, mountOptions)
	if err != nil {
		log.Printf("Mount fail: %v\n", err)
		return nil, err
	}

	go server.Serve()

	return server, nil
}
