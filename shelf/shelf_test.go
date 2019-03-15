package shelf

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stream-first/common"
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
	type args struct {
		orderID uuid.UUID
	}
	type outcome struct {
		ret bool
		has bool
	}
	tests := []struct {
		name string
		init func() *primaryShelf
		args args
		want outcome
	}{{
		name: "store to empty succeeds",
		init: func() *primaryShelf { return newPrimaryShelf(3) },
		args: args{orderIDs[1]},
		want: outcome{true, true},
	}, {
		name: "store below capacity succeeds",
		init: func() *primaryShelf { s := newPrimaryShelf(3); s.store(orderIDs[2]); return s },
		args: args{orderIDs[1]},
		want: outcome{true, true},
	}, {
		init: func() *primaryShelf {
			s := newPrimaryShelf(3)
			s.store(orderIDs[2])
			s.store(orderIDs[3])
			s.store(orderIDs[4])
			return s
		},
		name: "Store at capacity fails",
		args: args{orderIDs[1]},
		want: outcome{false, false},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.init()
			got := outcome{ret: s.store(tt.args.orderID), has: s.has(tt.args.orderID)}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_primaryShelf_remove(t *testing.T) {
	type args struct {
		orderID uuid.UUID
	}
	type outcome struct {
		ret bool
		has bool
	}
	tests := []struct {
		name string
		init func() *primaryShelf
		args args
		want outcome
	}{
		{
			name: "remove from empty fails",
			init: func() *primaryShelf { return newPrimaryShelf(3) },
			args: args{orderIDs[1]},
			want: outcome{ret: false, has: false},
		},
		{
			name: "remove non-member fails",
			init: func() *primaryShelf { s := newPrimaryShelf(3); s.store(orderIDs[2]); return s },
			args: args{orderIDs[1]},
			want: outcome{ret: false, has: false},
		},
		{
			name: "remove member succeeds",
			init: func() *primaryShelf {
				s := newPrimaryShelf(3)
				s.store(orderIDs[1])
				s.store(orderIDs[2])
				return s
			},
			args: args{orderIDs[1]},
			want: outcome{ret: true, has: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.init()
			got := outcome{ret: s.remove(tt.args.orderID), has: s.has(tt.args.orderID)}
			if got != tt.want {
				t.Errorf("primaryShelf.remove() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_overflowShelf_numOrders(t *testing.T) {
	s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
	t.Run("Empty returns zero", func(t *testing.T) {
		want := 0
		if got := s.numOrders(); got != want {
			t.Errorf("overflowShelf.numOrders() = %v, want %v", got, want)
		}
	})

	_, _ = s.store(orderIDs[1], "frozen", 1)
	_, _ = s.store(orderIDs[2], "frozen", 1)
	_, _ = s.store(orderIDs[3], "hot", 1)
	t.Run("Non empty returns as expected", func(t *testing.T) {
		want := 3
		if got := s.numOrders(); got != want {
			t.Errorf("overflowShelf.numOrders() = %v, want %v", got, want)
		}
	})
}

func Test_overflowShelf_store(t *testing.T) {
	t.Run("store fails for invalid temp", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, err := s.store(orderIDs[1], "blah", 1)
		assert.Error(t, err)
	})
	t.Run("store succeeds for valid temp", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		stored, err := s.store(orderIDs[1], "hot", 1)
		assert.NoError(t, err)
		assert.True(t, stored)
	})
}

func Test_overflowShelf_store_has(t *testing.T) {
	t.Run("has fails for invalid temp", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, err := s.has(orderIDs[1], "blah")
		assert.Error(t, err)
	})
	t.Run("has succeeds and returns false for empty shelf", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		has, err := s.has(orderIDs[1], "cold")
		assert.NoError(t, err)
		assert.False(t, has)
	})
	t.Run("has succeeds and returns true for id that is stored", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.store(orderIDs[1], "cold", 1)
		has, err := s.has(orderIDs[1], "cold")
		assert.NoError(t, err)
		assert.True(t, has)
	})
	t.Run("has succeeds and returns false for id that is not stored", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.store(orderIDs[1], "cold", 1)
		has, err := s.has(orderIDs[2], "cold")
		assert.NoError(t, err)
		assert.False(t, has)
	})
	t.Run("has succeeds and returns false for id that is in a different temp/section", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.store(orderIDs[1], "cold", 1)
		has, err := s.has(orderIDs[1], "hot")
		assert.NoError(t, err)
		assert.False(t, has)
	})
}

