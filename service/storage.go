package service

import (
	"errors"
	"fmt"
	"sync"
)

var ErrExist = errors.New("object already exists")

var storageNumberAssetsPreload = 3000

// storage is an in memory store which holds users, assets and favourites.
// It is safe to use concurrently.
type storage struct {
	usersMu sync.RWMutex // guards users
	// maps user id to User
	users       map[uint]*user
	userCounter uint         // monotonially increasing counter
	assetsMu    sync.RWMutex // guards assets
	// maps user id to Asset
	assets       map[uint]*asset
	assetCounter uint         // monotonially increasing counter
	favouritesMu sync.RWMutex // guards favourites
	// maps user id to a list of asset ids
	favourites map[uint][]uint
}

func newStorage() *storage {
	return &storage{
		users:      make(map[uint]*user),
		assets:     make(map[uint]*asset),
		favourites: make(map[uint][]uint),
	}
}

// // range assets and call f for every asset in the storage. if f returns false,
// // the loop is stopped.
// func (s *storage) rangeAssets(f func(*asset) bool) {
// 	s.assetsMu.RLock()
// 	defer s.assetsMu.RUnlock()
// 	for _, a := range s.assets {
// 		if !f(a) {
// 			return
// 		}

// 	}
// }

// // range users and call f for every asset in the storage. if f returns false,
// // the loop is stopped.
// func (s *storage) rangeUsers(f func(*user) bool) {
// 	s.usersMu.RLock()
// 	defer s.usersMu.RUnlock()
// 	for _, u := range s.users {
// 		if !f(u) {
// 			return
// 		}

// 	}
// }

func (s *storage) getUsers() []*user {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()
	users := make([]*user, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return users
}

func (s *storage) getAssets(atype assetType) []*asset {
	s.assetsMu.RLock()
	defer s.assetsMu.RUnlock()
	assets := make([]*asset, 0, len(s.assets))
	for _, a := range s.assets {
		if atype == "" || a.Type == atype {
			assets = append(assets, a)
		}
	}
	return assets
}

func (s *storage) userWithRLock(id uint) (*user, error) {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()
	return s.user(id)
}

func (s *storage) user(userID uint) (*user, error) {
	user, ok := s.users[userID]
	if !ok {
		return nil, fmt.Errorf("user id %d does not exist", userID)
	}
	return user, nil
}

func (s *storage) assetWithRLock(assetID uint) (*asset, error) {
	s.assetsMu.RLock()
	defer s.assetsMu.RUnlock()
	return s.asset(assetID)
}

func (s *storage) asset(assetID uint) (*asset, error) {
	asset, ok := s.assets[assetID]
	if !ok {
		return nil, fmt.Errorf("asset id %d does not exist", assetID)

	}
	return asset, nil
}

func (s *storage) userFavourites(userID uint, atype assetType) ([]*asset, error) {
	s.favouritesMu.RLock()
	defer s.favouritesMu.RUnlock()
	if _, err := s.user(userID); err != nil {
		return nil, err
	}
	ids, ok := s.favourites[userID]
	if !ok {
		return nil, nil
	}
	assets := make([]*asset, 0, len(ids))
	for _, id := range ids {
		if atype == "" || s.assets[id].Type == assetType(atype) {
			assets = append(assets, s.assets[id])
		}
	}
	return assets, nil
}

func (s *storage) addUser(u *user) uint {
	s.usersMu.Lock()
	defer s.usersMu.Unlock()
	s.userCounter++
	u.ID = s.userCounter
	s.users[s.userCounter] = u
	return u.ID
}

func (s *storage) addAsset(a *asset) uint {
	s.assetsMu.Lock()
	defer s.assetsMu.Unlock()
	s.assetCounter++
	a.ID = s.assetCounter
	s.assets[s.assetCounter] = a
	return a.ID
}

// complexity: O(n), where n = number of assets in user's favourites
func (s *storage) addFavourites(userID, assetID uint) error {
	s.favouritesMu.Lock()
	defer s.favouritesMu.Unlock()
	if _, err := s.user(userID); err != nil {
		return err
	}
	if _, err := s.asset(assetID); err != nil {
		return err
	}
	assets, ok := s.favourites[userID]
	if !ok {
		s.favourites[userID] = []uint{}
	}
	for _, id := range assets {
		if id == assetID {
			// asset id already exists
			return ErrExist
		}
	}
	s.favourites[userID] = append(s.favourites[userID], assetID)
	return nil
}

func (s *storage) updateUser(new *user) error {
	s.usersMu.Lock()
	defer s.usersMu.Unlock()
	old, err := s.user(new.ID)
	if err != nil {
		return err
	}
	old.update(new)
	return nil
}

func (s *storage) updateAsset(new *asset) error {
	s.assetsMu.Lock()
	defer s.assetsMu.Unlock()
	old, err := s.asset(new.ID)
	if err != nil {
		return err
	}
	old.update(new)
	return nil
}

func (s *storage) deleteUser(userID uint) {
	s.usersMu.Lock()
	s.favouritesMu.Lock()
	defer s.usersMu.Unlock()
	defer s.favouritesMu.Unlock()
	delete(s.users, userID)
	delete(s.favourites, userID)
}

func (s *storage) deleteAsset(assetID uint) {
	s.assetsMu.Lock()
	defer s.assetsMu.Unlock()
	delete(s.assets, assetID)
	for user := range s.favourites {
		s.deleteFavourite(user, assetID)
	}
}

// complexity: O(n), where n = number of assets in user's favourites
func (s *storage) deleteFavourite(userID, assetID uint) {
	s.favouritesMu.Lock()
	defer s.favouritesMu.Unlock()
	assets, ok := s.favourites[userID]
	if !ok {
		return
	}
	for i, id := range assets {
		if id == assetID {
			// We want to keep the slice ordered.  Shift all of the
			// elements at the right of the deleting index by one to
			// the left.
			s.favourites[userID] = append(assets[:i], assets[i+1:]...)
			return
		}
	}
}

// add to storage one user and many favourite assets on this user.
func (s *storage) fillWithObjects() error {
	s.addUser(&user{
		Name:  "user",
		Email: "user@gmail.com",
	})

	for i := 0; i < storageNumberAssetsPreload/3; i++ {

		chartID := s.addAsset(&asset{
			Type: "chart",
			Desc: fmt.Sprintf("awesome chart %d", i),
			Data: &chart{
				Title:      "chart",
				TitleAxisX: "x",
				TitleAxisY: "y",
				Data:       []byte("some data"),
			},
		})

		insightID := s.addAsset(&asset{
			Type: "insight",
			Desc: fmt.Sprintf("awesome insight %d", i),
			Data: &insight{
				Text: "40% of millenials spend more than 3hours on social media daily",
			},
		})

		audienceID := s.addAsset(&asset{
			Type: "audience",
			Desc: fmt.Sprintf("awesome audience %d", i),
			Data: &audience{
				Gender:       male,
				BirthCountry: "Greece",
				AgeGroup: ageGroup{
					Min: 20,
					Max: 30,
				},
				SocialMediaHoursUsage: 2,
			},
		})

		err := s.addFavourites(1, chartID)
		if err != nil {
			return err
		}
		err = s.addFavourites(1, insightID)
		if err != nil {
			return err
		}
		err = s.addFavourites(1, audienceID)
		if err != nil {
			return err
		}
	}
	return nil
}
