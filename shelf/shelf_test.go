package shelf_test

import (
	"github.com/cskr/pubsub"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stream-first/common"
	"stream-first/shelf"
	"testing"
	"time"
)

var (
	orderIDs = []uuid.UUID{
		uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
)

func generateOrderIds(number int) (orderIDs []uuid.UUID) {
	for i := 0; i < number; i++ {
		orderIDs = append(orderIDs, uuid.New())
	}
	return
}

func Test_primaryShelf_store(t *testing.T) {
	t.Run("Store succeeds when shelf is below capacity", func(t *testing.T) {
		capacity := 5
		s := shelf.NewPrimaryShelf(capacity)
		for i := 0; i < capacity; i++ {
			orderID := uuid.New()
			stored := s.Store(orderID)
			require.True(t, stored, "iteration %v", i)
			has := s.Has(orderID)
			require.True(t, has, "iteration %v", i)
		}
	})
	t.Run("Store fails when shelf is at capacity", func(t *testing.T) {
		capacity := 5
		s := shelf.NewPrimaryShelf(capacity)
		for i := 0; i <= capacity; i++ {
			_ = s.Store(uuid.New())
		}
		orderID := uuid.New()
		stored := s.Store(orderID)
		require.False(t, stored)
	})
}

func Test_primaryShelf_remove(t *testing.T) {
	t.Run("Remove fails when shelf is empty", func(t *testing.T) {
		s := shelf.NewPrimaryShelf(3)
		removed := s.Remove(uuid.New())
		assert.False(t, removed)
	})
	t.Run("Remove fails when order not on shelf", func(t *testing.T) {
		s := shelf.NewPrimaryShelf(3)
		id1, id2, id3, id4 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		_ = s.Store(id1)
		_ = s.Store(id2)
		_ = s.Store(id3)
		removed := s.Remove(id4)
		assert.False(t, removed)
	})
	t.Run("Remove succeeds when order on shelf", func(t *testing.T) {
		s := shelf.NewPrimaryShelf(3)
		id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
		_ = s.Store(id1)
		_ = s.Store(id2)
		_ = s.Store(id3)
		removed := s.Remove(id2)
		assert.True(t, removed)
	})
}

func Test_overflowShelf_store(t *testing.T) {
	t.Run("Store fails for invalid temp", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, err := s.Store(uuid.New(), "blah", 1)
		assert.Error(t, err)
	})
	t.Run("Store succeeds for valid temp", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		stored, err := s.Store(uuid.New(), "cold", 1)
		assert.NoError(t, err)
		assert.True(t, stored)
	})
}

func Test_overflowShelf_store_has(t *testing.T) {
	t.Run("Has returns error for invalid temp", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, err := s.Has(uuid.New(), "blah")
		assert.Error(t, err)
	})
	t.Run("Has returns false when shelf is empty", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		has, err := s.Has(uuid.New(), "cold")
		assert.NoError(t, err)
		assert.False(t, has)
	})
	t.Run("Has returns true when order is on shelf", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		id1 := uuid.New()
		_, _ = s.Store(uuid.New(), "cold", 1)
		_, _ = s.Store(id1, "hot", 1)
		_, _ = s.Store(uuid.New(), "hot", 1)

		has, err := s.Has(id1, "hot")
		assert.NoError(t, err)
		assert.True(t, has)
	})
	t.Run("Has returns false when order is not stored", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.Store(uuid.New(), "cold", 1)
		has, err := s.Has(uuid.New(), "cold")
		assert.NoError(t, err)
		assert.False(t, has)
	})
	t.Run("Has returns false order with same id and different temp is stored", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		id := uuid.New()
		_, _ = s.Store(id, "cold", 1)
		has, err := s.Has(id, "hot")
		assert.NoError(t, err)
		assert.False(t, has)
	})
}

func Test_overflowShelf_store_popMax(t *testing.T) {
	t.Run("PopMax returns error for invalid temp", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _, err := s.PopMax("blah")
		assert.Error(t, err)
	})
	t.Run("PopMax returns false when there are no orders for temp", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, has, err := s.PopMax("cold")
		assert.NoError(t, err)
		assert.False(t, has)
	})
	t.Run("PopMax returns the order id when one order is stored", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		wantOrderID := uuid.New()
		_, _ = s.Store(wantOrderID, "cold", 1)
		gotOrderID, has, err := s.PopMax("cold")
		assert.NoError(t, err)
		assert.True(t, has)
		assert.Equal(t, wantOrderID, gotOrderID)
	})
	t.Run("PopMax succeeds and returns the max when multiple ids are stored", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.Store(orderIDs[1], "cold", 1.4)
		_, _ = s.Store(orderIDs[2], "cold", 1.1)
		_, _ = s.Store(orderIDs[3], "cold", 1.7)
		_, _ = s.Store(orderIDs[4], "cold", 1.2)
		orderID, has, err := s.PopMax("cold")
		assert.NoError(t, err)
		assert.True(t, has)
		assert.Equal(t, orderID, orderIDs[3])
	})
}

