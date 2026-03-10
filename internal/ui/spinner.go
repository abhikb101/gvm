package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// Spinner displays an animated spinner with a message.
type Spinner struct {
	message string
	done    chan struct{}
	mu      sync.Mutex
	stopped bool
}

var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates and starts a new spinner.
func NewSpinner(message string) *Spinner {
	s := &Spinner{
		message: message,
		done:    make(chan struct{}),
	}
	s.start()
	return s
}

func (s *Spinner) start() {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Printf("%s...\n", s.message)
		return
	}

	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				s.mu.Lock()
				fmt.Printf("\r%s %s", Cyan(frames[i%len(frames)]), s.message)
				s.mu.Unlock()
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner and shows a result.
func (s *Spinner) Stop(success bool) {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()

	close(s.done)
	time.Sleep(100 * time.Millisecond)

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if success {
		fmt.Printf("\r%s %s\n", Green("✓"), s.message)
	} else {
		fmt.Printf("\r%s %s\n", Red("✗"), s.message)
	}
}

// StopWithMessage stops the spinner and shows a custom result message.
func (s *Spinner) StopWithMessage(success bool, msg string) {
	s.mu.Lock()
	s.message = msg
	s.mu.Unlock()
	s.Stop(success)
}
