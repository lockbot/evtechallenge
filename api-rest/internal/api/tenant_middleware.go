package api

import (
	"net/http"
	"strings"
)

// TenantChannelMiddleware routes requests through tenant channels if available
func TenantChannelMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, err := GetTenantFromRequest(r)
		if err != nil {
			// If no tenant ID, fallback to direct processing
			next.ServeHTTP(w, r)
			return
		}

		if channels, exists := GetTenantChannels(tenantID); exists {
			// Route through tenant channels instead of direct DB calls
			switch r.URL.Path {
			case "/encounters":
				channels.listEncountersCh <- struct{ entity, id string }{"Encounter", ""}
			case "/encounters/":
				id := extractID(r.URL.Path)
				channels.getEncounterCh <- struct{ entity, id string }{"Encounter", id}
			case "/patients":
				channels.listPatientsCh <- struct{ entity, id string }{"Patient", ""}
			case "/patients/":
				id := extractID(r.URL.Path)
				channels.getPatientCh <- struct{ entity, id string }{"Patient", id}
			case "/practitioners":
				channels.listPractitionersCh <- struct{ entity, id string }{"Practitioner", ""}
			case "/practitioners/":
				id := extractID(r.URL.Path)
				channels.getPractitionerCh <- struct{ entity, id string }{"Practitioner", id}
			case "/review-request":
				channels.reviewCh <- struct{ entity, id string }{"Review", ""}
			default:
				// For other routes, fallback to direct processing
				next.ServeHTTP(w, r)
				return
			}
		} else {
			// Fallback to direct DB calls if tenant not warmed up
			next.ServeHTTP(w, r)
		}
	})
}

// extractID extracts the ID from a path like "/encounters/123"
func extractID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
