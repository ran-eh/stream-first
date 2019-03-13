package screen_test

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"stream-first/mocks"
	"stream-first/ui/screen"
	"testing"
)

// Unit tests for display is intentionally incomplete as per the Readme doc.  The tests below are
// regressions for issues fixed during debugging.

func Test_Shelf(t *testing.T) {
	var ps = &mocks.MockPubsub{}
	t.Run("Add to first slot for empty shelf", func(t *testing.T) {
		s := screen.NewShelfState(ps)
		id1 := uuid.New()
		s.Add(id1)
		require.Equal(t, 1, len(s.DisplayPositionToOrderID))
		require.NotNil(t, s.DisplayPositionToOrderID[0])
		require.Equal(t, id1, *s.DisplayPositionToOrderID[0])

		require.Equal(t, 1, len(s.OrderIDToDisplayPosition))
		require.NotNil(t, *s.OrderIDToDisplayPosition[id1])
		require.Equal(t, 0, *s.OrderIDToDisplayPosition[id1])
	})
	t.Run("Remove leaves gap that is filled by next add", func(t *testing.T) {
		s := screen.NewShelfState(ps)
		id1, id2, id3, id4 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		s.Add(id1)
		s.Add(id2)
		s.Add(id3)
		require.Equal(t, 3, len(s.DisplayPositionToOrderID))
		require.NotNil(t, s.DisplayPositionToOrderID[1])
		require.Equal(t, id2, *s.DisplayPositionToOrderID[1])

		require.Equal(t, 3, len(s.OrderIDToDisplayPosition))
		require.NotNil(t, s.OrderIDToDisplayPosition[id2])
		require.Equal(t, 1, *s.OrderIDToDisplayPosition[id2])

		s.Remove(id2)
		require.Equal(t, 3, len(s.DisplayPositionToOrderID))
		require.Nil(t, s.DisplayPositionToOrderID[1])

		require.Equal(t, 2, len(s.OrderIDToDisplayPosition))
		require.Nil(t, s.OrderIDToDisplayPosition[id2])

		s.Add(id4)
		require.Equal(t, 3, len(s.DisplayPositionToOrderID))
		require.NotNil(t, s.DisplayPositionToOrderID[1])
		require.Equal(t, id4, *s.DisplayPositionToOrderID[1])

		require.Equal(t, 3, len(s.OrderIDToDisplayPosition))
		require.NotNil(t, s.OrderIDToDisplayPosition[id4])
		require.Equal(t, 1, *s.OrderIDToDisplayPosition[id4])
	})
	t.Run("Add two orders and removing one works as expected", func(t *testing.T) {
		s := screen.NewShelfState(ps)
		id1, id2 := uuid.UUID{1}, uuid.UUID{2}
		s.Add(id1)
		s.Add(id2)
		require.Equal(t, 2, len(s.DisplayPositionToOrderID))
		require.NotNil(t, s.DisplayPositionToOrderID[1])
		require.Equal(t, id2, *s.DisplayPositionToOrderID[1])

		require.Equal(t, 2, len(s.OrderIDToDisplayPosition))
		require.NotNil(t, s.OrderIDToDisplayPosition[id2])
		require.Equal(t, 1, *s.OrderIDToDisplayPosition[id2])

		s.Remove(id2)
		require.Equal(t, 2, len(s.DisplayPositionToOrderID))
		require.Nil(t, s.DisplayPositionToOrderID[1])

		require.Equal(t, 1, len(s.OrderIDToDisplayPosition))
		require.Nil(t, s.OrderIDToDisplayPosition[id2])

		s.Remove(id1)
		require.Equal(t, 2, len(s.DisplayPositionToOrderID))
		require.Nil(t, s.DisplayPositionToOrderID[0])

		require.Equal(t, 0, len(s.OrderIDToDisplayPosition))
	})
}