func Test_overflowShelf_store_popMax(t *testing.T) {
	t.Run("popMax fails for invalid temp", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _, err := s.popMax("blah")
		assert.Error(t, err)
	})
	t.Run("popMax succeeds and returns false for empty shelf", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, has, err := s.popMax("cold")
		assert.NoError(t, err)
		assert.False(t, has)
	})
	t.Run("popMax succeeds and returns correct id when one id is stored", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.store(orderIDs[1], "cold", 1)
		orderID, has, err := s.popMax("cold")
		assert.NoError(t, err)
		assert.True(t, has)
		assert.Equal(t, orderID, orderIDs[1])
	})
	t.Run("popMax succeeds and returns false for id that is in a different temp/section", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.store(orderIDs[1], "cold", 1)
		_, has, err := s.popMax("hot")
		assert.NoError(t, err)
		assert.False(t, has)
	})
	t.Run("popMax succeeds and returns the max when multiple ids are stored", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.store(orderIDs[1], "cold", 1.4)
		_, _ = s.store(orderIDs[2], "cold", 1.1)
		_, _ = s.store(orderIDs[3], "cold", 1.7)
		_, _ = s.store(orderIDs[4], "cold", 1.2)
		orderID, has, err := s.popMax("cold")
		assert.NoError(t, err)
		assert.True(t, has)
		assert.Equal(t, orderID, orderIDs[3])
	})
}

func Test_overflowShelf_remove(t *testing.T) {
	t.Run("remove fails for invalid temp", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, err := s.remove(orderIDs[1], "blah")
		assert.Error(t, err)
	})
	t.Run("remove fails for empty shelf", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, err := s.remove(orderIDs[1], "cold")
		assert.Error(t, err)
	})
	t.Run("remove succeeds when one id is stored", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.store(orderIDs[1], "cold", 1)
		removed, removeErr := s.remove(orderIDs[1], "cold")
		assert.NoError(t, removeErr)
		assert.True(t, removed)
		has, _ := s.has(orderIDs[1], "cold")
		assert.False(t, has)
	})
	t.Run("remove fails for id that is in a different temp/section", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.store(orderIDs[1], "cold", 1)
		_, err := s.remove(orderIDs[1], "frozen")
		assert.Error(t, err)
	})
	t.Run("remove succeeds and returns the max when multiple ids are stored", func(r *testing.T) {
		s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
		_, _ = s.store(orderIDs[1], "cold", 1.4)
		_, _ = s.store(orderIDs[2], "cold", 1.1)
		_, _ = s.store(orderIDs[3], "cold", 1.7)
		_, _ = s.store(orderIDs[4], "cold", 1.2)
		removed, removeErr := s.remove(orderIDs[3], "cold")
		assert.NoError(t, removeErr)
		assert.True(t, removed)
		has, _ := s.has(orderIDs[3], "cold")
		assert.False(t, has)
	})
}

type mockPubSub struct {
	mock.Mock
}

func Test_warehouse_store_has(t *testing.T) {
	t.Run("store fails for invalid temp", func(t *testing.T) {
		ps := newMockPubSub()
		w := newWarehouse(ps, 3, 5)
		o := common.Order{}
		_, err := w.store(o, "blah", time.Time{})
		assert.Error(t, err)
	})
	t.Run("store in primary for expected temp when primary is not full", func(t *testing.T) {
		ps := newMockPubSub()
		w := newWarehouse(ps, 4, 5)
		for _, id := range []uuid.UUID{orderIDs[1], orderIDs[2], orderIDs[3], orderIDs[4]} {
			stored, storeErr := w.store(common.Order{ID: id}, "frozen", time.Time{})
			require.NoError(t, storeErr)
			require.True(t, stored)
			shelf, found, err := w.has(id, "frozen")
			require.NoError(t, err)
			require.True(t, found)
			assert.Equal(t, "frozen", shelf)
			shelf, found, err = w.has(id, "hot")
			require.NoError(t, err)
			require.False(t, found)
		}
	})
	t.Run("store in overflow when primary is full", func(t *testing.T) {
		ps := newMockPubSub()
		w := newWarehouse(ps, 3, 5)

		// Fill up primary
		for _, id := range []uuid.UUID{orderIDs[1], orderIDs[2], orderIDs[3]} {
			_, _ = w.store(common.Order{ID: id}, "frozen", time.Time{})
		}
		stored, _ := w.store(common.Order{ID: orderIDs[4]}, "frozen", time.Time{})
		require.True(t, stored)
		shelf, found, err := w.has(orderIDs[4], "frozen")
		require.NoError(t, err)
		require.True(t, found)
		assert.Equal(t, "overflow", shelf)
	})
	t.Run("store stores in overflow when overflow is nearly full", func(t *testing.T) {
		ps := newMockPubSub()
		w := newWarehouse(ps, 3, 5)
		orderIDs := generateOrderIds(8)

		// Fill up primary and nearly all of overflow
		for i := 0; i < 7; i++ {
			_, _ = w.store(common.Order{ID: orderIDs[i]}, "frozen", time.Time{})
		}
		stored, storeErr := w.store(common.Order{ID: orderIDs[7]}, "frozen", time.Time{})
		require.NoError(t, storeErr)
		require.True(t, stored)
		shelf, found, err := w.has(orderIDs[7], "frozen")
		require.NoError(t, err)
		require.True(t, found)
		assert.Equal(t, "overflow", shelf)
	})
	t.Run("store returns false and does not store when overflow is full", func(t *testing.T) {
		ps := newMockPubSub()
		w := newWarehouse(ps, 3, 5)
		orderIDs := generateOrderIds(9)

		// Fill up primary and overflow
		for i := 0; i < 8; i++ {
			_, _ = w.store(common.Order{ID: orderIDs[i]}, "frozen", time.Time{})
		}
		stored, storeErr := w.store(common.Order{ID: orderIDs[8]}, "frozen", time.Time{})
		require.NoError(t, storeErr)
		require.False(t, stored)
		_, found, err := w.has(orderIDs[8], "frozen")
		require.NoError(t, err)
		require.False(t, found)
	})

}

