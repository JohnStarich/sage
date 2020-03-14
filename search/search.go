package search

import (
	"sort"
	"strings"
)

// score assumes 'name' and 'search' are the same case
func score(name, search string) int {
	if strings.Contains(name, search) {
		return 1
	}
	if matchesInitialism(name, search) {
		return 0
	}
	return -1
}

// matchesInitialism assumes inputs are the same case
func matchesInitialism(name, search string) bool {
	for _, word := range strings.Fields(name) {
		if len(search) == 0 {
			return true
		}
		if word[0] == search[0] {
			search = search[1:]
		}
	}
	return len(search) == 0
}

type scoreItem struct {
	index int
	name  string
	score int
}

func query(names []string, search string) []scoreItem {
	search = strings.ToLower(search)
	scores := make([]scoreItem, 0, len(names))
	for i, name := range names {
		s := score(strings.ToLower(name), search)
		if s >= 0 {
			scores = append(scores, scoreItem{
				index: i,
				name:  name,
				score: s,
			})
		}
	}
	sort.SliceStable(scores, func(a, b int) bool {
		return scores[a].score > scores[b].score // sort scores largest to smallest
	})
	return scores
}

func Query(names []string, search string) []string {
	scores := query(names, search)
	results := make([]string, len(scores))
	for i, item := range scores {
		results[i] = item.name
	}
	return results
}

func QueryIndexes(names []string, search string) []int {
	scores := query(names, search)
	results := make([]int, len(scores))
	for i, item := range scores {
		results[i] = item.index
	}
	return results
}
