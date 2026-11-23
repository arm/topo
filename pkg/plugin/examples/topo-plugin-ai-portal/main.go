// This example opens AI Portal only in response to an explicitly selected action.
package main

import (
	"os"

	sdk "github.com/arm/topo/pkg/plugin"
)

func main() {
	portalURL := os.Getenv("TOPO_AI_PORTAL_URL")
	if portalURL == "" {
		portalURL = "http://localhost:3000/ai/models"
	}
	sdk.ServeSuggestions("example.ai_portal", Provider{PortalURL: portalURL})
}
