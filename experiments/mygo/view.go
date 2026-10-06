package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *application) view(c *ui.Context) {
	t := *c.Theme()
	if v := a.theme["background"]; v != "" {
		t.Background = ui.Hex(v)
	}
	if v := a.theme["foreground"]; v != "" {
		t.Text = ui.Hex(v)
	}
	if v := a.theme["accent"]; v != "" {
		t.Accent = ui.Hex(v)
	}
	if v := a.theme["red"]; v != "" {
		t.Danger = ui.Hex(v)
	}
	t.Surface = t.Background.Mix(t.Text, 0.045)
	t.SurfaceHover = t.Background.Mix(t.Text, 0.10)
	t.SurfacePressed = t.Background.Mix(t.Text, 0.16)
	t.TextMuted = t.Background.Mix(t.Text, 0.65)
	t.Border = t.Background.Mix(t.Text, 0.15)
	t.Focus = t.Accent
	t.Selection = t.Accent.Alpha(0.3)
	t.Scrollbar = t.Background.Mix(t.Text, 0.4)
	t.Spacing = 3
	c.SetTheme(&t)
	c.Root().Background(t.Background)
	ui.Scroll(c).Fill().Padding(22).Gap(12).Children(func() {
		ui.Row(c).Gap(10).Children(func() {
			ui.Text(c, "GitHub").FontSize(22).Bold().Grow(1)
			label := "MYGO EXPERIMENT"
			if a.demo {
				label = "DEMO · SYNTHETIC DATA"
			}
			ui.Text(c, label).FontSize(10).TextColor(t.TextMuted)
		})
		ui.Row(c).Gap(8).Children(func() {
			submitted := ui.TextInput(c, &a.input).Label("GitHub username").Placeholder("GitHub username").Grow(1).Disabled(a.demo).Submitted()
			if ui.Button(c, "Load").Disabled(a.demo).Clicked() || submitted {
				a.refresh()
			}
			if ui.Button(c, "Close").Clicked() && a.window != nil {
				a.window.Close()
			}
		})
		ui.Row(c).Gap(5).Children(func() {
			for _, family := range []string{"small", "medium", "large"} {
				b := ui.Button(c, family).Key(family)
				if family == a.family {
					b.Background(t.SurfaceHover).Border(1, t.Accent)
				}
				if b.Clicked() {
					a.family = family
					a.selected = ""
					if a.window != nil {
						w, h := dimensions(family)
						a.window.SetSize(w, h)
					}
				}
			}
			ui.Spacer(c)
			label := "Green"
			if a.palette == "theme" {
				label = "Accent"
			}
			if ui.Button(c, label).Label("Toggle calendar palette").Clicked() {
				if a.palette == "green" {
					a.palette = "theme"
				} else {
					a.palette = "green"
				}
			}
		})
		ui.Column(c).Key("calendar-card").Padding(16).Gap(10).Radius(12).Background(t.Surface).Border(1, t.Border).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				count := "—"
				if len(a.data.calendar.Days) > 0 {
					count = number(a.data.calendar.Total)
				}
				ui.Text(c, count).FontSize(30).Bold()
				ui.Column(c).Gap(2).Children(func() {
					ui.Text(c, "contributions").FontSize(12)
					ui.Text(c, "@"+a.data.calendar.Username).FontSize(11).TextColor(t.TextMuted)
				})
			})
			if len(a.data.calendar.Days) == 0 {
				ui.Box(c).Height(106).Center().Children(func() { ui.Text(c, a.data.status(time.Now())).TextColor(t.TextMuted) })
			} else {
				for i, weeks := range sections(a.data.calendar.Days, a.family) {
					a.heatmap(c, weeks, i)
				}
			}
			var selected *day
			for i := range a.data.calendar.Days {
				if a.data.calendar.Days[i].Date == a.selected {
					selected = &a.data.calendar.Days[i]
					break
				}
			}
			ui.Text(c, describe(selected)).FontSize(11).TextColor(t.TextMuted)
		})
		status := a.data.status(time.Now())
		if a.demo {
			status = "Synthetic calendar · no network requests"
		}
		if a.family == "small" {
			status += " · last 13 weeks"
		}
		ui.Text(c, status).FontSize(11).TextColor(t.TextMuted)
		if a.validation != "" {
			ui.Text(c, a.validation).FontSize(11).TextColor(t.Danger)
		}
		if a.data.err != "" {
			ui.Text(c, a.data.err).FontSize(11).TextColor(t.Danger)
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "Refresh").Disabled(a.demo || a.data.loading).Clicked() {
				a.refresh()
			}
			if ui.Button(c, "Reload theme").Clicked() {
				a.theme = loadTheme()
			}
			ui.Spacer(c)
			if ui.Button(c, "Profile").Disabled(a.demo || a.data.calendar.Username == "").Clicked() {
				c.OpenURL("https://github.com/" + a.data.calendar.Username)
			}
		})
	})
}

func (a *application) heatmap(c *ui.Context, weeks []week, section int) {
	if len(weeks) == 0 {
		return
	}
	t := c.Theme()
	chart := ui.Row(c).Key(fmt.Sprintf("chart-%d-%s", section, a.family)).Gap(2).Height(106).Focusable().Label("Contribution calendar").Description("Arrow keys inspect days; left and right move one week")
	current := -1
	for col, w := range weeks {
		for row, d := range w {
			if d != nil && d.Date == a.selected {
				current = col*7 + row
			}
		}
	}
	for _, key := range []struct {
		key   ui.Key
		delta int
	}{{ui.KeyLeft, -7}, {ui.KeyRight, 7}, {ui.KeyUp, -1}, {ui.KeyDown, 1}} {
		if chart.Shortcut(0, key.key) {
			next := 0
			if current >= 0 {
				next = current + key.delta
			}
			next = max(0, min(len(weeks)*7-1, next))
			direction := 1
			if key.delta < 0 {
				direction = -1
			}
			for next >= 0 && next < len(weeks)*7 && weeks[next/7][next%7] == nil {
				next += direction
			}
			if next >= 0 && next < len(weeks)*7 {
				a.selected = weeks[next/7][next%7].Date
			}
		}
	}
	chart.Children(func() {
		lastMonth := ""
		for col, w := range weeks {
			month := ""
			for _, d := range w {
				if d != nil {
					month = d.Date[:7]
					break
				}
			}
			showMonth := month != "" && month != lastMonth && col < len(weeks)-2
			lastMonth = month
			ui.Column(c).Key(col).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
				ui.Box(c).Height(13).Children(func() {
					if showMonth {
						date, _ := time.Parse("2006-01", month)
						ui.Text(c, date.Format("Jan")).FontSize(9).TextColor(t.TextMuted).Absolute()
					}
				})
				for row, d := range w {
					cell := ui.Box(c).Key(row).Height(11).Radius(2)
					if d == nil {
						cell.Invisible()
						continue
					}
					colour := t.Background.Mix(t.Text, 0.09)
					if d.Level > 0 {
						if a.palette == "green" {
							colour = ui.Hex([]string{"", "#0e4429", "#006d32", "#26a641", "#39d353"}[d.Level])
						} else {
							colour = t.Background.Mix(t.Accent, []float32{0, 0.25, 0.45, 0.7, 1}[d.Level])
						}
					}
					cell.Background(colour).Tooltip(describe(d)).Label(describe(d))
					if cell.Hovered() || cell.Clicked() {
						a.selected = d.Date
					}
					if cell.Clicked() {
						chart.Focus()
					}
					if d.Date == a.selected {
						cell.Border(1, t.Text)
					}
				}
			})
		}
	})
}
