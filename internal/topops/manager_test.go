package topops

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseContainerList(t *testing.T) {
	raw := `{"ID":"abc123def456","Names":"svc-1","Image":"img","State":"running","Status":"Up 1 second"}
{"ID":"789abc456def","Names":"svc-2","Image":"img2","State":"exited","Status":"Exited (0)"}
`

	items, err := parseContainerList(raw)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "svc-1", items[0].Names)
	require.Equal(t, "exited", items[1].State)
}

func TestParseInspect(t *testing.T) {
	raw := `{"Id":"abc123def4569999","HostConfig":{"Runtime":"runc","Annotations":{"k":"v"}}}
`
	byID, err := parseInspect(raw)
	require.NoError(t, err)
	require.Contains(t, byID, "abc123def456")
	require.Equal(t, "runc", byID["abc123def456"].HostConfig.Runtime)
}

func TestSanitizePathComponent(t *testing.T) {
	require.Equal(t, "foo_bar baz", sanitizePathComponent(" foo/bar\tbaz "))
	require.Equal(t, "container", sanitizePathComponent(" \n\t "))
}
