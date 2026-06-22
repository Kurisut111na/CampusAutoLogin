package main

import (
	"sync"
	"time"

	"github.com/lxn/walk"
)

// =============================================================================
// Colored Log View — custom-painted widget with per-level text colors
// =============================================================================
//
// Renders log entries using DrawTextPixels with level-specific colors:
//   DEBUG = gray, INFO = green, WARN = orange, ERROR = red.
//
// The widget always shows the most recent entries that fit in the visible area,
// newest at the bottom.

// LogView is a custom widget that renders log entries with color-coded levels.
type LogView struct {
	*walk.CustomWidget
	mu       sync.Mutex
	entries  []LogEntry
	maxLines int
	font     *walk.Font
	bgBrush  walk.Brush
}

// NewLogView creates a new colored log viewer.
func NewLogView(parent walk.Container) (*LogView, error) {
	lv := &LogView{
		maxLines: 500,
	}

	cw, err := walk.NewCustomWidgetPixels(parent, 0, func(canvas *walk.Canvas, updateBounds walk.Rectangle) error {
		return lv.paint(canvas, updateBounds)
	})
	if err != nil {
		return nil, err
	}
	lv.CustomWidget = cw

	if err := walk.InitWrapperWindow(lv); err != nil {
		lv.Dispose()
		return nil, err
	}

	lv.SetInvalidatesOnResize(true)

	// Cache font and background brush for paint performance
	lv.font, _ = walk.NewFont("Segoe UI", 9, 0)
	bg, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))
	lv.bgBrush = bg
	lv.SetBackground(bg)

	return lv, nil
}

// Append adds a log entry and schedules a repaint.
func (lv *LogView) Append(level LogLevel, message string) {
	lv.mu.Lock()
	lv.entries = append(lv.entries, LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	})
	if len(lv.entries) > lv.maxLines {
		lv.entries = lv.entries[len(lv.entries)-lv.maxLines:]
	}
	lv.mu.Unlock()

	lv.Invalidate()
}

// Clear removes all entries.
func (lv *LogView) Clear() {
	lv.mu.Lock()
	lv.entries = nil
	lv.mu.Unlock()
	lv.Invalidate()
}

// SetEntries replaces all entries and repaints (used for filter refresh).
func (lv *LogView) SetEntries(entries []LogEntry) {
	lv.mu.Lock()
	lv.entries = make([]LogEntry, len(entries))
	copy(lv.entries, entries)
	lv.mu.Unlock()
	lv.Invalidate()
}

// =============================================================================
// Painting
// =============================================================================

func (lv *LogView) paint(canvas *walk.Canvas, updateBounds walk.Rectangle) error {
	bounds := lv.ClientBoundsPixels()

	// Fill background white (cached brush)
	if lv.bgBrush != nil {
		canvas.FillRectanglePixels(lv.bgBrush, bounds)
	}

	lv.mu.Lock()
	entries := make([]LogEntry, len(lv.entries))
	copy(entries, lv.entries)
	lv.mu.Unlock()

	if len(entries) == 0 {
		return nil
	}

	font := lv.font // cached font

	dpi := lv.DPI()
	lineH := lv.lineHeightPixels(canvas, font)
	if lineH < 14 {
		lineH = 16
	}

	padX := scalePx(4, dpi)
	padY := scalePx(2, dpi)

	visible := (bounds.Height - padY*2) / lineH
	if visible < 1 {
		visible = 1
	}

	start := len(entries) - visible
	if start < 0 {
		start = 0
	}

	tsW := scalePx(62, dpi)  // timestamp column width
	lvW := scalePx(56, dpi)  // level tag column width

	for i, entry := range entries[start:] {
		y := padY + i*lineH

		// --- Timestamp (gray) ---
		ts := entry.Timestamp.Format("15:04:05")
		tsRect := walk.Rectangle{X: padX, Y: y, Width: tsW, Height: lineH}
		if font != nil {
			canvas.DrawTextPixels(ts, font, walk.RGB(150, 150, 150), tsRect,
				walk.TextLeft|walk.TextVCenter|walk.TextSingleLine|walk.TextNoPrefix)
		}

		// --- Level tag (colored) ---
		x2 := padX + tsW
		lvlStr := "[" + levelShortStr(entry.Level) + "]"
		lvlRect := walk.Rectangle{X: x2, Y: y, Width: lvW, Height: lineH}
		if font != nil {
			canvas.DrawTextPixels(lvlStr, font, levelColor(entry.Level), lvlRect,
				walk.TextLeft|walk.TextVCenter|walk.TextSingleLine|walk.TextNoPrefix)
		}

		// --- Message (dark text) ---
		x3 := x2 + lvW
		msgRect := walk.Rectangle{X: x3, Y: y, Width: bounds.Width - x3 - padX, Height: lineH}
		if font != nil {
			canvas.DrawTextPixels(entry.Message, font, walk.RGB(30, 30, 30), msgRect,
				walk.TextLeft|walk.TextVCenter|walk.TextSingleLine|walk.TextNoPrefix|walk.TextEndEllipsis)
		}
	}

	return nil
}

func (lv *LogView) lineHeightPixels(canvas *walk.Canvas, font *walk.Font) int {
	if font == nil {
		return 18
	}
	bounds, _, err := canvas.MeasureTextPixels("Ag", font,
		walk.Rectangle{Width: 9999, Height: 9999}, walk.TextSingleLine)
	if err != nil {
		return 18
	}
	return bounds.Height + 2
}

// =============================================================================
// Helpers
// =============================================================================

func levelColor(level LogLevel) walk.Color {
	switch level {
	case LogDebug:
		return walk.RGB(140, 140, 140) // gray
	case LogInfo:
		return walk.RGB(0, 130, 0) // green
	case LogWarning:
		return walk.RGB(210, 140, 0) // orange
	case LogError:
		return walk.RGB(200, 40, 40) // red
	default:
		return walk.RGB(30, 30, 30) // black
	}
}

func levelShortStr(level LogLevel) string {
	switch level {
	case LogDebug:
		return "DBG"
	case LogInfo:
		return "INF"
	case LogWarning:
		return "WRN"
	case LogError:
		return "ERR"
	default:
		return "???"
	}
}

func scalePx(v int, dpi int) int {
	return int(float64(v) * float64(dpi) / 96.0)
}
