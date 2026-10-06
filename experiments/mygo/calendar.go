package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// This projection follows Widget Core's src/github.rs at 70712a403d65.
// The public fragment is not a stable GitHub API: reject incomplete data.
const maxCalendarBytes = 524288

type day struct {
	Date         string
	Count, Level int
	Weekday      int
}

type calendar struct {
	Username string
	Days     []day
	Total    int
}

var usernameRE = regexp.MustCompile(`^[A-Za-z0-9]+(-[A-Za-z0-9]+)*$`)
var tipRE = regexp.MustCompile(`(?s)<tool-tip\s+([^>]+)>(.*?)</tool-tip>`)
var cellRE = regexp.MustCompile(`<td\s+([^>]+)>`)
var attributeRE = regexp.MustCompile(`(?:^|\s)([a-zA-Z-]+)="([^"]*)"`)
var countRE = regexp.MustCompile(`^(No|[0-9]+(?:,[0-9]{3})*) contributions?\b`)

func username(s string) (string, error) {
	if len(s) > 39 || !usernameRE.MatchString(s) {
		return "", fmt.Errorf("enter a GitHub username using letters, digits and single hyphens")
	}
	return strings.ToLower(s), nil
}

func attributes(s string) (map[string]string, error) {
	m := map[string]string{}
	for _, match := range attributeRE.FindAllStringSubmatch(s, -1) {
		if _, exists := m[match[1]]; exists {
			return nil, fmt.Errorf("duplicate calendar attribute")
		}
		m[match[1]] = match[2]
	}
	return m, nil
}

func parseCalendar(body []byte, login string) (calendar, error) {
	result := calendar{Username: login}
	if len(body) > maxCalendarBytes {
		return result, fmt.Errorf("calendar exceeds size limit")
	}
	counts := map[string]int{}
	for _, tip := range tipRE.FindAllSubmatch(body, -1) {
		a, err := attributes(string(tip[1]))
		if err != nil {
			return result, err
		}
		id := a["for"]
		if !strings.HasPrefix(id, "contribution-day-component-") {
			continue
		}
		m := countRE.FindStringSubmatch(strings.TrimSpace(string(tip[2])))
		if m == nil {
			return result, fmt.Errorf("invalid contribution count")
		}
		count := 0
		if m[1] != "No" {
			count, err = strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
			if err != nil || count > 1000000 {
				return result, fmt.Errorf("invalid contribution count")
			}
		}
		if _, exists := counts[id]; exists {
			return result, fmt.Errorf("duplicate contribution count")
		}
		counts[id] = count
	}
	seen := map[string]bool{}
	seenIDs := map[string]bool{}
	for _, cell := range cellRE.FindAllSubmatch(body, -1) {
		a, err := attributes(string(cell[1]))
		if err != nil {
			return result, err
		}
		date, exists := a["data-date"]
		if !exists {
			continue
		}
		t, err := time.Parse("2006-01-02", date)
		if err != nil || t.Year() < 1970 || t.Format("2006-01-02") != date {
			return result, fmt.Errorf("invalid calendar date")
		}
		level, err := strconv.Atoi(a["data-level"])
		count, ok := counts[a["id"]]
		if err != nil || level < 0 || level > 4 || !ok || (level == 0) != (count == 0) || seen[date] || seenIDs[a["id"]] {
			return result, fmt.Errorf("invalid or duplicate calendar cell")
		}
		seen[date], seenIDs[a["id"]] = true, true
		result.Days = append(result.Days, day{date, count, level, int(t.Weekday())})
		result.Total += count
	}
	// GitHub pads its rolling year back to Sunday (367 days observed on
	// 2026-10-06). A complete 53-week grid can therefore contain 365–371
	// days. Longer-than-year responses must start on that Sunday boundary.
	if len(result.Days) < 365 || len(result.Days) > 371 {
		return result, fmt.Errorf("incomplete annual contribution calendar")
	}
	sort.Slice(result.Days, func(i, j int) bool { return result.Days[i].Date < result.Days[j].Date })
	first, _ := time.Parse("2006-01-02", result.Days[0].Date)
	if len(result.Days) > 366 && first.Weekday() != time.Sunday {
		return result, fmt.Errorf("invalid padded calendar boundary")
	}
	for i, d := range result.Days {
		if d.Date != first.AddDate(0, 0, i).Format("2006-01-02") {
			return result, fmt.Errorf("contribution calendar has gaps")
		}
	}
	return result, nil
}

func fetchCalendar(ctx context.Context, client *http.Client, login string) (calendar, error) {
	login, err := username(login)
	if err != nil {
		return calendar{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://github.com/users/"+login+"/contributions", nil)
	if err != nil {
		return calendar{}, err
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", "Omarchy-GitHub-MyGo-Experiment/0.1")
	res, err := client.Do(req)
	if err != nil {
		return calendar{}, fmt.Errorf("GitHub request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return calendar{}, fmt.Errorf("GitHub returned HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxCalendarBytes+1))
	if err != nil {
		return calendar{}, fmt.Errorf("reading GitHub response: %w", err)
	}
	return parseCalendar(body, login)
}

func publicClient() *http.Client {
	return &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return fmt.Errorf("GitHub redirected the calendar request")
	}}
}

type week [7]*day

func sections(days []day, family string) [][]week {
	var weeks []week
	for i := range days {
		d := &days[i]
		if len(weeks) == 0 || d.Weekday == 0 {
			weeks = append(weeks, week{})
		}
		weeks[len(weeks)-1][d.Weekday] = d
	}
	if family == "small" && len(weeks) > 13 {
		weeks = weeks[len(weeks)-13:]
	}
	if family == "large" && len(weeks) > 1 {
		split := (len(weeks) + 1) / 2
		return [][]week{weeks[:split], weeks[split:]}
	}
	return [][]week{weeks}
}

func describe(d *day) string {
	if d == nil {
		return "Hover a day, or focus the calendar and use arrow keys"
	}
	noun := "contributions"
	if d.Count == 1 {
		noun = "contribution"
	}
	return fmt.Sprintf("%s %s · %s", number(d.Count), noun, d.Date)
}

func number(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func demoCalendar() calendar {
	c := calendar{Username: "demo"}
	start := time.Date(2025, 10, 6, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 365; i++ {
		t := start.AddDate(0, 0, i)
		level := (i*17 + i/9) % 5
		d := day{t.Format("2006-01-02"), level * 3, level, int(t.Weekday())}
		c.Days = append(c.Days, d)
		c.Total += d.Count
	}
	return c
}
