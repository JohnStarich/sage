package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScore(t *testing.T) {
	for _, tc := range []struct {
		name, search string
		score        int
	}{
		{"university federal credit union", "university federal", 1},
		{"university federal credit union", "ufcu", 0},
		{"university federal credit union", "ufc", 0},
		{"university federal credit union", "uf", 0},
		{"university federal credit union", "ufdu", -1},
		{"university federal credit tribe", "ufcu", -1},
	} {
		t.Run(tc.name+" == "+tc.search, func(t *testing.T) {
			assert.Equal(t, tc.score, score(tc.name, tc.search))
		})
	}
}

func TestQuery(t *testing.T) {
	for _, tc := range []struct {
		description string
		names       []string
		search      string
		expect      []string
	}{
		{
			description: "equal",
			names:       []string{"University Federal Credit Union"},
			search:      "university federal credit union",
			expect:      []string{"University Federal Credit Union"},
		},
		{
			description: "contains",
			names:       []string{"University Federal Credit Union"},
			search:      "university federal",
			expect:      []string{"University Federal Credit Union"},
		},
		{
			description: "not equal",
			names:       []string{"University Federal Credit Union"},
			search:      "university federal credit tribe",
			expect:      []string{},
		},
		{
			description: "initialism",
			names:       []string{"University Federal Credit Union"},
			search:      "ufcu",
			expect:      []string{"University Federal Credit Union"},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expect, Query(tc.names, tc.search))
		})
	}
}

func TestQueryIndexes(t *testing.T) {
	for _, tc := range []struct {
		description   string
		names         []string
		search        string
		expectIndexes []int
	}{
		{
			description: "equal",
			names: []string{
				"University Federal Credit Union",
				"Some Other Federal Credit Union",
			},
			search:        "university federal credit union",
			expectIndexes: []int{0},
		},
		{
			description: "not equal",
			names: []string{
				"University Federal Credit Union",
				"Some Other Federal Credit Union",
			},
			search:        "something else",
			expectIndexes: []int{},
		},
		{
			description: "contains",
			names: []string{
				"University Federal Credit Union",
				"Some Other Federal Credit Union",
			},
			search:        "federal credit union",
			expectIndexes: []int{0, 1},
		},
		{
			description: "equal to second",
			names: []string{
				"University Federal Credit Union",
				"Some Other Federal Credit Union",
			},
			search:        "some other",
			expectIndexes: []int{1},
		},
		{
			description: "initialism",
			names: []string{
				"University Federal Credit Union",
				"Some Other Federal Credit Union",
			},
			search:        "sofcu",
			expectIndexes: []int{1},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expectIndexes, QueryIndexes(tc.names, tc.search))
		})
	}
}
