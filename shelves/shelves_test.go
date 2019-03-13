package shelves

import (
	"reflect"
	"stream-first/event"
	"testing"
	"time"

	"github.com/cskr/pubsub"
	"github.com/google/uuid"
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

func Test_overflowShelf_store(t *testing.T) {
	type fields struct {
		capacity int
		slots    map[string]map[uuid.UUID]float32
	}
	type args struct {
		orderID   uuid.UUID
		temp      string
		dacayRate float32
	}
	tests := []struct {
		name    string
		init    func()
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &overflowShelf{
				capacity: tt.fields.capacity,
				slots:    tt.fields.slots,
			}
			got, err := s.store(tt.args.orderID, tt.args.temp, tt.args.dacayRate)
			if (err != nil) != tt.wantErr {
				t.Errorf("overflowShelf.store() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("overflowShelf.store() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_overflowShelf_remove(t *testing.T) {
	type fields struct {
		capacity int
		slots    map[string]map[uuid.UUID]float32
	}
	type args struct {
		orderID uuid.UUID
		temp    string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &overflowShelf{
				capacity: tt.fields.capacity,
				slots:    tt.fields.slots,
			}
			if got := s.remove(tt.args.orderID, tt.args.temp); got != tt.want {
				t.Errorf("overflowShelf.remove() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_overflowShelf_popMax(t *testing.T) {
	type fields struct {
		capacity int
		slots    map[string]map[uuid.UUID]float32
	}
	type args struct {
		temp string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   uuid.UUID
		want1  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &overflowShelf{
				capacity: tt.fields.capacity,
				slots:    tt.fields.slots,
			}
			got, got1 := s.popMax(tt.args.temp)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("overflowShelf.popMax() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("overflowShelf.popMax() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_newPickupShelves(t *testing.T) {
	type args struct {
		ps *pubsub.PubSub
	}
	tests := []struct {
		name string
		args args
		want *pickupShelves
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newPickupShelves(tt.args.ps); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newPickupShelves() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pickupShelves_store(t *testing.T) {
	type fields struct {
		shelves  map[string]*primaryShelf
		overflow overflowShelf
		ps       *pubsub.PubSub
	}
	type args struct {
		order event.Order
		temp  string
		Dt    time.Time
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pus := &pickupShelves{
				shelves:  tt.fields.shelves,
				overflow: tt.fields.overflow,
				ps:       tt.fields.ps,
			}
			if got := pus.store(tt.args.order, tt.args.temp, tt.args.Dt); got != tt.want {
				t.Errorf("pickupShelves.store() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pickupShelves_remove(t *testing.T) {
	type fields struct {
		shelves  map[string]*primaryShelf
		overflow overflowShelf
		ps       *pubsub.PubSub
	}
	type args struct {
		orderID uuid.UUID
		temp    string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pus := &pickupShelves{
				shelves:  tt.fields.shelves,
				overflow: tt.fields.overflow,
				ps:       tt.fields.ps,
			}
			if got := pus.remove(tt.args.orderID, tt.args.temp); got != tt.want {
				t.Errorf("pickupShelves.remove() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pickupShelves_reshelf(t *testing.T) {
	type fields struct {
		shelves  map[string]*primaryShelf
		overflow overflowShelf
		ps       *pubsub.PubSub
	}
	type args struct {
		temp string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pus := &pickupShelves{
				shelves:  tt.fields.shelves,
				overflow: tt.fields.overflow,
				ps:       tt.fields.ps,
			}
			if got := pus.reshelf(tt.args.temp); got != tt.want {
				t.Errorf("pickupShelves.reshelf() = %v, want %v", got, tt.want)
			}
		})
	}
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
