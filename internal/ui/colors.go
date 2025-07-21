package ui

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ANSI color codes
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	
	// Foreground colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"
	
	// Background colors
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// ColorEnabled determines if colors should be used
var ColorEnabled = true

func init() {
	// Disable colors if not running in a terminal or if NO_COLOR is set
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		ColorEnabled = false
	}
}

// colorize wraps text with color codes if colors are enabled
func colorize(color, text string) string {
	if !ColorEnabled || text == "" {
		return text
	}
	return color + text + Reset
}

// ColorRed returns red colored text
func ColorRed(text string) string {
	return colorize(Red, text)
}

// ColorGreen returns green colored text
func ColorGreen(text string) string {
	return colorize(Green, text)
}

// ColorYellow returns yellow colored text
func ColorYellow(text string) string {
	return colorize(Yellow, text)
}

// ColorBlue returns blue colored text
func ColorBlue(text string) string {
	return colorize(Blue, text)
}

// ColorMagenta returns magenta colored text
func ColorMagenta(text string) string {
	return colorize(Magenta, text)
}

// ColorCyan returns cyan colored text
func ColorCyan(text string) string {
	return colorize(Cyan, text)
}

// ColorWhite returns white colored text
func ColorWhite(text string) string {
	return colorize(White, text)
}

// ColorGray returns gray colored text
func ColorGray(text string) string {
	return colorize(Gray, text)
}

// ColorBold returns bold text
func ColorBold(text string) string {
	return colorize(Bold, text)
}

// ProgressSpinner shows a simple spinner animation
type ProgressSpinner struct {
	message   string
	frames    []string
	active    bool
	done      chan bool
	startTime time.Time
}

// NewProgressSpinner creates a new progress spinner
func NewProgressSpinner(message string) *ProgressSpinner {
	return &ProgressSpinner{
		message:   message,
		frames:    []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:      make(chan bool),
		startTime: time.Now(),
	}
}

// Start begins the spinner animation
func (s *ProgressSpinner) Start() {
	if !ColorEnabled {
		fmt.Printf("%s...", s.message)
		return
	}
	
	s.active = true
	go func() {
		frame := 0
		for {
			select {
			case <-s.done:
				// Clear the current line and return
				fmt.Printf("\r%s\r", strings.Repeat(" ", len(s.message)+10))
				return
			default:
				fmt.Printf("\r%s %s", ColorCyan(s.frames[frame]), s.message)
				frame = (frame + 1) % len(s.frames)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Stop ends the spinner animation
func (s *ProgressSpinner) Stop() {
	if !s.active {
		return
	}
	
	// Ensure spinner runs for at least 200ms for visibility
	elapsed := time.Since(s.startTime)
	if elapsed < 200*time.Millisecond {
		time.Sleep(200*time.Millisecond - elapsed)
	}
	
	s.active = false
	s.done <- true
	
	// Clear the spinner line
	fmt.Printf("\r%s\r", strings.Repeat(" ", len(s.message)+10))
}

// UpdateMessage changes the spinner message
func (s *ProgressSpinner) UpdateMessage(message string) {
	s.message = message
}