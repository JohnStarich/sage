package ledger

type accountNode map[string]accountNode

func newAccountNode(entries [][]string) accountNode {
	root := make(accountNode)
	for _, entry := range entries {
		node := root
		for i, segment := range entry {
			if node[segment] != nil && len(node[segment]) == 0 {
				// if encountering an empty map, then we've hit a leaf node of a previous entry
				break
			}
			if node[segment] == nil || i+1 == len(entry) {
				node[segment] = make(accountNode)
			}
			node = node[segment]
		}
	}
	return root
}

func (n accountNode) HasPrefixTo(entry []string) bool {
	for i := range entry {
		if len(n) == 0 {
			return n != nil
		}
		n = n[entry[i]]
	}
	return len(n) == 0 && n != nil
}
