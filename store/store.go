package store

import (
	"errors"
	"sync"
)

var (
	ErrNotFound     = errors.New("Volume not found")
	ErrAlreadyExist = errors.New("Volume already exists")
)

type Store interface {
	Get(string) (*Volume, error)
	Set(*Volume) error
	Setx(*Volume) error
	Del(string) error
}

type MemoryStore struct {
	volumes map[string]*Volume
	lock    *sync.Mutex
}

func NewMemoryStore() Store {
	return &MemoryStore{
		volumes: make(map[string]*Volume),
		lock:    new(sync.Mutex),
	}
}

func (m *MemoryStore) Get(name string) (*Volume, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	v, ok := m.volumes[name]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}

func (m *MemoryStore) Set(volume *Volume) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.volumes[volume.Name] = volume
	return nil
}

func (m *MemoryStore) Setx(volume *Volume) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.volumes[volume.Name]; ok {
		return ErrAlreadyExist
	}
	m.volumes[volume.Name] = volume
	return nil
}

func (m *MemoryStore) Del(name string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.volumes, name)
	return nil
}
