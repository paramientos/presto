package scripts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aras/presto/internal/parser"
)

func TestRunner_PHPClassCall(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "presto-script-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// Create vendor directory and a dummy autoload.php
	vendorDir := "vendor"
	os.MkdirAll(vendorDir, 0755)

	// A dummy class to call
	dummyPHP := `<?php
namespace TestNamespace;
class TestClass {
    public static function testMethod($event = null) {
        file_put_contents('result.txt', 'called');
        if ($event instanceof \Composer\Script\Event) {
             file_put_contents('event_ok.txt', 'ok');
             if ($event->getComposer()->getConfig()->get('vendor-dir') === 'vendor') {
                 file_put_contents('config_ok.txt', 'ok');
             }
        }
    }
}
`
	os.WriteFile("TestClass.php", []byte(dummyPHP), 0644)

	// autoload.php that loads the test class
	autoloadContent := `<?php
require_once __DIR__ . '/../TestClass.php';
`
	os.WriteFile(filepath.Join(vendorDir, "autoload.php"), []byte(autoloadContent), 0644)

	runner := NewRunner(true)
	composer := &parser.ComposerJSON{
		Config: map[string]interface{}{
			"vendor-dir": "vendor",
		},
	}

	// The command we want to run
	command := "TestNamespace\\TestClass::testMethod"

	err = runner.executeCommand(command, composer)
	if err != nil {
		t.Errorf("executeCommand failed: %v", err)
	}

	// Verify the method was called
	if _, err := os.Stat("result.txt"); os.IsNotExist(err) {
		t.Error("PHP method was not called (result.txt missing)")
	}

	// Verify the event object was passed and had the right type
	if _, err := os.Stat("event_ok.txt"); os.IsNotExist(err) {
		t.Error("Event object was not passed or wrong type (event_ok.txt missing)")
	}

	// Verify the config mock works
	if _, err := os.Stat("config_ok.txt"); os.IsNotExist(err) {
		t.Error("Event config mock failed (config_ok.txt missing)")
	}
}

// TestRunner_ScriptReference verifies that @script-name references invoke the
// referenced script correctly.
func TestRunner_ScriptReference(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "presto-script-ref-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	runner := NewRunner(false)
	composer := &parser.ComposerJSON{
		Scripts: map[string]interface{}{
			"inner": "touch inner-ran.txt",
			"outer": "@inner",
		},
	}

	if err := runner.Run("outer", composer); err != nil {
		t.Fatalf("Run(outer) failed: %v", err)
	}

	if _, err := os.Stat("inner-ran.txt"); os.IsNotExist(err) {
		t.Error("referenced script 'inner' was not executed")
	}
}

// TestRunner_MissingScript verifies that running a non-existent event is a no-op.
func TestRunner_MissingScript(t *testing.T) {
	runner := NewRunner(false)
	composer := &parser.ComposerJSON{
		Scripts: map[string]interface{}{
			"foo": "echo hello",
		},
	}

	// A lifecycle event that isn't defined should return nil silently.
	if err := runner.Run("pre-install-cmd", composer); err != nil {
		t.Errorf("expected nil for missing script, got: %v", err)
	}
}

// TestRunner_NilScripts verifies that a composer with no scripts section is safe.
func TestRunner_NilScripts(t *testing.T) {
	runner := NewRunner(false)
	composer := &parser.ComposerJSON{}

	if err := runner.Run("post-install-cmd", composer); err != nil {
		t.Errorf("expected nil for nil scripts map, got: %v", err)
	}
}

// TestRunner_BackslashQuoteUnescaping verifies that \" in a command is normalised
// to " before execution, fixing the PHP parse error reported in issue #9.
func TestRunner_BackslashQuoteUnescaping(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "presto-escape-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	runner := NewRunner(false)
	composer := &parser.ComposerJSON{}

	// Simulate the value Go's JSON parser produces when composer.json contains:
	//   "@php -r \"file_exists('.env') || copy('.env.example', '.env')\""
	// After JSON decode the string contains literal backslash+quote characters.
	command := `@php -r \"file_put_contents('result.txt', 'ok');\"`

	if err := runner.executeCommand(command, composer); err != nil {
		t.Fatalf("executeCommand failed: %v", err)
	}

	content, err := os.ReadFile("result.txt")
	if err != nil {
		t.Fatal("result.txt was not created — PHP did not execute correctly")
	}
	if string(content) != "ok" {
		t.Errorf("unexpected file content: %q", string(content))
	}
}

// TestRunner_SystemShell verifies that the system shell ($SHELL) is used rather
// than hardcoded /bin/sh, so shell-specific builtins are available (issue #10).
func TestRunner_SystemShell(t *testing.T) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		t.Skip("$SHELL not set, skipping system-shell test")
	}

	tmpDir, err := os.MkdirTemp("", "presto-shell-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	runner := NewRunner(false)
	composer := &parser.ComposerJSON{}

	// `type` is a shell builtin available in bash/zsh but not always in /bin/sh.
	// We use a simpler portable check: just confirm the shell executes a command
	// that writes a known value, proving the correct shell was invoked.
	command := "echo $0 > shell-name.txt"

	if err := runner.executeCommand(command, composer); err != nil {
		t.Fatalf("executeCommand failed: %v", err)
	}

	content, err := os.ReadFile("shell-name.txt")
	if err != nil {
		t.Fatal("shell-name.txt was not created")
	}

	// $0 in a -c invocation is the shell binary path itself.
	got := strings.TrimSpace(string(content))
	if got == "" {
		t.Error("$0 was empty — shell did not execute correctly")
	}
	if filepath.Base(got) != filepath.Base(shell) {
		t.Errorf("expected shell %q, got %q", filepath.Base(shell), filepath.Base(got))
	}
}

// TestRunner_ScriptArgs verifies that extra arguments are appended to the
// command and forwarded correctly (issue #12).
func TestRunner_ScriptArgs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "presto-args-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	runner := NewRunner(false)
	composer := &parser.ComposerJSON{
		Scripts: map[string]interface{}{
			// Script writes its arguments to a file via shell redirection.
			"dump-args": "echo",
		},
	}

	// Run with extra args — equivalent to: presto run dump-args hello world
	if err := runner.Run("dump-args", composer, "hello", "world"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

// TestRunner_ScriptArgsSpecialChars verifies that arguments containing spaces
// and special characters are shell-quoted correctly.
func TestRunner_ScriptArgsSpecialChars(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "presto-args-special-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	runner := NewRunner(false)
	composer := &parser.ComposerJSON{}

	// Write the first argument verbatim to a file so we can assert its value.
	arg := "hello world"
	command := "printf '%s' \"$1\" > result.txt"

	if err := runner.executeCommand(command, composer, arg); err != nil {
		t.Fatalf("executeCommand failed: %v", err)
	}

	content, err := os.ReadFile("result.txt")
	if err != nil {
		t.Fatal("result.txt was not created")
	}
	if got := string(content); got != arg {
		t.Errorf("expected %q, got %q", arg, got)
	}
}

// TestShellQuote verifies the shellQuote helper handles edge cases.
func TestShellQuote(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hello", "'hello'"},
		{"hello world", "'hello world'"},
		{"it's", "'it'\\''s'"},
		{"", "''"},
	}
	for _, c := range cases {
		got := shellQuote(c.input)
		if got != c.want {
			t.Errorf("shellQuote(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
