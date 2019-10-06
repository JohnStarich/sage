package ledger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAccountNode(t *testing.T) {
	for _, tc := range []struct {
		description string
		entries     [][]string
		expected    accountNode
	}{
		{
			description: "no entries",
			expected:    accountNode{},
		},
		{
			description: "empty entry",
			entries:     [][]string{{}},
			expected:    accountNode{},
		},
		{
			description: "one entry",
			entries:     [][]string{{"A", "B"}},
			expected:    accountNode{"A": accountNode{"B": accountNode{}}},
		},
		{
			description: "two joined entries",
			entries:     [][]string{{"A", "B"}, {"A", "C"}},
			expected: accountNode{"A": accountNode{
				"B": accountNode{},
				"C": accountNode{},
			}},
		},
		{
			description: "two disjoint entries",
			entries:     [][]string{{"A", "B"}, {"C", "D"}},
			expected: accountNode{
				"A": accountNode{"B": accountNode{}},
				"C": accountNode{"D": accountNode{}},
			},
		},
		{
			description: "prefixed entries 1",
			entries:     [][]string{{"A", "B", "C"}, {"A", "B"}},
			expected: accountNode{
				"A": accountNode{"B": accountNode{}},
			},
		},
		{
			description: "prefixed entries 2",
			entries:     [][]string{{"A", "B"}, {"A", "B", "C"}},
			expected: accountNode{
				"A": accountNode{"B": accountNode{}},
			},
		},
		{
			description: "many entries",
			entries: [][]string{
				{"A", "B", "D", "E"},
				{"A", "B", "C"},
				{"E", "F", "A"},
			},
			expected: accountNode{
				"A": accountNode{"B": accountNode{
					"C": accountNode{},
					"D": accountNode{"E": accountNode{}},
				}},
				"E": accountNode{"F": accountNode{"A": accountNode{}}},
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expected, newAccountNode(tc.entries))
		})
	}
}

func TestAccountNodeHasPrefixTo(t *testing.T) {
	for _, tc := range []struct {
		entries   [][]string
		testEntry []string
		expected  bool
	}{
		{
			entries:   [][]string{{"A"}},
			testEntry: []string{"A"},
			expected:  true,
		},
		{
			entries:   [][]string{{"A"}},
			testEntry: []string{"B"},
			expected:  false,
		},
		{
			entries:   [][]string{{"A"}, {"B"}},
			testEntry: []string{"B"},
			expected:  true,
		},
		{
			entries:   [][]string{{"A"}, {"B", "C"}},
			testEntry: []string{"B"},
			expected:  false,
		},
		{
			entries:   [][]string{{"A"}, {"B", "C"}},
			testEntry: []string{"B", "C"},
			expected:  true,
		},
		{
			entries:   [][]string{{"A"}, {"B", "C"}},
			testEntry: []string{"B", "C", "D"},
			expected:  true,
		},
		{
			entries:   [][]string{{"A", "D"}, {"B", "C"}},
			testEntry: []string{"A", "E"},
			expected:  false,
		},
		{
			entries:   [][]string{{"A"}, {"B", "C"}},
			testEntry: []string{"D", "E"},
			expected:  false,
		},
	} {
		t.Run("", func(t *testing.T) {
			t.Log("Entries:   ", tc.entries)
			t.Log("Test entry:", tc.testEntry)
			assert.Equal(t, tc.expected, newAccountNode(tc.entries).HasPrefixTo(tc.testEntry))
		})
	}
}
