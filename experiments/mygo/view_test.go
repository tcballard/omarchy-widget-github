package main

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestNativeLayoutsAndControls(t *testing.T) {
	for _, family := range []string{"small", "medium", "large"} {
		t.Run(family, func(t *testing.T) {
			a := &application{input: "tcballard", family: family, palette: "green", demo: true}
			a.data.calendar = demoCalendar()
			w, h := dimensions(family)
			tt := ui.NewTester(a.view, w, h)
			if !tt.HasText("DEMO · SYNTHETIC DATA") || !tt.HasText(number(a.data.calendar.Total)) {
				t.Fatal("missing demo or annual total")
			}
			if err := tt.Click("Toggle calendar palette"); err != nil {
				t.Fatal(err)
			}
			if a.palette != "theme" {
				t.Fatal("palette did not change")
			}
			if err := tt.Click("large"); err != nil {
				t.Fatal(err)
			}
			if a.family != "large" {
				t.Fatal("size did not change")
			}
		})
	}
}

func TestKeyboardCalendar(t *testing.T) {
	a := &application{input: "tcballard", family: "medium", palette: "green", demo: true}
	a.data.calendar = demoCalendar()
	tt := ui.NewTester(a.view, 960, 430)
	if err := tt.Click("Contribution calendar"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyRight)
	if a.selected == "" {
		t.Fatal("keyboard did not inspect day")
	}
	before := a.selected
	tt.Key(0, ui.KeyRight)
	if before == a.selected {
		t.Fatal("keyboard did not advance week")
	}
	if err := tt.Click("Profile"); err == nil && len(tt.OpenedURLs()) > 0 {
		t.Fatal("demo opened a profile")
	}
}

func Example_number() {
	fmt.Println(number(6729))
	// Output: 6,729
}
