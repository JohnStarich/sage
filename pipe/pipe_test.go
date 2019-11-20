package pipe

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpFuncDo(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		op := OpFunc(func() error {
			return nil
		})
		assert.NoError(t, op.Do())
	})

	t.Run("error", func(t *testing.T) {
		someErr := errors.New("some error")
		op := OpFunc(func() error {
			return someErr
		})
		assert.Equal(t, someErr, op.Do())
	})
}

func TestOpsDo(t *testing.T) {
	nilOp := OpFunc(func() error {
		return nil
	})
	someErr := errors.New("some error")
	errOp := OpFunc(func() error {
		return someErr
	})

	detectOp := func(ran *bool) Op {
		return OpFunc(func() error {
			*ran = true
			return nil
		})
	}

	t.Run("no errors", func(t *testing.T) {
		var ranLast bool
		assert.NoError(t, Ops{nilOp, nilOp, nilOp, detectOp(&ranLast)}.Do())
		assert.True(t, ranLast)
	})

	t.Run("stops on first error", func(t *testing.T) {
		var ranAfterError bool
		assert.Equal(t, someErr, Ops{nilOp, errOp, detectOp(&ranAfterError), nilOp}.Do())
		assert.False(t, ranAfterError)
	})
}

func TestOpFuncsDo(t *testing.T) {
	nilOp := func() error {
		return nil
	}
	someErr := errors.New("some error")
	errOp := func() error {
		return someErr
	}

	detectOp := func(ran *bool) func() error {
		return func() error {
			*ran = true
			return nil
		}
	}

	t.Run("no errors", func(t *testing.T) {
		var ranLast bool
		assert.NoError(t, OpFuncs{nilOp, nilOp, nilOp, detectOp(&ranLast)}.Do())
		assert.True(t, ranLast)
	})

	t.Run("stops on first error", func(t *testing.T) {
		var ranAfterError bool
		assert.Equal(t, someErr, OpFuncs{nilOp, errOp, detectOp(&ranAfterError), nilOp}.Do())
		assert.False(t, ranAfterError)
	})
}
