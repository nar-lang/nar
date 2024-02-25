package logger

import (
	"io"
	"os"
)

type LogWriter struct {
	errors    []error
	warns     []error
	msgs      []string
	OutStream io.Writer
	FailOnErr bool
}

func (l *LogWriter) Err(err ...error) bool {
	l.errors = append(l.errors, err...)
	return l.FailOnErr && len(l.errors) > 0
}

func (l *LogWriter) Warn(err error) {
	l.warns = append(l.warns, err)
}

func (l *LogWriter) Info(msg string) {
	l.msgs = append(l.msgs, msg)
}

func (l *LogWriter) IsEmpty() bool {
	return len(l.errors) == 0 && len(l.warns) == 0 && len(l.msgs) == 0
}

func (l *LogWriter) Trace(s string) {
	w := l.OutStream
	if w == nil {
		w = os.Stdout
	}
	_, _ = w.Write([]byte(s))
	_, _ = w.Write([]byte("\n"))
}

func (l *LogWriter) Flush(w io.Writer) {
	for _, err := range l.errors {
		_, _ = w.Write([]byte("error: " + err.Error() + "\n"))
	}
	for _, err := range l.warns {
		_, _ = w.Write([]byte("warning: " + err.Error() + "\n"))
	}
	for _, msg := range l.msgs {
		_, _ = w.Write([]byte(msg + "\n"))
	}
	l.errors = nil
	l.warns = nil
	l.msgs = nil
}

func (l *LogWriter) Errors() []error {
	return l.errors
}

func (l *LogWriter) Warnings() []error {
	return l.warns
}

func (l *LogWriter) Messages() []string {
	return l.msgs
}
