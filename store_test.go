package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	s := newStore()
	defer teardown(t, s)

	data := []byte("Some content")
	key := "momsspecials"

	reader := bytes.NewReader(data)
	if n, err := s.writeStream(key, reader); err != nil {
		t.Error(err)
	} else {
		fmt.Println(n)
	}

	if ok := s.Has(key); !ok {
		t.Errorf("expected to have key %s", key)
	}

	_, r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		t.Errorf("want %s have %s", data, b)
	}
	if string(b) != string(data) {
		t.Errorf("want %s have %s", data, b)
	}

	if err := s.Delete(key); err != nil {
		t.Error(err)
	}

	if ok := s.Has(key); ok {
		t.Errorf("expected to NOT have key %s", key)
	}
}

func TestStoreDeleteKey(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	data := []byte("Some content")
	s := NewStore(opts)
	key := "momsspecials"

	if _, err := s.Write(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	return NewStore(opts)
}

func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}

func TestPathTransformFunc(t *testing.T) {
	key := "mombestpicture"
	pathKey := CASPathTransformFunc(key)
	expectedOriginalKey := "cf5d4b01c4d9438c22c56c832f83bd3e8c6304f9"
	expectedPathName := "cf5d4/b01c4/d9438/c22c5/6c832/f83bd/3e8c6/304f9"

	assert.Equal(t, expectedPathName, pathKey.PathName, "Pathname is wrong")
	assert.Equal(t, expectedOriginalKey, pathKey.FileName, "Original key is wrong")
}
