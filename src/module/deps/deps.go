package deps

import (
	"fmt"

	"ocr/src/module/check"
	"ocr/src/module/exif"
	"ocr/src/module/heic"
	"ocr/src/module/skills"
)

// Verify checks all external dependencies and returns results for each.
// imagePath is used to decide whether HEIC/sips checks are needed.
// Pass "" when the image path is not yet known.
func Verify(skillsDir string, imagePath string) []check.Result {
	checkers := []check.Checker{
		exif.Dep{},
		heic.Dep{ImagePath: imagePath},
		skills.Dep{Dir: skillsDir},
	}

	var results []check.Result
	for _, c := range checkers {
		if r := c.CheckDependency(); r.Name != "" {
			results = append(results, r)
		}
	}
	return results
}

const (
	green = "\033[32m"
	red   = "\033[31m"
	reset = "\033[0m"
)

// Print displays a human-readable dependency check summary.
func Print(results []check.Result) {
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
func Fatal(results []check.Result) bool {
	for _, r := range results {
		if !r.OK {
			return true
		}
	}
	return false
}
