package shelves

import (
	"github.com/cskr/pubsub"
	"github.com/google/uuid"
	"testing"
)

var (
	orderID1 = uuid.New()
	orderID2 = uuid.New()
	orderID3 = uuid.New()
	orderID4 = uuid.New()
)

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
		args: args{orderID1},
		want: outcome{true, true},
	}, {
		name: "store below capacity succeeds",
		init: func() *primaryShelf { s := newPrimaryShelf(3); s.store(orderID2); return s },
		args: args{orderID1},
		want: outcome{true, true},
	}, {
		init: func() *primaryShelf {
			s := newPrimaryShelf(3)
			s.store(orderID2)
			s.store(orderID3)
			s.store(orderID4)
			return s
		},
		name: "Store at capacity fails",
		args: args{orderID1},
		want: outcome{false, false},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.init()
			got := outcome{ret: s.store(tt.args.orderID), has: s.has(tt.args.orderID)}

			if got != tt.want {
				t.Errorf("primaryShelf.store() = %v, want %v", got, tt.want)
			}
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
			args: args{orderID1},
			want: outcome{ret: false, has: false},
		},
		{
			name: "remove non-member fails",
			init: func() *primaryShelf { s := newPrimaryShelf(3); s.store(orderID2); return s },
			args: args{orderID1},
			want: outcome{ret: false, has: false},
		},
		{
			name: "remove member succeeds",
			init: func() *primaryShelf {
				s := newPrimaryShelf(3)
				s.store(orderID1)
				s.store(orderID2)
				return s
			},
			args: args{orderID1},
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

// TODO: use standard structure
func Test_overflowShelf_numOrders(t *testing.T) {
	s := newOverflowShelf(5, []string{"frozen", "cold", "hot"})
	t.Run("Empty returns zero", func(t *testing.T) {
		want := 0
		if got := s.numOrders(); got != want {
			t.Errorf("overflowShelf.numOrders() = %v, want %v", got, want)
		}
	})

	s.store(orderID1, "frozen", 1)
	s.store(orderID2, "frozen", 1)
	s.store(orderID3, "hot", 1)
	t.Run("Non empty returns as expected", func(t *testing.T) {
		want := 3
		if got := s.numOrders(); got != want {
			t.Errorf("overflowShelf.numOrders() = %v, want %v", got, want)
		}
	})
}

func TestRun(t *testing.T) {
	type args struct {
		ps *pubsub.PubSub
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Run(tt.args.ps)
		})
	}
}
