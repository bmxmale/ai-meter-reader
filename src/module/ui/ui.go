package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"ocr/src/module/skills"
)

const (
	Bold   = "\033[1m"
	Cyan   = "\033[36m"
	Yellow = "\033[33m"
	Green  = "\033[32m"
	Gray   = "\033[90m"
	Reset  = "\033[0m"
)

// Step prints a status line with trailing "...".
func Step(msg string) {
	fmt.Printf("  %s→%s  %s ... ", Cyan, Reset, msg)
}

// StepLabel prints a section header without trailing "...".
func StepLabel(msg string) {
	fmt.Printf("  %s→%s  %s\n", Cyan, Reset, msg)
}

// Done prints a completion checkmark on the current line.
func Done() {
	fmt.Printf("%s✓%s\n", Green, Reset)
}

// Spin runs fn while displaying an animated spinner.
func Spin(msg string, fn func()) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stop:
				fmt.Printf("\r  %s✓%s  %s          \n", Green, Reset, msg)
				return
			default:
				fmt.Printf("\r  %s%s%s  %s", Cyan, frames[i%len(frames)], Reset, msg)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()
	fn()
	close(stop)
	wg.Wait()
}

// PrintBanner prints the application header box.
func PrintBanner() {
	lines := []string{
		"  OCR Meter Reader",
		"  Read utility meters from photos",
	}
	width := 40
	fmt.Println(Cyan + "  ┌" + strings.Repeat("─", width) + "┐" + Reset)
	for _, l := range lines {
		pad := width - len(l) - 1
		fmt.Printf("%s  │%s %s%s%s%s│%s\n", Cyan, Reset, Bold, l, Reset, Cyan+strings.Repeat(" ", pad), Reset)
	}
	fmt.Println(Cyan + "  └" + strings.Repeat("─", width) + "┘" + Reset)
	fmt.Println()
}

// PrintHelp prints usage instructions.
func PrintHelp(skillsDir string) {
	fmt.Printf("  Reads utility meter photos and extracts readings using a local\n")
	fmt.Printf("  or cloud-based vision model via Ollama. EXIF metadata (date, GPS)\n")
	fmt.Printf("  is preserved in the output but stripped before sending to the model.\n\n")

	fmt.Printf("  %sUsage:%s\n", Bold, Reset)
	fmt.Printf("    ./ocr [flags] <skill> <image> [model]\n\n")

	fmt.Printf("  %sFlags:%s\n", Bold, Reset)
	fmt.Printf("    %s-o <file>%s      Write JSON output to file\n", Yellow, Reset)
	fmt.Printf("    %s-skills <dir>%s  Path to skills directory %s(default: %s)%s\n\n", Yellow, Reset, Gray, skillsDir, Reset)

	fmt.Printf("  %sExamples:%s\n", Bold, Reset)
	fmt.Printf("    %s# Default model (cloud)%s\n", Gray, Reset)
	fmt.Printf("    ./ocr gas ./example/IMG_3290.HEIC\n")
	fmt.Printf("    ./ocr -o result.json gas ./example/IMG_3290.HEIC\n\n")
	fmt.Printf("    %s# Local models (run on your machine)%s\n", Gray, Reset)
	fmt.Printf("    ./ocr gas ./example/IMG_3290.HEIC %sgemma4:e4b%s\n", Yellow, Reset)
	fmt.Printf("    ./ocr gas ./example/IMG_3290.HEIC %sqwen2.5vl:3b%s\n", Yellow, Reset)
	fmt.Printf("    ./ocr gas ./example/IMG_3290.HEIC %sqwen2.5vl:7b%s\n\n", Yellow, Reset)
	fmt.Printf("    %s# Cloud models (via Ollama cloud endpoint)%s\n", Gray, Reset)
	fmt.Printf("    ./ocr gas ./example/IMG_3290.HEIC %sgemma4:31b-cloud%s\n\n", Cyan, Reset)
	fmt.Printf("  %sAvailable cloud models:%s https://ollama.com/search?c=vision&c=cloud\n\n", Bold, Reset)
}

// ListSkills prints a formatted table of available skills.
func ListSkills(skillsDir string) {
	infos := skills.List(skillsDir)

	fmt.Printf("  %sAvailable skills:%s\n", Bold, Reset)
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.Style().Color.Header = text.Colors{text.FgCyan, text.Bold}
	t.Style().Color.Border = text.Colors{text.FgCyan}
	t.Style().Color.Separator = text.Colors{text.FgCyan}
	t.AppendHeader(table.Row{"Skill", "Type", "Description"})
	for _, s := range infos {
		t.AppendRow(table.Row{
			text.Colors{text.FgYellow}.Sprint(s.File),
			s.MeterType,
			s.Description,
		})
	}
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMin: 10},
		{Number: 2, WidthMin: 8},
		{Number: 3, WidthMin: 50},
	})
	t.Render()
	fmt.Println()
}

// PrintSummary parses JSON output and prints a human-readable summary.
func PrintSummary(output string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return
	}

	width := 44
	line := strings.Repeat("─", width)
	row := func(label, value, color string) {
		fmt.Printf("  %s%-18s%s %s%s%s\n", Gray, label, Reset, color, value, Reset)
	}

	fmt.Println()
	fmt.Printf("  %s%s%s\n", Cyan, line, Reset)
	fmt.Printf("  %s%sSummary%s\n", Bold, Cyan, Reset)
	fmt.Printf("  %s%s%s\n", Cyan, line, Reset)

	if meter, ok := data["meter"].(map[string]interface{}); ok {
		if t, ok := meter["type"].(string); ok {
			row("Meter type:", t, Yellow)
		}
		if sn, ok := meter["serial_number"].(string); ok {
			row("Serial number:", sn, Bold)
		}
		if v, ok := meter["value"].(map[string]interface{}); ok {
			reading, _ := v["reading"].(string)
			unit, _ := v["unit"].(string)
			row("Reading:", reading+" "+unit, Bold)
		}
		if req, ok := meter["requires_confirmation"].(bool); ok {
			if req {
				row("Confirmation:", "⚠  required — digit unreadable", Yellow)
			} else {
				row("Confirmation:", "✓  not required", Green)
			}
		}
	}

	if exif, ok := data["exif"].(map[string]interface{}); ok {
		if ts, ok := exif["created_at"].(string); ok {
			t, err := time.Parse("2006-01-02T15:04:05Z07:00", ts)
			if err == nil {
				row("Date taken:", t.Format("2006-01-02  15:04:05"), "")
			}
		}
		if gps, ok := exif["gps"].(map[string]interface{}); ok {
			lat, _ := gps["lat"].(float64)
			lon, _ := gps["lon"].(float64)
			row("GPS:", fmt.Sprintf("%.6f°N  %.6f°E", lat, lon), "")
		}
	}

	fmt.Printf("  %s%s%s\n\n", Cyan, line, Reset)
}
