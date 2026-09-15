package ui

import (
	"fmt"
	"io"
	"os"
	"time"
)

var (
	statusOut io.Writer = os.Stderr
	dataOut   io.Writer = os.Stdout
)

// Status writes progress to stderr, keeping stdout free for data.
func Status(format string, a ...any) {
	_, _ = fmt.Fprintf(statusOut, format+"\n", a...)
}

func Print(format string, a ...any) {
	_, _ = fmt.Fprintf(dataOut, format+"\n", a...)
}

func Blank() {
	_, _ = fmt.Fprintln(statusOut)
}

func Result(verb string, count int, d time.Duration) {
	Status("%s %d %s %s",
		boldStyle.Render(verb),
		count,
		plural(count, "package"),
		dimStyle.Render("in "+Duration(d)),
	)
}

func Added(name, version string) {
	Status(" %s %s %s", addStyle.Render("+"), name, dimStyle.Render(version))
}

func Removed(name, version string) {
	Status(" %s %s %s", dropStyle.Render("-"), name, dimStyle.Render(version))
}

func Warn(format string, a ...any) {
	Status("%s %s", warnStyle.Render("warning:"), fmt.Sprintf(format, a...))
}

func Fail(format string, a ...any) {
	Status("%s %s", errorStyle.Render("error:"), fmt.Sprintf(format, a...))
}

func Note(format string, a ...any) {
	Status("%s", dimStyle.Render(fmt.Sprintf(format, a...)))
}

func Heading(s string) {
	Print("%s", boldStyle.Render(s))
}

func Duration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.2fs", d.Seconds())
	default:
		return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
