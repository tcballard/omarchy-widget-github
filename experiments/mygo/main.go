package main

import (
	"context"
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type application struct {
	data                               dataState
	input, family, palette, validation string
	demo                               bool
	selected                           string
	window                             *mygo.Window
	ctx                                context.Context
	cancel                             context.CancelFunc
	requestCancel                      context.CancelFunc
	workers                            sync.WaitGroup
	theme                              map[string]string
}

func dimensions(family string) (int, int) {
	switch family {
	case "small":
		return 440, 430
	case "large":
		return 680, 550
	default:
		return 960, 430
	}
}

func (a *application) refresh() {
	if a.demo {
		return
	}
	login, err := username(a.input)
	if err != nil {
		a.validation = err.Error()
		return
	}
	a.validation = ""
	if a.data.calendar.Username == login && !a.data.updated.IsZero() && time.Since(a.data.updated) < 15*time.Minute && a.data.err == "" {
		return
	}
	if a.requestCancel != nil {
		a.requestCancel()
	}
	a.selected = ""
	generation := a.data.begin(login)
	ctx, cancel := context.WithCancel(a.ctx)
	a.requestCancel = cancel
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		result, err := fetchCalendar(ctx, publicClient(), login)
		if ctx.Err() != nil {
			return
		}
		a.window.Update(func() { a.data.complete(generation, result, err, time.Now()) })
	}()
}

func main() {
	login := flag.String("username", "tcballard", "public GitHub username")
	family := flag.String("size", "medium", "small, medium or large")
	palette := flag.String("palette", "green", "green or theme")
	demo := flag.Bool("demo", false, "synthetic calendar; no network")
	check := flag.Bool("check", false, "fetch and validate public data without opening a window")
	render := flag.String("render", "", "render a --demo PNG without opening a window")
	flag.Parse()
	if flag.NArg() != 0 {
		log.Fatal("unexpected positional arguments")
	}
	if _, err := username(*login); err != nil {
		log.Fatal(err)
	}
	if *family != "small" && *family != "medium" && *family != "large" {
		log.Fatal("size must be small, medium or large")
	}
	if *palette != "green" && *palette != "theme" {
		log.Fatal("palette must be green or theme")
	}
	if *check && (*demo || *render != "") {
		log.Fatal("--check cannot be combined with --demo or --render")
	}
	if *render != "" && !*demo {
		log.Fatal("--render requires --demo; screenshots must identify synthetic data")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if *check {
		c, err := fetchCalendar(ctx, publicClient(), *login)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s: %d public contributions across %d days (%s – %s)\n", c.Username, c.Total, len(c.Days), c.Days[0].Date, c.Days[len(c.Days)-1].Date)
		return
	}
	a := &application{input: *login, family: *family, palette: *palette, demo: *demo, ctx: ctx, cancel: cancel, theme: loadTheme()}
	if *demo {
		a.data.calendar = demoCalendar()
	}
	w, h := dimensions(*family)
	if *render != "" {
		f, err := os.Create(*render)
		if err != nil {
			log.Fatal(err)
		}
		err = png.Encode(f, ui.Render(a.view, w, h, 1))
		closeErr := f.Close()
		if err != nil {
			log.Fatal(err)
		}
		if closeErr != nil {
			log.Fatal(closeErr)
		}
		return
	}
	mygo.App.SetName("GitHub Contributions — MyGo")
	mygo.App.WhenReady(func() {
		a.window = mygo.NewWindow(mygo.WindowOptions{Title: "GitHub Contributions — MyGo experiment", Width: w, Height: h, MinWidth: 440, MinHeight: 430, Content: ui.View(a.view)})
		a.window.OnClosed(func() { a.data.generation++; cancel() })
		a.refresh()
	})
	err := mygo.App.Run()
	cancel()
	a.workers.Wait()
	if err != nil {
		log.Fatal(err)
	}
}
