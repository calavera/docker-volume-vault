package main

import (
	"encoding/base64"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/calavera/docker-volume-vault/store"
	"github.com/calavera/docker-volume-vault/vault"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/hashicorp/vault/api"
)

type driver struct {
	root  string
	token string
	store store.Store
}

func newDriver(root, token string) *driver {
	return &driver{
		root:  root,
		token: token,
		store: store.NewMemoryStore(),
	}
}

func (d *driver) Create(r volume.Request) volume.Response {
	vol := store.NewVolume(r.Name, d.token, r.Options)
	if err := d.store.Setx(vol); err != nil {
		return volume.Response{Err: err.Error()}
	}

	if rules, ok := r.Options["policy-rules"]; ok {
		name := r.Options["policy-name"]
		if name == "" {
			name = "docker-policy-" + r.Name
		}
		token, err := d.createPolicy(name, rules)
		if err != nil {
			return volume.Response{Err: err.Error()}
		}
		vol.Token = token
		d.store.Set(vol)
	}
	return volume.Response{}
}

func (d *driver) Get(r volume.Request) volume.Response {
	return volume.Response{}
}

func (d *driver) List(r volume.Request) volume.Response {
	return volume.Response{}
}

func (d *driver) Remove(r volume.Request) volume.Response {
	err := d.store.Del(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}
}

func (d *driver) Path(r volume.Request) volume.Response {
	return volume.Response{Mountpoint: d.mountpoint(r.Name)}
}

func (d *driver) Mount(r volume.Request) volume.Response {
	vol, err := d.store.Get(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	mount, err := vol.Mount(d.root)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Mountpoint: mount}
}

func (d driver) Unmount(r volume.Request) volume.Response {
	vol, err := d.store.Get(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	if vol.Mounted() {
		if err := vol.Unmount(); err != nil {
			return volume.Response{Err: err.Error()}
		}
	}

	return volume.Response{}
}

func (d *driver) mountpoint(name string) string {
	return filepath.Join(d.root, name)
}

func (d *driver) client() (*api.Client, error) {
	return vault.Client(d.token)
}

func (d *driver) createPolicy(name, policy string) (string, error) {
	var rules []byte
	var err error
	if strings.HasPrefix(policy, "@") {
		rules, err = ioutil.ReadFile(strings.TrimPrefix(policy, "@"))
	} else {
		rules, err = base64.StdEncoding.DecodeString(policy)
	}
	if err != nil {
		return "", err
	}

	client, err := d.client()
	if err != nil {
		return "", err
	}

	if err := client.Sys().PutPolicy(name, string(rules)); err != nil {
		return "", err
	}

	req := &api.TokenCreateRequest{
		Policies: []string{name},
	}

	secret, err := client.Auth().Token().Create(req)
	if err != nil {
		return "", err
	}
	return secret.Auth.ClientToken, nil
}
