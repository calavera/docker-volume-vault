package store

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/calavera/docker-volume-vault/fs"
	"github.com/calavera/docker-volume-vault/vault"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type Volume struct {
	Name    string
	Token   string
	Options map[string]string
	server  *fuse.Server
	lock    *sync.Mutex
}

func NewVolume(name, token string, options map[string]string) *Volume {
	return &Volume{
		Name:    name,
		Token:   token,
		Options: options,
		lock:    new(sync.Mutex),
	}
}

func (v *Volume) Mount(root string) (string, error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	m := filepath.Join(root, v.Name)
	if v.server != nil {
		// Server already mounted
		return m, nil
	}

	log.Printf("Mounting volume %s on %s\n", v.Name, m)
	fi, err := os.Lstat(m)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(m, 0755); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	if fi != nil && !fi.IsDir() {
		return "", fmt.Errorf("%v already exist and it's not a directory", m)
	}

	server, err := mountServer(v.Token, m)
	if err != nil {
		return "", err
	}

	v.server = server

	return m, nil
}

func (vol *Volume) Mounted() bool {
	return vol.server != nil
}

func (vol *Volume) Unmount() error {
	return vol.server.Unmount()
}

func mountServer(token, mountpoint string) (*fuse.Server, error) {
	if err := os.MkdirAll(filepath.Dir(mountpoint), 0755); err != nil {
		return nil, err
	}

	client, err := vault.Client(token)
	if err != nil {
		return nil, err
	}

	kwfs, root := fs.NewFs(client)

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
