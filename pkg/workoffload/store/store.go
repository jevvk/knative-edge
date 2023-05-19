package store

import (
	"context"
	"sync"
	"time"

	"edge.jevv.dev/pkg/controllers"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// old keys are removed every 30 minutes
const StoreCleanupPeriod = time.Minute * 30

// retain items for max 1 hour
const StoreMaxItemTtl = time.Hour

type Store struct {
	manager.Runnable
	manager.LeaderElectionRunnable

	Log logr.Logger

	data map[string]storeItem
	lock sync.Mutex
}

type storeItem struct {
	Timestamp time.Time
	Key       string
	Value     int64
}

func (s *Store) NeedLeaderElection() bool {
	return false
}

func (s *Store) cleanUp() {
	debug := s.Log.V(controllers.DebugLevel)

	s.lock.Lock()
	defer s.lock.Unlock()

	debug.Info("cleaning up data store")

	keysToRemove := make([]string, 0)
	now := time.Now()

	for key, item := range s.data {
		if item.Timestamp.Add(StoreMaxItemTtl).After(now) {
			continue
		}

		keysToRemove = append(keysToRemove, key)
	}

	for _, key := range keysToRemove {
		delete(s.data, key)
	}

	debug.Info("finished cleaning up data store", "keysRemoved", len(keysToRemove))
}

func (s *Store) Start(ctx context.Context) error {
	if s.data == nil {
		s.data = make(map[string]storeItem)
	}

	go func() {
		for {
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), StoreCleanupPeriod)

			select {
			case <-timeoutCtx.Done():
				timeoutCancel() // not sure if necessary
			case <-ctx.Done():
				timeoutCancel()
				return
			}

			s.cleanUp()
		}
	}()

	return nil
}

func (s *Store) Set(key string, value int64) {
	debug := s.Log.V(controllers.DebugLevel)

	s.lock.Lock()
	defer s.lock.Unlock()

	debug.Info("updating data store item", "key", key, "value", value)

	item := storeItem{
		Timestamp: time.Now(),
		Key:       key,
		Value:     value,
	}

	s.data[key] = item
}

func (s *Store) Get(key string) (int64, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	item, exists := s.data[key]

	if !exists {
		return 0, false
	}

	return item.Value, true
}

func (s *Store) GetLastUpdateTimestamp(key string) (time.Time, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	item, exists := s.data[key]

	if !exists {
		return time.Now(), false
	}

	return item.Timestamp, true
}
