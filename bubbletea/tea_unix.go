//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package bubbletea

import (
	"image/color"
	"os"
	"time"

	"github.com/Kasama/charmbracelet-ssh"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/term"
	"github.com/charmbracelet/x/exp/term/ansi"
	"github.com/charmbracelet/x/exp/term/input"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"
)

func makeOpts(s ssh.Session) []tea.ProgramOption {
	pty, _, ok := s.Pty()
	if !ok || s.EmulatedPty() {
		return []tea.ProgramOption{
			tea.WithInput(s),
			tea.WithOutput(s),
		}
	}

	return []tea.ProgramOption{
		tea.WithInput(pty.Slave),
		tea.WithOutput(pty.Slave),
	}
}

func newRenderer(s ssh.Session) *lipgloss.Renderer {
	pty, _, ok := s.Pty()
	env := sshEnviron(append(s.Environ(), "TERM="+pty.Term))
	var r *lipgloss.Renderer
	var bg color.Color
	if ok && pty.Slave != nil {
		r = lipgloss.NewRenderer(
			pty.Slave,
			termenv.WithEnvironment(env),
			termenv.WithColorCache(true),
		)
		bg = backgroundColor(pty.Slave, pty.Slave)
	} else {
		r = lipgloss.NewRenderer(
			s,
			termenv.WithEnvironment(env),
			termenv.WithUnsafe(),
			termenv.WithColorCache(true),
		)
		bg = queryBackgroundColor(s)
	}
	c, ok := colorful.MakeColor(bg)
	if ok {
		_, _, l := c.Hsl()
		r.SetHasDarkBackground(l < 0.5)
	}
	return r
}

// BackgroundColor queries the terminal for the background color.
// If the terminal does not support querying the background color, nil is
// returned.
func backgroundColor(in, out *os.File) (c color.Color) {
	c = color.White
	state, err := term.MakeRaw(in.Fd())
	if err != nil {
		return
	}

	defer term.Restore(in.Fd(), state) // nolint: errcheck

	// nolint: errcheck
	term.QueryTerminal(in, out, time.Second, func(events []input.Event) bool {
		for _, e := range events {
			switch e := e.(type) {
			case input.BackgroundColorEvent:
				c = e.Color
				continue // we need to consume the next DA1 event
			case input.PrimaryDeviceAttributesEvent:
				return false
			}
		}
		return true
	}, ansi.RequestBackgroundColor+ansi.RequestPrimaryDeviceAttributes)
	return
}

// copied from x/exp/term.
func queryBackgroundColor(s ssh.Session) (bg color.Color) {
	bg = color.White
	_ = term.QueryTerminal(s, s, time.Second, func(events []input.Event) bool {
		for _, e := range events {
			switch e := e.(type) {
			case input.BackgroundColorEvent:
				bg = e.Color
				continue // we need to consume the next DA1 event
			case input.PrimaryDeviceAttributesEvent:
				return false
			}
		}
		return true
	}, ansi.RequestBackgroundColor+ansi.RequestPrimaryDeviceAttributes)
	return
}
