package restclient

import (
	"net/url"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestURL(t *testing.T) {
	u, err := url.Parse("http://test.com//test")

	require.NoError(t, err)

	u.Path = filepath.Clean(u.Path)

	println(u.String())
}
