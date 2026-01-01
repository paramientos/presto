package scripts

import (
	"os"
	"path/filepath"
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
