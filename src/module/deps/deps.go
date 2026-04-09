package deps

import (
	"fmt"
	"os"
	"os/exec"
)

type CheckResult struct {
	Name    string
	OK      bool
	Message string
}

// Verify checks all external dependencies and returns results for each.
func Verify(skillsDir string) []CheckResult {
	return []CheckResult{
		checkBinary("exiftool", "exiftool", "-ver"),
		checkBinary("sips", "sips", "--help"),
		checkDir("skills", skillsDir),
	}
}

const (
	green = "\033[32m"
	red   = "\033[31m"
	reset = "\033[0m"
)

// Print displays a human-readable dependency check summary.
func Print(results []CheckResult) {
	fmt.Println("Checking dependencies:")
	for _, r := range results {
		if r.OK {
			fmt.Printf("  %s✓%s  %s\n", green, reset, r.Name)
		} else {
			fmt.Printf("  %s✗%s  %s — %s\n", red, reset, r.Name, r.Message)
		}
	}
	fmt.Println()
}

// Fatal returns true if any dependency check failed.
func Fatal(results []CheckResult) bool {
	for _, r := range results {
		if !r.OK {
			return true
		}
	}
	return false
}

func checkDir(name, path string) CheckResult {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return CheckResult{
			Name:    name,
			OK:      false,
			Message: fmt.Sprintf("skills directory %q not found", path),
		}
	}
	return CheckResult{Name: name, OK: true}
}

func checkBinary(name string, bin string, args ...string) CheckResult {
	if _, err := exec.LookPath(bin); err != nil {
		return CheckResult{
			Name:    name,
			OK:      false,
			Message: fmt.Sprintf("%q not found in PATH", bin),
		}
	}
	if err := exec.Command(bin, args...).Run(); err != nil {
		return CheckResult{
			Name:    name,
			OK:      false,
			Message: fmt.Sprintf("%q found but failed to run: %v", bin, err),
		}
	}
	return CheckResult{Name: name, OK: true}
}