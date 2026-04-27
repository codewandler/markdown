# Streaming Markdown Fixture

This fixture is intentionally separate from the project README. It is meant to
exercise the prototype event pipeline with a mix of regular prose, headings,
and a long fenced code block that should start rendering before the whole
document has arrived.

## Why This Exists

The parser should be able to accept very small chunks from an AI stream and keep
rendering progress visible. Fenced code blocks are especially important because
agent responses often contain long snippets.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type Event struct {
	Kind string
	Text string
}

type Parser struct {
	buffer strings.Builder
	line   int
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Write(chunk string) ([]Event, error) {
	if chunk == "" {
		return nil, nil
	}

	// Keep only unresolved stream tails in memory.
	p.buffer.WriteString(chunk)
	var events []Event
	for {
		current := p.buffer.String()
		next := strings.IndexByte(current, '\n')
		if next < 0 {
			break
		}

		line := current[:next]
		p.buffer.Reset()
		p.buffer.WriteString(current[next+1:])
		p.line++

		events = append(events, Event{
			Kind: "line",
			Text: fmt.Sprintf("%03d: %s", p.line, line),
		})
	}
	return events, nil
}

func (p *Parser) Flush() ([]Event, error) {
	if p.buffer.Len() == 0 {
		return nil, nil
	}
	p.line++
	event := Event{
		Kind: "line",
		Text: fmt.Sprintf("%03d: %s", p.line, p.buffer.String()),
	}
	p.buffer.Reset()
	return []Event{event}, nil
}

func Stream(ctx context.Context, r io.Reader, parser *Parser, chunkSize int) error {
	if chunkSize <= 0 {
		return errors.New("chunk size must be greater than zero")
	}

	buf := make([]byte, chunkSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := r.Read(buf)
		if n > 0 {
			events, writeErr := parser.Write(string(buf[:n]))
			if writeErr != nil {
				return writeErr
			}
			for _, event := range events {
				fmt.Println(event.Text)
			}
		}
		if err == io.EOF {
			events, flushErr := parser.Flush()
			if flushErr != nil {
				return flushErr
			}
			for _, event := range events {
				fmt.Println(event.Text)
			}
			return nil
		}
		if err != nil {
			return err
		}
		time.Sleep(15 * time.Millisecond)
	}
}

func main() {
	const chunkSize = 11
	input := strings.NewReader(strings.Repeat("alpha beta gamma\n", 12))
	parser := NewParser()
	if err := Stream(context.Background(), input, parser, chunkSize); err != nil {
		panic(err)
	}
}
```

## After The Code

The text after the fence verifies that the parser exits code mode and resumes
normal paragraph handling.

## Rust Fixture

The optional Chroma adapter should make this fence much more useful than the
built-in fallback highlighter.

```rust
use std::collections::VecDeque;
use std::fmt::{self, Display};
use std::time::{Duration, Instant};

#[derive(Debug, Clone, PartialEq, Eq)]
enum EventKind {
    Text,
    SoftBreak,
    HardBreak,
    EnterBlock(&'static str),
    ExitBlock(&'static str),
}

#[derive(Debug, Clone)]
struct Event {
    kind: EventKind,
    text: String,
    created_at: Instant,
}

impl Display for Event {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match &self.kind {
            EventKind::Text => write!(f, "text({})", self.text),
            EventKind::SoftBreak => f.write_str("soft-break"),
            EventKind::HardBreak => f.write_str("hard-break"),
            EventKind::EnterBlock(name) => write!(f, "enter<{name}>"),
            EventKind::ExitBlock(name) => write!(f, "exit</{name}>"),
        }
    }
}

struct StreamWindow {
    pending: VecDeque<Event>,
    max_age: Duration,
}

impl StreamWindow {
    fn new(max_age: Duration) -> Self {
        Self {
            pending: VecDeque::new(),
            max_age,
        }
    }

    fn push(&mut self, event: Event) {
        self.pending.push_back(event);
        self.drop_expired();
    }

    fn drain_ready(&mut self) -> Vec<Event> {
        let mut ready = Vec::new();
        while let Some(front) = self.pending.front() {
            if front.created_at.elapsed() < self.max_age {
                break;
            }
            if let Some(event) = self.pending.pop_front() {
                ready.push(event);
            }
        }
        ready
    }

    fn drop_expired(&mut self) {
        while self.pending.len() > 256 {
            self.pending.pop_front();
        }
    }
}

fn tokenize_lines(input: &str) -> impl Iterator<Item = Event> + '_ {
    input.lines().map(|line| Event {
        kind: if line.trim().is_empty() {
            EventKind::SoftBreak
        } else {
            EventKind::Text
        },
        text: line.to_owned(),
        created_at: Instant::now(),
    })
}

fn main() {
    let mut window = StreamWindow::new(Duration::from_millis(30));
    for event in tokenize_lines("alpha\nbeta\n\nfinal") {
        window.push(event);
    }

    for event in window.drain_ready() {
        println!("{event}");
    }
}
```
