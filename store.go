package luaprovider

import (
	"github.com/philippgille/gokv"
)

var _ gokv.Store = (*lazyStore)(nil)

// lazyStore opens the cache store on each use, like mangoprovider does: a bbolt store locks its file
// while open, so keeping it open for the provider's lifetime hangs other mangal processes loading it.
type lazyStore struct {
	open func() (gokv.Store, error)
}

func (s *lazyStore) Set(k string, v any) error {
	store, err := s.open()
	if err != nil {
		return err
	}
	defer store.Close()
	return store.Set(k, v)
}

func (s *lazyStore) Get(k string, v any) (bool, error) {
	store, err := s.open()
	if err != nil {
		return false, err
	}
	defer store.Close()
	return store.Get(k, v)
}

func (s *lazyStore) Delete(k string) error {
	store, err := s.open()
	if err != nil {
		return err
	}
	defer store.Close()
	return store.Delete(k)
}

func (s *lazyStore) Close() error {
	return nil
}
