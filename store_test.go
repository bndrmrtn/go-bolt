package gale

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemStore(t *testing.T) {
	s := NewMemStorage(time.Second)

	err := s.Set("key", []byte("value"))
	assert.Nil(t, err, "Error should be nil")

	data, err := s.Get("key")
	assert.Nil(t, err, "Error should be nil")

	assert.Equal(t, []byte("value"), data, "Data should be value")

	err = s.SetEx("foo", []byte("bar"), time.Second*2)
	assert.Nil(t, err, "Error should be nil")

	data, err = s.Get("foo")
	assert.Nil(t, err, "Error should be nil")

	assert.Equal(t, []byte("bar"), data, "foo should be bar")

	time.Sleep(time.Second * 2)

	_, err = s.Get("foo")
	assert.Error(t, err, "Error should be true, value should be deleted")
}
