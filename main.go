package main

import (
	"fmt"
	"log"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/mgutz/ansi"
	"github.com/nsf/termbox-go"
)

var version = "1.0"

var usage = `tmux-autocomplete - provides autocomplete interface for pane contents.

Usage:
  tmux-autocomplete -h | --help
  tmux-autocomplete [options]
  tmux-autocomplete [options] -W <pane> <cursor-x> <cursor-y>


Options:
  -h --help    Show this help.
  -r <regexp>  Identifier regexp to match.
                [default: [!-~]+]
  -l <log>     Specify log file [default: /dev/stderr]
`

func main() {
	args, err := docopt.Parse(
		usage,
		nil,
		true,
		"tmux-autocomplete "+version,
		false,
	)
	if err != nil {
		panic(err)
	}

	logfile, err := os.OpenFile(
		args["-l"].(string),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0600,
	)
	if err != nil {
		panic(err)
	}

	log.SetOutput(logfile)

	tmux := &Tmux{}

	if !args["-W"].(bool) {
		err := start(args, tmux)
		if err != nil {
			log.Fatalln(err)
		}

		return
	}

	var (
		cursorX int
		cursorY int
	)

	_, err = fmt.Sscan(args["<cursor-x>"].(string), &cursorX)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = fmt.Sscan(args["<cursor-y>"].(string), &cursorY)
	if err != nil {
		log.Fatalln(err)
	}

	pane, err := CapturePane(tmux, args["<pane>"].(string), "-eJ")
	if err != nil {
		log.Fatalln(err)
	}

	x, y := pane.GetBufferXY(cursorX, cursorY)
	identifier, err := getIdentifierToComplete(args, pane, x, y)
	if err != nil {
		log.Fatalln(err)
	}

	if identifier == nil {
		return
	}

	moveCursor(cursorX, cursorY)

	candidates, err := getCompletionCandidates(args, pane, identifier)
	if err != nil {
		log.Fatalln(err)
	}

	selectDefaultCandidate(candidates, identifier.X, identifier.Y)

	if len(getUniqueCandidates(candidates)) == 1 {
		useCurrentCandidate(tmux, pane, identifier, candidates)

		return
	}

	err = termbox.Init()
	if err != nil {
		log.Fatalln(err)
	}

	// TODO: make customizable by CLI
	colorscheme := Colorscheme{}
	colorscheme.Identifier = `default+ub:default`
	colorscheme.Candidate.Normal = `green:default`
	colorscheme.Candidate.Selected = `16+b:green`
	colorscheme.Fog.Text = `236:default`
	colorscheme.Fog.Background = `238:236`

	for {
		renderPane(pane, colorscheme)
		renderIdentifier(tmux, pane, colorscheme, identifier)
		renderCandidates(tmux, pane, colorscheme, candidates)

		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyArrowUp:
				selectNextCandidate(candidates, 0, -1)

			case termbox.KeyArrowDown:
				selectNextCandidate(candidates, 0, 1)

			case termbox.KeyArrowLeft:
				selectNextCandidate(candidates, -1, 0)

			case termbox.KeyArrowRight:
				selectNextCandidate(candidates, 1, 0)

			case termbox.KeyEnter:
				useCurrentCandidate(tmux, pane, identifier, candidates)

				return

			case termbox.KeyCtrlC:
				return
			}

		case termbox.EventError:
			log.Fatalln(ev.Err)
		}
	}
}

func start(args map[string]interface{}, tmux *Tmux) error {
	var (
		pane    string
		cursorX string
		cursorY string
	)

	err := tmux.Eval(
		map[string]interface{}{
			"pane_id":  &pane,
			"cursor_x": &cursorX,
			"cursor_y": &cursorY,
		},
	)
	if err != nil {
		return err
	}

	return tmux.NewWindow(
		os.Args[0],
		"-l",
		args["-l"].(string),
		pane,
		cursorX,
		cursorY,
		"-W",
	)
}

func renderPane(pane *Pane, colors Colorscheme) {
	moveCursor(0, 0)

	fmt.Print(ansi.ColorCode(colors.Fog.Text))

	text := reEscapeSequence.ReplaceAllStringFunc(
		pane.String(),
		func(sequence string) string {
			return decolorize(sequence, colors)
		},
	)

	fmt.Print(text)
}

func decolorize(sequence string, colors Colorscheme) string {
	var (
		matches = reEscapeSequence.FindStringSubmatch(sequence)
		code    = matches[1]
	)

	switch code {
	// 49 means reset background color to default,
	// we remove background color and set dim color for foreground
	case "49":
		return ansi.DefaultBG + ansi.ColorCode(colors.Fog.Text)

	// 39 means reset foreground color to default,
	// 0 means reset all style to default,
	// we reset style and set dim color for foreground
	case "0", "39":
		return ansi.Reset + ansi.ColorCode(colors.Fog.Text)

	// 1 means bold,
	// we remove bold
	case "1":
		return ""

	// 7 means reverse,
	// we set dim background color
	case "7":
		return ansi.ColorCode(colors.Fog.Background)

	default:
		// all single-number codes,
		// we remove
		if len(code) == 1 {
			return ""
		}

		// all foreground colors,
		// we replace with dim foreground color
		if code[0] == '3' || code[0] == '9' {
			return ansi.ColorCode(colors.Fog.Text)
		}

		// all background colors,
		// we replace with dim background color
		if code[0] == '4' || code[0] == '1' {
			return ansi.ColorCode(colors.Fog.Background)
		}
	}

	return ansi.Reset
}

func renderIdentifier(
	tmux *Tmux,
	pane *Pane,
	colors Colorscheme,
	identifier *Identifier,
) {
	moveCursor(pane.GetScreenXY(identifier.X, identifier.Y))

	fmt.Print(ansi.ColorFunc(colors.Identifier)(identifier.Value))
}

func renderCandidates(
	tmux *Tmux,
	pane *Pane,
	colors Colorscheme,
	candidates []*Candidate,
) {
	for _, candidate := range candidates {
		x := candidate.X
		y := candidate.Y

		moveCursor(pane.GetScreenXY(x, y))

		color := colors.Candidate.Normal

		if candidate.Selected {
			color = colors.Candidate.Selected
		}

		fmt.Print(ansi.ColorFunc(color)(candidate.Value))
	}
}

func useCurrentCandidate(
	tmux *Tmux,
	pane *Pane,
	identifier *Identifier,
	candidates []*Candidate,
) {
	selected := getSelectedCandidate(candidates)
	if selected == nil {
		return
	}

	text := string([]rune(selected.Value)[identifier.Length():])

	err := tmux.Paste(text, "-t", pane.ID)
	if err != nil {
		panic(err)
	}
}
