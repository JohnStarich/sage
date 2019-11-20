package pipe

// Op is the common pipe operation. Can be composed into Ops and run as a single unit
type Op interface {
	Do() error
}

// OpFunc makes it easy to wrap an anonymous function into an Op
type OpFunc func() error

// Do implements the Op interface
func (o OpFunc) Do() error {
	return o()
}

// Ops can run a slice of Op's in series, stopping on the first error
type Ops []Op

// Do implements the Op interface
func (ops Ops) Do() error {
	for _, op := range ops {
		if err := op.Do(); err != nil {
			return err
		}
	}
	return nil
}

// OpFuncs makes it easy to wrap a slice of functions into an Op
type OpFuncs []func() error

// Do implements the Op interface
func (ops OpFuncs) Do() error {
	for _, op := range ops {
		if err := op(); err != nil {
			return err
		}
	}
	return nil
}
