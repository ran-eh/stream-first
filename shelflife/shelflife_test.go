package shelflife_test

import (
	"github.com/stretchr/testify/mock"
	"stream-first/common"
	"stream-first/mocks"
	"stream-first/shelflife"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const timeFormat = "2006-01-02 15:04:05"

var testOrder = common.Order{ID: uuid.New(), Name: "an order", Temp: "hot", ShelfLife: 100, DecayRate: 1}

func Test_orderState_Value(t *testing.T) {
	type fields struct {
		shelfLife                       float32
		decayRate                       float32
		timePlacedOnPrimaryShelfString  string
		timePlacedOnOverflowShelfString string
		secondsSinceShelving            float64
	}
	tests := []struct {
		name      string
		fields    fields
		wantValue float32
		wantErr   bool
	}{
		{
			name:    "Value returns error if neither start times is set",
			fields:  fields{},
			wantErr: true,
		},
		{
			name: "Value returns error if reshelved from primary to overflow",
			fields: fields{
				timePlacedOnPrimaryShelfString:  "2019-01-02 15:04:05",
				timePlacedOnOverflowShelfString: "2019-01-02 15:05:07",
			},
			wantErr: true,
		},
		{
			name: "Value subtracts age and age times decay rate if order was always on a primary shelf",
			fields: fields{
				shelfLife:                      100,
				decayRate:                      0.1,
				timePlacedOnPrimaryShelfString: "2019-01-02 15:04:05",
				secondsSinceShelving:           7,
			},
			wantValue: 100 - 7 - 0.7,
		},
		{
			name: "Value subtracts age and age times twice decay rate if order was always on a overflow shelf",
			fields: fields{
				shelfLife:                       100,
				decayRate:                       0.1,
				timePlacedOnOverflowShelfString: "2019-01-02 15:04:05",
				secondsSinceShelving:            4,
			},
			wantValue: 100 - 4 - 2*0.4,
		},
		{
			name: "Value considers both decay rates for orders that were reshelved from overflow to primary",
			fields: fields{
				shelfLife:                       100,
				decayRate:                       0.1,
				timePlacedOnOverflowShelfString: "2019-01-02 15:04:05",
				// Spent 2 seconds on overflow...
				timePlacedOnPrimaryShelfString: "2019-01-02 15:04:07",
				// ... then 3 seconds on primary
				secondsSinceShelving: 5,
			},
			wantValue: 100 - 5 - 2*(2*0.1) - 3*0.1,
		},
		{
			name: "Value returns 0 once it the order expiration time arrives",
			fields: fields{
				shelfLife:                       10,
				decayRate:                       0.1,
				timePlacedOnOverflowShelfString: "2019-01-02 15:04:05",
				// Spent 2 seconds on overflow...
				timePlacedOnPrimaryShelfString: "2019-01-02 15:04:07",
				// ... then given plenty of time to expire
				secondsSinceShelving: 15,
			},
			wantValue: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// "now" is a bit of a misnomer - it refers to the time for which order value is calculated.
			var timePlacedOnOverflowShelf, timePlacedOnPrimaryShelf, now time.Time

			if tt.fields.timePlacedOnOverflowShelfString != "" {
				timePlacedOnOverflowShelf, _ = time.Parse(timeFormat, tt.fields.timePlacedOnOverflowShelfString)
			}

			if tt.fields.timePlacedOnPrimaryShelfString != "" {
				timePlacedOnPrimaryShelf, _ = time.Parse(timeFormat, tt.fields.timePlacedOnPrimaryShelfString)
			}

			now = minTime(timePlacedOnOverflowShelf, timePlacedOnPrimaryShelf).Add(common.Seconds(tt.fields.secondsSinceShelving))

			s := shelflife.OrderState{
				Order:                     &common.Order{ID: uuid.New(), ShelfLife: tt.fields.shelfLife, DecayRate: tt.fields.decayRate},
				TimePlacedOnPrimaryShelf:  timePlacedOnPrimaryShelf,
				TimePlacedOnOverflowShelf: timePlacedOnOverflowShelf,
			}

			gotValue, err := s.Value(now)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantValue, gotValue)
			}
		})
	}
}

func TestRun0(t *testing.T) {
	t.Run("Order state recorded when the order is shelved to primary", func(t *testing.T) {
		ps, shelvedCh, reShelvedCh, pickupCh, stopCh := initRun()
		ps.On("Pub", mock.Anything, mock.Anything)

		go shelflife.Run0(ps, shelvedCh, reShelvedCh, pickupCh, stopCh)
		defer func() { stopCh <- true }()

		// Order not yet recorded
		require.Nil(t, shelflife.OrderStates[testOrder.ID])
		timeShelved := time.Now()
		// Shelve it
		shelvedCh <- &common.ShelvedEvent{Dt: timeShelved, Order: testOrder, Shelf: testOrder.Temp}
		time.Sleep(common.Seconds(common.SchedulerDelay))
		require.NotNil(t, shelflife.OrderStates[testOrder.ID])
		require.Equal(t,
			shelflife.OrderState{
				TimePlacedOnPrimaryShelf: timeShelved,
				Order:                    &testOrder,
				Shelf:                    testOrder.Temp,
			},
			*shelflife.OrderStates[testOrder.ID])
	})
	t.Run("Order state recorded when the order is shelved to overflow", func(t *testing.T) {
		ps, shelvedCh, reShelvedCh, pickupCh, stopCh := initRun()
		ps.On("Pub", mock.Anything, mock.Anything)

		go shelflife.Run0(ps, shelvedCh, reShelvedCh, pickupCh, stopCh)
		defer func() { stopCh <- true }()

		// Order not yet recorded
		require.Nil(t, shelflife.OrderStates[testOrder.ID])
		timeShelved := time.Now()
		// Shelve it
		shelvedCh <- &common.ShelvedEvent{Dt: timeShelved, Order: testOrder, Shelf: "overflow"}
		time.Sleep(common.Seconds(common.SchedulerDelay))
		require.NotNil(t, shelflife.OrderStates[testOrder.ID])
		require.Equal(t,
			shelflife.OrderState{
				TimePlacedOnOverflowShelf: timeShelved,
				Order:                     &testOrder,
				Shelf:                     "overflow",
			},
			*shelflife.OrderStates[testOrder.ID])
	})
}

func initRun() (ps *mocks.MockPubsub, shelvedCh chan interface{}, reShelvedCh chan interface{}, pickupCh chan interface{}, stopCh chan bool) {
	ps = &mocks.MockPubsub{}
	shelvedCh = make(chan interface{})
	reShelvedCh = make(chan interface{})
	pickupCh = make(chan interface{})
	stopCh = make(chan bool)
	// Start every test with an empty states store.
	shelflife.ResetStates()
	return
}

func minTime(t1 time.Time, t2 time.Time) time.Time {
	if t1.IsZero() {
		return t2
	}
	if t2.IsZero() {
		return t1
	}
	if t1.Before(t2) {
		return t1
	}
	return t2
}