func Test_overflowShelf_numOrders(t *testing.T) {
	t.Run("NumOrders returns 0 when shelf is empty", func(t *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		assert.Equal(t, 0, s.NumOrders())
	})
	t.Run("NumOrders returns 2 when shelf Has 2 orders with same temp", func(t *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.Store(uuid.New(), "hot", 1)
		_, _ = s.Store(uuid.New(), "hot", 1)
		assert.Equal(t, 2, s.NumOrders())
	})
	t.Run("NumOrders returns 2 when shelf Has 2 orders with different shelf", func(t *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.Store(uuid.New(), "frozen", 1)
		_, _ = s.Store(uuid.New(), "hot", 1)
		assert.Equal(t, 2, s.NumOrders())
	})
}

func Test_overflowShelf_remove(t *testing.T) {
	t.Run("Remove returns error for invalid temp", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, err := s.Remove(orderIDs[1], "blah")
		assert.Error(t, err)
	})
	t.Run("Remove returns error when no orders stored for temp", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, err := s.Remove(orderIDs[1], "cold")
		assert.Error(t, err)
	})
	t.Run("Remove succeeds when one order is stored", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.Store(orderIDs[1], "cold", 1)
		removed, removeErr := s.Remove(orderIDs[1], "cold")
		assert.NoError(t, removeErr)
		assert.True(t, removed)
		has, _ := s.Has(orderIDs[1], "cold")
		assert.False(t, has)
	})
	t.Run("Remove returns error when an order is stored with same id and different temp", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.Store(orderIDs[1], "cold", 1)
		_, err := s.Remove(orderIDs[1], "frozen")
		assert.Error(t, err)
	})
	t.Run("Remove returns the order with max decay rate when multiple orders are stored for a given temp", func(r *testing.T) {
		s := shelf.NewOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.Store(orderIDs[1], "cold", 1.4)
		_, _ = s.Store(orderIDs[2], "cold", 1.1)
		_, _ = s.Store(orderIDs[3], "cold", 1.7)
		_, _ = s.Store(orderIDs[4], "cold", 1.2)
		removed, removeErr := s.Remove(orderIDs[3], "cold")
		assert.NoError(t, removeErr)
		assert.True(t, removed)
		has, _ := s.Has(orderIDs[3], "cold")
		assert.False(t, has)
	})
}

func Test_warehouse_store_has(t *testing.T) {
	t.Run("Store fails for invalid temp, does not publish", func(t *testing.T) {
		ps := newMockPubSub(map[string]bool{})
		ps.On("Pub", mock.Anything, mock.Anything)
		m := shelf.NewManager(ps, 3, 5)
		o := common.Order{}
		_, err := m.Store(o, "blah", time.Time{})
		assert.Error(t, err)
		ps.AssertNotCalled(t, "Pub")
	})
	t.Run("Store places order on primary shelf if it is not full for order's temp", func(t *testing.T) {
		ps := newMockPubSub(map[string]bool{})
		ps.On("Pub", mock.Anything, mock.Anything)
		m := shelf.NewManager(ps, 4, 5)

		var wantEvents []common.ShelvedEvent
		for _, id := range []uuid.UUID{orderIDs[0], orderIDs[1], orderIDs[2], orderIDs[3]} {
			o, dt, temp := common.Order{ID: id}, time.Now(), "frozen"
			wantEvents = append(wantEvents, common.ShelvedEvent{Dt: dt, Shelf: temp, Order: o})
			stored, storeErr := m.Store(o, temp, dt)
			require.NoError(t, storeErr)
			require.True(t, stored)
			shelfName, found, err := m.Has(id, temp)
			require.NoError(t, err)
			require.True(t, found)
			assert.Equal(t, temp, shelfName)
		}
		ps.AssertCalled(t, "Pub", mock.Anything, []string{common.ShelvedTopic})
		ps.AssertNumberOfCalls(t, "Pub", 4)
	})
	t.Run("Store places order in overflow when primary shelf is full for order's temp", func(t *testing.T) {
		ps := pubsub.New(1000)
		m := shelf.NewManager(ps, 3, 5)

		// Fill up primary
		for _, id := range []uuid.UUID{orderIDs[1], orderIDs[2], orderIDs[3]} {
			_, _ = m.Store(common.Order{ID: id}, "frozen", time.Time{})
		}
		stored, _ := m.Store(common.Order{ID: orderIDs[4]}, "frozen", time.Time{})
		require.True(t, stored)
		shelfName, found, err := m.Has(orderIDs[4], "frozen")
		require.NoError(t, err)
		require.True(t, found)
		assert.Equal(t, "overflow", shelfName)
	})
	t.Run("Store stores in overflow when overflow is nearly full", func(t *testing.T) {
		ps := pubsub.New(1000)
		m := shelf.NewManager(ps, 3, 5)
		orderIDs := generateOrderIds(8)

		// Fill up primary and nearly all of overflow
		for i := 0; i < 7; i++ {
			_, _ = m.Store(common.Order{ID: orderIDs[i]}, "frozen", time.Time{})
		}
		stored, storeErr := m.Store(common.Order{ID: orderIDs[7]}, "frozen", time.Time{})
		require.NoError(t, storeErr)
		require.True(t, stored)
		shelfName, found, err := m.Has(orderIDs[7], "frozen")
		require.NoError(t, err)
		require.True(t, found)
		assert.Equal(t, "overflow", shelfName)
	})
	t.Run("Store returns false and does not Store when overflow is full", func(t *testing.T) {
		ps := pubsub.New(1000)
		m := shelf.NewManager(ps, 3, 5)
		orderIDs := generateOrderIds(9)

		// Fill up primary and overflow
		for i := 0; i < 8; i++ {
			_, _ = m.Store(common.Order{ID: orderIDs[i]}, "frozen", time.Time{})
		}
		stored, storeErr := m.Store(common.Order{ID: orderIDs[8]}, "frozen", time.Time{})
		require.NoError(t, storeErr)
		require.False(t, stored)
		_, found, err := m.Has(orderIDs[8], "frozen")
		require.NoError(t, err)
		require.False(t, found)
	})
}

