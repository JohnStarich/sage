package client

import "github.com/aclindsa/ofxgo"

var (
	clientCache = make(map[string]ofxgo.Client)
)

// clientForInstitution returns a cached client. This helps preserve and enforce rate limits.
func clientForInstitution(institution Institution) (ofxgo.Client, error) {
	url := institution.URL()
	if client, ok := clientCache[url]; ok {
		return client, nil
	}

	client, err := New(url, institution.Config())
	if err != nil {
		return nil, err
	}
	clientCache[url] = client
	return client, nil
}
