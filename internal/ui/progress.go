package ui

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// Progress tracks completion of parallel tasks with a simple counter display.
type Progress struct {
	out       io.Writer
	total     int
	completed atomic.Int32
	mu        sync.Mutex
}

// NewProgress creates a progress tracker for n tasks.
func NewProgress(out io.Writer, total int) *Progress {
	return &Progress{out: out, total: total}
}

// Done marks one task as completed and prints the current progress.
func (p *Progress) Done(label string) {
	n := int(p.completed.Add(1))
	p.mu.Lock()
	defer p.mu.Unlock()
	_, _ = fmt.Fprintf(p.out, "[%d/%d] %s\n", n, p.total, label)
}

// Log prints an informational message within the progress context.
func (p *Progress) Log(format string, args ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, _ = fmt.Fprintf(p.out, format+"\n", args...)
}
