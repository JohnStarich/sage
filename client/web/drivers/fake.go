package drivers

// TODO delete this fake driver once a second real driver exists

func init() {
	Register("fake", func(connector Connector) (Requestor, error) {
		return nil, nil
	})
}
