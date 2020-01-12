package budget

import (
	"strings"
	"testing"

	"github.com/johnstarich/sage/plaindb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgrade(t *testing.T) {
	for _, tc := range []struct {
		description string
		input       string
		expected    string
		err         string
	}{
		{
			description: "happy path",
			input:       `{}`,
			expected: `
{
	"Version": "2",
	"Data": {}
}`,
		},
		{
			description: "bad version",
			input: `
{
	"Version": "blah",
	"Data": {
		"some key": "some value"
	}
}`,
			err: `Unsupported version: "blah"`,
		},
		{
			description: "v1 budget",
			input: `
{
	"Version": "1",
	"Data": {
		"expenses": {
			"Account": "expenses",
			"Budget": 10.0
		},
		"revenues": {
			"Account": "revenues",
			"Budget": 15.2
		}
	}
}`,
			expected: `
{
	"Version": "2",
	"Data": {
		"2019": {
			"BudgetYear": 2019,
			"Months": {
				"1": {
					"expenses": "10",
					"revenues": "15.2"
				}
			}
		}
	}
}`,
		},
		{
			description: "v2 budget",
			input: `
{
	"Version": "2",
	"Data": {
		"2020": {
			"BudgetYear": 2020,
			"Months": {
				"2": {
					"expenses": "10",
					"revenues": "15.2"
				}
			}
		}
	}
}`,
			expected: `
{
	"Version": "2",
	"Data": {
		"2020": {
			"BudgetYear": 2020,
			"Months": {
				"2": {
					"expenses": "10",
					"revenues": "15.2"
				}
			}
		}
	}
}`,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			db := plaindb.NewMockDB(plaindb.MockConfig{
				FileReader: func(string) ([]byte, error) {
					return []byte(tc.input), nil
				},
			})

			store, err := NewStore(db)
			if tc.err != "" {
				require.Error(t, err)
				assert.Equal(t, tc.err, err.Error())
				return
			}
			require.NoError(t, err)
			dump := db.Dump(store.bucket)
			dump = strings.TrimSpace(dump)
			dump = strings.ReplaceAll(dump, "    ", "\t")
			assert.Equal(t, strings.TrimSpace(tc.expected), dump)
		})
	}
}
