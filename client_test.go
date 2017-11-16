package crest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	baseURL := "base URL"
	c := NewClient(baseURL)
	cImpl, ok := c.(*client)
	require.True(t, ok)
	require.Equal(t, baseURL, cImpl.baseURL)
	require.NoError(t, c.Error())
}
