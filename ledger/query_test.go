package ledger

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaginateFromEnd(t *testing.T) {
	for _, tc := range []struct {
		page, results, size int
		start, end          int
	}{
		{page: 1, results: 10, size: 10, start: 0, end: 10},
		{page: 1, results: 5, size: 10, start: 5, end: 10},
		{page: 2, results: 5, size: 10, start: 0, end: 5},

		{page: 1, results: 10, size: 5, start: 0, end: 5},
		{page: 2, results: 2, size: 3, start: 0, end: 1},

		{page: 3, results: 1, size: 1, start: 0, end: 0},
	} {
		t.Run(fmt.Sprintf("page %d results %d size %d", tc.page, tc.results, tc.size), func(t *testing.T) {
			start, end := paginateFromEnd(tc.page, tc.results, tc.size)
			require.Truef(t, tc.start >= 0, "Test case start must be greater than or equal to 0: %d", tc.start)
			require.Truef(t, tc.end >= tc.start, "Test case end must be greater than or equal to start: end=%d, start=%d", tc.end, tc.start)
			require.Truef(t, tc.end <= tc.size, "Test case end must be less than or equal to size: end=%d, size=%d", tc.end, tc.size)

			assert.Equal(t, tc.start, start, "Incorrect start")
			assert.Equal(t, tc.end, end, "Incorrect end")
		})
	}
}
