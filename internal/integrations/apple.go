package integrations

import "github.com/ddvk/rmfakecloud/internal/model"

// newApple builds a calendar integration backed by an Apple iCloud published
// calendar.
//
// iCloud exposes a shared calendar as a read-only ICS feed advertised with a
// webcal:// URL ("Public Calendar" sharing in the Calendar app). Fetching and
// parsing that feed is identical to any other ICS source, so the Apple provider
// reuses the ICS implementation; newICS normalises the webcal(s):// scheme to
// https:// so the URL copied straight from iCloud works as-is.
//
// Authenticated CalDAV against a private iCloud account (app-specific password)
// would require an additional dependency and is intentionally out of scope here.
func newApple(cfg model.IntegrationConfig) *icsIntegration {
	return newICS(cfg)
}
