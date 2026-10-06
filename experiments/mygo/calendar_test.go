package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func fragment(start time.Time, n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		fmt.Fprintf(&b, `<td id="contribution-day-component-%d" data-date="%s" data-level="1"></td><tool-tip for="contribution-day-component-%d">1 contribution on a day.</tool-tip>`, i, date, i)
	}
	return b.String()
}

func TestCalendarProjection(t *testing.T) {
	for _, n := range []int{365, 366} {
		body := fragment(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), n)
		c, err := parseCalendar([]byte(body), "tcballard")
		if err != nil || c.Total != n || len(c.Days) != n || c.Days[0].Weekday != 1 {
			t.Fatalf("%d: %+v %v", n, c, err)
		}
		for _, family := range []string{"small", "medium", "large"} {
			seen := map[string]bool{}
			parts := sections(c.Days, family)
			if family == "large" && len(parts) != 2 {
				t.Fatal("large must split the year")
			}
			if family == "small" && len(parts[0]) != 13 {
				t.Fatal("small must show 13 weeks")
			}
			for _, part := range parts {
				for _, week := range part {
					for row, d := range week {
						if d == nil {
							continue
						}
						if row != d.Weekday || seen[d.Date] {
							t.Fatal("misaligned or duplicate day")
						}
						seen[d.Date] = true
					}
				}
			}
			if family != "small" && len(seen) != n {
				t.Fatal("lost days")
			}
		}
	}
}

func TestMalformedCalendarFailsClosed(t *testing.T) {
	body := fragment(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 365)
	cases := map[string]string{
		"short":               fragment(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 364),
		"oversized":           strings.Repeat(" ", maxCalendarBytes+1),
		"invalid date":        strings.Replace(body, "2025-01-01", "2025-02-30", 1),
		"duplicate date":      strings.Replace(body, "2025-01-02", "2025-01-01", 1),
		"gap":                 strings.Replace(body, "2025-12-31", "2026-01-01", 1),
		"missing tooltip":     strings.Replace(body, `for="contribution-day-component-0"`, `for="other"`, 1),
		"bad count":           strings.Replace(body, "1 contribution", "unknown contribution", 1),
		"huge count":          strings.Replace(body, "1 contribution", "1,000,001 contributions", 1),
		"bad level":           strings.Replace(body, `data-level="1"`, `data-level="5"`, 1),
		"zero mismatch":       strings.Replace(body, `data-level="1"`, `data-level="0"`, 1),
		"duplicate attribute": strings.Replace(body, `data-level="1"`, `data-level="1" data-level="2"`, 1),
		"duplicate tooltip":   body + `<tool-tip for="contribution-day-component-0">1 contribution</tool-tip>`,
	}
	for name, b := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseCalendar([]byte(b), "test"); err == nil {
				t.Fatal("accepted malformed response")
			}
		})
	}
	zero := strings.ReplaceAll(strings.ReplaceAll(body, `data-level="1"`, `data-level="0"`), "1 contribution", "No contributions")
	c, err := parseCalendar([]byte(zero), "test")
	if err != nil || c.Total != 0 || len(c.Days) != 365 {
		t.Fatal("valid zero calendar rejected", err)
	}
}

func TestSundayPaddedRollingYear(t *testing.T) {
	start := time.Date(2025, 10, 5, 0, 0, 0, 0, time.UTC)
	for n := 365; n <= 371; n++ {
		c, err := parseCalendar([]byte(fragment(start, n)), "test")
		if err != nil || len(c.Days) != n {
			t.Fatalf("%d-day Sunday-padded year: %v", n, err)
		}
		parts := sections(c.Days, "medium")
		if len(parts[0]) != 53 {
			t.Fatalf("expected 53 weeks for %d days", n)
		}
	}
	for _, body := range []string{fragment(start, 372), fragment(start.AddDate(0, 0, 1), 367)} {
		if _, err := parseCalendar([]byte(body), "test"); err == nil {
			t.Fatal("accepted invalid padded calendar")
		}
	}
}

func TestUsernames(t *testing.T) {
	for _, s := range []string{"", "-name", "name-", "a--b", "a/b", "a?x", "a\n", "é", strings.Repeat("a", 40)} {
		if _, err := username(s); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	if s, err := username("TC-Ballard"); err != nil || s != "tc-ballard" {
		t.Fatal(s, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTransportBoundsAndIdentity(t *testing.T) {
	for _, status := range []int{200, 403, 404, 429, 500} {
		client := publicClient()
		client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://github.com/users/tcballard/contributions" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
				t.Fatal("wrong endpoint or credentials")
			}
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(fragment(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 365)))}, nil
		})
		_, err := fetchCalendar(context.Background(), client, "TCBALLARD")
		if (err == nil) != (status == 200) {
			t.Fatalf("HTTP %d: %v", status, err)
		}
	}
	client := publicClient()
	client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://example.com/"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	if _, err := fetchCalendar(context.Background(), client, "tcballard"); err == nil {
		t.Fatal("followed redirect")
	}
	client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", maxCalendarBytes+1)))}, nil
	})
	if _, err := fetchCalendar(context.Background(), client, "tcballard"); err == nil {
		t.Fatal("accepted oversized HTTP body")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })
	if _, err := fetchCalendar(ctx, client, "tcballard"); !errors.Is(err, context.Canceled) {
		t.Fatal("lost cancellation", err)
	}
}

func TestStaleDataAndSupersededRequests(t *testing.T) {
	var s dataState
	now := time.Now()
	first := s.begin("demo")
	c := demoCalendar()
	s.complete(first, c, nil, now)
	second := s.begin("demo")
	s.complete(second, calendar{}, errors.New("offline"), now)
	if len(s.calendar.Days) != 365 || !strings.HasPrefix(s.status(now), "Stale") {
		t.Fatal("lost cached calendar")
	}
	third := s.begin("other")
	s.complete(second, c, nil, now)
	if s.calendar.Username != "other" || len(s.calendar.Days) != 0 || !s.loading {
		t.Fatal("old request overwrote new account")
	}
	s.complete(third, calendar{}, errors.New("offline"), now)
	if s.status(now) != "Contributions unavailable" {
		t.Fatal("wrong unavailable state")
	}
}

func TestCapturedPublicCalendar(t *testing.T) {
	path := os.Getenv("GITHUB_CALENDAR_FIXTURE")
	if path == "" {
		t.Skip("optional captured public fragment")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	c, err := parseCalendar(b, "tcballard")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("captured public calendar: %d days, %d contributions", len(c.Days), c.Total)
}
