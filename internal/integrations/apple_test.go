package integrations

import (
	"testing"

	"github.com/ddvk/rmfakecloud/internal/model"
)

func TestNormalizeCalendarURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"webcal", "webcal://p1-caldav.icloud.com/published/2/abc", "https://p1-caldav.icloud.com/published/2/abc"},
		{"webcals", "webcals://p1-caldav.icloud.com/published/2/abc", "https://p1-caldav.icloud.com/published/2/abc"},
		{"https untouched", "https://example.com/calendar.ics", "https://example.com/calendar.ics"},
		{"http untouched", "http://example.com/calendar.ics", "http://example.com/calendar.ics"},
		{"empty", "", ""},
		{"scheme only in path untouched", "https://example.com/webcal://x", "https://example.com/webcal://x"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeCalendarURL(tc.in); got != tc.want {
				t.Errorf("normalizeCalendarURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNewAppleNormalizesWebcal(t *testing.T) {
	i := newApple(model.IntegrationConfig{
		Provider: AppleProvider,
		Address:  "webcal://p1-caldav.icloud.com/published/2/abc",
	})
	if i.url != "https://p1-caldav.icloud.com/published/2/abc" {
		t.Errorf("apple integration url = %q, want normalized https URL", i.url)
	}
}

func TestAppleProviderMetadata(t *testing.T) {
	if got := ProviderType(AppleProvider); got != "Calendar" {
		t.Errorf("ProviderType(apple) = %q, want Calendar", got)
	}
	if got := fixProviderName(AppleProvider); got != "AppleCalendar" {
		t.Errorf("fixProviderName(apple) = %q, want AppleCalendar", got)
	}
}
