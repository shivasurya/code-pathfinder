package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallable(t *testing.T) {
	t.Run("New Callable with name", func(t *testing.T) {
		callable := &Callable{
			CallableName: "testFunction",
		}
		assert.Equal(t, "testFunction", callable.CallableName)
	})

	t.Run("Empty Callable name", func(t *testing.T) {
		callable := &Callable{}
		assert.Equal(t, "", callable.CallableName)
	})

	t.Run("Callable with special characters", func(t *testing.T) {
		callable := &Callable{
			CallableName: "test$Function_123",
		}
		assert.Equal(t, "test$Function_123", callable.CallableName)
	})
}
