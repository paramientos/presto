package scripts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aras/presto/internal/parser"
)

// Runner handles the execution of Composer scripts
type Runner struct {
	Verbose bool
}

// NewRunner creates a new script runner
func NewRunner(verbose bool) *Runner {
	return &Runner{
		Verbose: verbose,
	}
}

// Run executes scripts associated with a specific event
func (r *Runner) Run(event string, composer *parser.ComposerJSON) error {
	if composer.Scripts == nil {
		return nil
	}

	script, ok := composer.Scripts[event]
	if !ok {
		return nil
	}

	fmt.Printf("🚀 Executing script: %s\n", event)

	switch v := script.(type) {
	case string:
		return r.executeCommand(v, composer)
	case []interface{}:
		for _, cmd := range v {
			if cmdStr, ok := cmd.(string); ok {
				if err := r.executeCommand(cmdStr, composer); err != nil {
					// We log the error but for some scripts we might want to continue
					fmt.Printf("⚠️  Script failed: %v\n", err)
				}
			}
		}
	}

	return nil
}

func (r *Runner) executeCommand(command string, composer *parser.ComposerJSON) error {
	command = strings.TrimSpace(command)

	// Case 1: Reference to another script
	if strings.HasPrefix(command, "@") && !strings.HasPrefix(command, "@php") {
		refScript := strings.TrimPrefix(command, "@")
		return r.Run(refScript, composer)
	}

	// Case 2: PHP Class Method call (e.g. ClassName::method)
	if strings.Contains(command, "::") && !strings.Contains(command, " ") {
		if r.Verbose {
			fmt.Printf("🔍 Detected PHP class call: %s\n", command)
		}
		// Wrap class call in a PHP runner command
		// We need to include vendor/autoload.php if it exists
		vendorDir := "vendor"
		if composer != nil && composer.Config != nil {
			if v, ok := composer.Config["vendor-dir"].(string); ok {
				vendorDir = v
			} else if v, ok := composer.Config["vendorDir"].(string); ok {
				vendorDir = v
			}
		}

		autoloadPath := filepath.Join(vendorDir, "autoload.php")
		if _, err := os.Stat(autoloadPath); os.IsNotExist(err) {
			// If autoloader doesn't exist yet, we can't call classes easily
			return fmt.Errorf("cannot call PHP class %s because %s is missing", command, autoloadPath)
		}

		// Improved PHP snippet to mock Composer Event system
		// This fixes the "Too few arguments" error common in Laravel scripts (ArgumentCountError)
		phpSnippet := fmt.Sprintf(`
require_once '%s';
if (!class_exists('Composer\Script\Event')) {
    eval('namespace Composer\Script; class Event {
        public function getComposer() {
            return new class {
                public function getConfig() {
                    return new class {
                        public function get($k) { return "%s"; }
                    };
                }
            };
        }
    }');
}
%s(new \Composer\Script\Event());
`, autoloadPath, vendorDir, command)

		// Trim and replace newlines to make it a one-liner for php -r
		phpSnippet = strings.ReplaceAll(strings.TrimSpace(phpSnippet), "\n", " ")
		// Use single quotes for the shell command to avoid $ issues and escape internal single quotes
		escapedSnippet := strings.ReplaceAll(phpSnippet, "'", "'\\''")
		command = fmt.Sprintf("php -r '%s'", escapedSnippet)
	}

	// Case 3: @php shortcut
	if strings.HasPrefix(command, "@php ") {
		command = "php " + strings.TrimPrefix(command, "@php ")
	} else if command == "@php" {
		command = "php"
	}

	if r.Verbose {
		fmt.Printf("🔍 Running command: %s\n", command)
	}

	// Prepend vendor/bin to PATH so packages can use their binaries
	path := os.Getenv("PATH")
	vendorBin, _ := filepath.Abs("vendor/bin")
	newPath := vendorBin + string(os.PathListSeparator) + path

	cmd := exec.Command("sh", "-c", command)
	cmd.Env = append(os.Environ(), "PATH="+newPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}