func Test_warehouse_remove(t *testing.T) {
	t.Run("Remove returns error for invalid temp", func(t *testing.T) {
		ps := pubsub.New(1000)
		m := shelf.NewManager(ps, 3, 5)
		_, err := m.Remove(uuid.New(), "blah", time.Time{})
		assert.Error(t, err)
	})
	t.Run("Remove returns error when order not in storage", func(t *testing.T) {
		ps := pubsub.New(1000)
		m := shelf.NewManager(ps, 3, 5)
		_, err := m.Remove(uuid.New(), "hot", time.Time{})
		assert.Error(t, err)
	})
	t.Run("Remove removes order when it's on the primary shelf", func(t *testing.T) {
		ps := pubsub.New(1000)
		m := shelf.NewManager(ps, 3, 5)
		_, _ = m.Store(common.Order{ID: orderIDs[0]}, "cold", time.Time{})
		found, err := m.Remove(orderIDs[0], "cold", time.Time{})
		require.NoError(t, err)
		require.True(t, found)
		_, found, err = m.Has(orderIDs[0], "cold")
		require.NoError(t, err)
		assert.False(t, found)
	})
	t.Run("Remove removes order when it's on the overflow shelf", func(t *testing.T) {
		ps := pubsub.New(1000)
		m := shelf.NewManager(ps, 3, 5)
		// Fill up primary
		for i := 0; i < 3; i++ {
			_, _ = m.Store(common.Order{ID: orderIDs[i]}, "hot", time.Time{})
		}
		_, _ = m.Store(common.Order{ID: orderIDs[3]}, "hot", time.Time{})
		found, err := m.Remove(orderIDs[3], "hot", time.Time{})
		require.NoError(t, err)
		require.True(t, found)
		_, found, err = m.Has(orderIDs[3], "hot")
		require.NoError(t, err)
		assert.False(t, found)
	})
	t.Run("Remove reshelves order with max decay rate from overflow to primary when it becomes available", func(t *testing.T) {
		ps := pubsub.New(1000)
		m := shelf.NewManager(ps, 3, 5)

		// Fill up the primary hot shelf
		for i := 0; i < 3; i++ {
			_, _ = m.Store(common.Order{ID: orderIDs[i]}, "hot", time.Time{})
		}
		// Fill up overflow
		_, _ = m.Store(common.Order{ID: orderIDs[3], DecayRate: 1.1}, "hot", time.Time{})
		_, _ = m.Store(common.Order{ID: orderIDs[4], DecayRate: 1.6}, "hot", time.Time{})
		_, _ = m.Store(common.Order{ID: orderIDs[5], DecayRate: 1.2}, "hot", time.Time{})
		_, _ = m.Store(common.Order{ID: orderIDs[6], DecayRate: 1.7}, "hot", time.Time{})
		_, _ = m.Store(common.Order{ID: orderIDs[7], DecayRate: 1.2}, "hot", time.Time{})
		// Make space available on the hot shelf
		found, err := m.Remove(orderIDs[1], "hot", time.Time{})
		require.NoError(t, err)
		require.True(t, found)
		// The hot order with maximal decay rate is now on the primary shelf
		var shelfName string
		shelfName, found, err = m.Has(orderIDs[6], "hot")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "hot", shelfName)
	})
}

type mockPubSub struct {
	mock.Mock
	topicsToFollow map[string]bool
}

func newMockPubSub(topics map[string]bool) *mockPubSub {
	return &mockPubSub{topicsToFollow: topics}
}

func (ps *mockPubSub) Sub(topics ...string) chan interface{} {
	return make(chan interface{})
}

func (ps *mockPubSub) Pub(msg interface{}, topics ...string) {
	if len(ps.topicsToFollow) == 0 || ps.topicsToFollow[topics[0]] {
		ps.Called(msg, topics)
	}
}
