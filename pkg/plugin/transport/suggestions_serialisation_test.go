package transport

import (
	"testing"

	"github.com/arm/topo/pkg/plugin/suggestions"
	"github.com/arm/topo/pkg/plugin/transport/pluginpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestSerialiseDeserialiseSuggestionRequest(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		value *string
	}{
		{name: "absent current value", value: nil},
		{name: "explicitly empty current value", value: new("")},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			request := suggestions.Request{CurrentValue: scenario.value}
			var wire pluginpb.SuggestionRequest

			encoded, err := proto.Marshal(serialiseSuggestionRequest(request))
			require.NoError(t, err)
			require.NoError(t, proto.Unmarshal(encoded, &wire))
			got := deserialiseSuggestionRequest(&wire)

			assert.Equal(t, request, got)
		})
	}
}