func Test_warehouse_remove(t *testing.T) {
	t.Run("remove fails for invalid temp", func(t *testing.T) {
		ps := newMockPubSub()
		w := newWarehouse(ps, 3, 5)
		_, err := w.remove(uuid.New(), "blah", time.Time{})
		assert.Error(t, err)
	})
	t.Run("remove fails when not a member", func(t *testing.T) {
		ps := newMockPubSub()
		w := newWarehouse(ps, 3, 5)
		_, err := w.remove(uuid.New(), "hot", time.Time{})
		assert.Error(t, err)
	})
	t.Run("remove removes when in primary", func(t *testing.T) {
		ps := newMockPubSub()
		w := newWarehouse(ps, 3, 5)
		_, _ = w.store(common.Order{ID: orderIDs[0]}, "cold", time.Time{})
		found, err := w.remove(orderIDs[0], "cold", time.Time{})
		require.NoError(t, err)
		require.True(t, found)
		_, found, err = w.has(orderIDs[0], "cold")
		require.NoError(t, err)
		assert.False(t, found)
	})
	t.Run("remove removes and returns true when in overflow", func(t *testing.T) {
		ps := newMockPubSub()
		w := newWarehouse(ps, 3, 5)
		// Fill up primary
		for i := 0; i < 3; i++ {
			_, _ = w.store(common.Order{ID: orderIDs[i]}, "hot", time.Time{})
		}
		_, _ = w.store(common.Order{ID: orderIDs[3]}, "hot", time.Time{})
		found, err := w.remove(orderIDs[3], "hot", time.Time{})
		require.NoError(t, err)
		require.True(t, found)
		_, found, err = w.has(orderIDs[3], "hot")
		require.NoError(t, err)
		assert.False(t, found)
	})
	t.Run("remove reshelves order with max decay rate from overflow to primary when it becomes available", func(t *testing.T) {
		ps := newMockPubSub()
		//ps.On("Pub", mock.Anything, "reshelved").Return()
		w := newWarehouse(ps, 3, 5)
		// Fill up primary
		for i := 0; i < 3; i++ {
			_, _ = w.store(common.Order{ID: orderIDs[i]}, "hot", time.Time{})
		}
		_, _ = w.store(common.Order{ID: orderIDs[3], DecayRate: 1.1}, "hot", time.Time{})
		_, _ = w.store(common.Order{ID: orderIDs[4], DecayRate: 1.6}, "hot", time.Time{})
		_, _ = w.store(common.Order{ID: orderIDs[5], DecayRate: 1.2}, "hot", time.Time{})
		_, _ = w.store(common.Order{ID: orderIDs[6], DecayRate: 1.7}, "hot", time.Time{})
		_, _ = w.store(common.Order{ID: orderIDs[7], DecayRate: 1.2}, "hot", time.Time{})
		ps.On("Pub", mock.Anything, mock.Anything).Return()
		found, err := w.remove(orderIDs[1], "hot", time.Time{})
		require.NoError(t, err)
		require.True(t, found)
		var shelf string
		shelf, found, err = w.has(orderIDs[6], "hot")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "hot", shelf)
		//ps.AssertExpectations(t)
		ps.AssertCalled(t, "Pub", mock.Anything, "reshelved")
	})
}

func newMockPubSub() *mockPubSub {
	return &mockPubSub{}
}

func (ps *mockPubSub) Sub(topics ...string) chan interface{} {
	return make(chan interface{})
}

func (ps *mockPubSub) Pub(msg interface{}, topics ...string) {
	ps.Called(msg, topics)
}
