package service

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageSequential(t *testing.T) {
	s := newStorage()

	// generate 4 random names and 4 random emails
	names := []string{"John", "Jane", "Jack", "Jill"}
	emails := []string{"john@gmail.com", "jane@gmail.com", "jack@gmail.com", "jill@gmail.com"}

	for i := 0; i < len(names); i++ {
		s.addUser(&user{
			Name:  names[i],
			Email: emails[i],
		})
	}
	// check that the storage contains 4 users
	assert.Len(t, s.users, 4)

	a1 := &asset{
		Type: "insight",
		Desc: "This is asset 1",
		Data: &insight{
			Text: "A simple insight",
		},
	}

	s.addAsset(a1)

	assert.Len(t, s.assets, 1)

	// this should update only the description and
	// leave other fields unchanged.
	assert.NoError(t, s.updateAsset(&asset{
		ID:   1,
		Desc: "Updated desc",
	}))

	a, err := s.asset(1)
	assert.NoError(t, err)
	// assert description changed
	assert.Equal(t, "Updated desc", a.Desc)
	// assert other fields are unchanged
	assert.Equal(t, assetType("insight"), a.Type)

	assert.NoError(t, s.addFavourites(1, 1))

	fav, err := s.userFavourites(1, "")
	assert.NoError(t, err)
	assert.Len(t, fav, 1)

	err = s.addFavourites(1, 1)
	assert.Error(t, err)
	// duplicates should not be added.
	assert.True(t, errors.Is(err, ErrExist))

	s.deleteAsset(1)

	assert.Len(t, s.assets, 0)
	fav, err = s.userFavourites(1, "")
	assert.NoError(t, err)
	assert.Len(t, fav, 0)

	s.deleteUser(4)
	assert.Len(t, s.users, 3)
}

// Tests that adding many objects concurrently to storage is OK.
func TestStorageConcurrency(t *testing.T) {

	s := newStorage()

	// spawn 5 goroutines. Each goroutine will add 10000 users.
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 10000; j++ {
				u := &user{
					Name:  "John",
					Email: "john@gmail.com",
				}
				s.addUser(u)
			}
		}(i)
	}
	wg.Wait()

	require.Len(t, s.users, 5*10000)

	// spawn 5 goroutines. Each goroutine will add 10000 assets.
	var wg2 sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg2.Add(1)
		go func(i int) {
			defer wg2.Done()
			for j := 0; j < 10000; j++ {
				a := &asset{
					Type: "insight",
					Desc: "This is asset",
					Data: &insight{
						Text: "A simple insight",
					},
				}
				s.addAsset(a)
			}
		}(i)
	}
	wg2.Wait()

	require.Len(t, s.assets, 5*10000)

	// For the first 50000 users, add 5 favourites.
	var wg3 sync.WaitGroup
	for i := 0; i < 5*1000; i++ {
		wg3.Add(1)
		go func(i int) {
			defer wg3.Done()
			for j := 0; j < 5; j++ {
				require.NoError(t, s.addFavourites(uint(i+1), uint(j+1)))
			}
		}(i)
	}
	wg3.Wait()

	for i := 0; i < 5*1000; i++ {
		fav, err := s.userFavourites(uint(i+1), "")
		require.NoError(t, err)
		require.Len(t, fav, 5, i)
	}
}

func BenchmarkReadUserFavourites(b *testing.B) {

	s := newStorage()

	s.addUser(&user{
		Name:  "user",
		Email: "user@gmail.com",
	})

	s.addAsset(&asset{
		Type: "insight",
		Desc: "awesome insight",
		Data: &insight{
			Text: "40% of millenials spend more than 3hours on social media daily",
		},
	})

	assert.NoError(b, s.addFavourites(1, 1))

	b.SetBytes(183) // number of bytes of the marshalled json asset.
	for i := 0; i < b.N; i++ {
		assets, _ := s.userFavourites(1, "")
		json.Marshal(assets)
	}

}
