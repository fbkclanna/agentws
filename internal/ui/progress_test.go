package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgress_Done(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 3)

	p.Done("task A")
	p.Done("task B")
	p.Done("task C")

	out := buf.String()
	if !strings.Contains(out, "[1/3] task A") {
		t.Errorf("missing progress line for task A: %s", out)
	}
	if !strings.Contains(out, "[2/3] task B") {
		t.Errorf("missing progress line for task B: %s", out)
	}
	if !strings.Contains(out, "[3/3] task C") {
		t.Errorf("missing progress line for task C: %s", out)
	}
}

func TestProgress_Log(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 1)

	p.Log("hello %s", "world")

	out := buf.String()
	if !strings.Contains(out, "hello world") {
		t.Errorf("missing log message: %s", out)
	}
}
