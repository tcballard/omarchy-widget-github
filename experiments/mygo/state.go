package main

import "time"

// Owned exclusively by the UI thread. Completions carry a generation so a
// canceled request cannot overwrite a newly chosen account or a closed window.
type dataState struct {
	calendar   calendar
	loading    bool
	err        string
	updated    time.Time
	generation uint64
}

func (s *dataState) begin(login string) uint64 {
	s.generation++
	if s.calendar.Username != login {
		s.calendar = calendar{Username: login}
		s.updated = time.Time{}
	}
	s.loading, s.err = true, ""
	return s.generation
}

func (s *dataState) complete(generation uint64, result calendar, err error, now time.Time) {
	if generation != s.generation {
		return
	}
	s.loading = false
	if err != nil {
		s.err = err.Error()
		return
	}
	s.calendar, s.updated, s.err = result, now, ""
}

func (s *dataState) status(now time.Time) string {
	if s.loading {
		return "Loading public contributions…"
	}
	if s.err != "" {
		if len(s.calendar.Days) > 0 {
			return "Stale · last successful graph retained"
		}
		return "Contributions unavailable"
	}
	if s.updated.IsZero() {
		return "Public GitHub contributions · no sign-in required"
	}
	if now.Sub(s.updated) >= 15*time.Minute {
		return "Refresh available · last fetched " + s.updated.Format("15:04")
	}
	return "Public contributions · fetched " + s.updated.Format("15:04")
}
