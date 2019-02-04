package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestE2E(t *testing.T) {
	RunE2ETestSuit(t)
}

func TestDoGRPC(t *testing.T) {
	err := doGRPC("127.0.0.1:3001", "")
	assert.Equal(t, nil, err)
}
