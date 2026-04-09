package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"ocr/src/module/deps"
	exifreader "ocr/src/module/exif"
	"ocr/src/module/heic"
	ollama "ocr/src/module/ollama"
)

const defaultSkillsDir = "meter-skills"

const (
	bold   = "\033[1m"
	cyan   = "\033[36m"
	yellow = "\033[33m"
	green  = "\033[32m"
	gray   = "\033[90m"
	reset  = "\033[0m"
)

func main() {
	outFile := flag.String("o", "", "write JSON output to file")
	skillsDir := flag.String("skills", defaultSkillsDir, "path to meter skills directory")
	flag.Parse()
	args := flag.Args()

	results := deps.Verify(*skillsDir)

	if len(args) < 2 {
		printBanner()
		printHelp(*skillsDir)
		listSkills(*skillsDir)
		deps.Print(results)
		os.Exit(1)
	}

	if deps.Fatal(results) {
		deps.Print(results)
		os.Exit(1)
	}

	skillName := args[0]
	imagePath := args[1]
	var model string
	if len(args) >= 3 {
		model = args[2]
	}

	// Step 1: read EXIF from original file before any modification.
	step("Reading EXIF metadata")
	exifData, _ := exifreader.Read(imagePath)
	done()

	// Step 2: produce a stripped JPEG copy for Ollama.
	step("Preparing image (stripping metadata)")
	var cleanPath string
	var err error
	if heic.IsHEIC(imagePath) {
		cleanPath, err = heic.ToJPG(imagePath)
	} else {
		cleanPath, err = heic.StripJPG(imagePath)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(cleanPath)
	done()

	// Step 3: load skill.
	step("Loading skill: " + skillName)
	system, meterType, err := loadSkill(*skillsDir, skillName)
	if err != nil {
		log.Fatal(err)
	}
	done()

	// Step 4: check Ollama.
	client := &ollama.OllamaClient{
		URL:   ollama.DefaultOllamaURL,
		Model: model,
	}
	if !client.IsRunning() {
		log.Fatal("ollama is not running on ", ollama.DefaultOllamaURL)
	}

	// Step 5: run model with spinner.
	modelName := model
	if modelName == "" {
		modelName = ollama.DefaultOllamaModel
	}
	var result string
	spin("Analyzing image with "+modelName, func() {
		result, err = client.Generate(system, "Analyze this image.", cleanPath)
	})
	if err != nil {
		log.Fatal(err)
	}

	output := mergeExif(result, exifData, meterType)

	stepLabel("Processed data:")
	fmt.Println()

	printSummary(output)

	if *outFile != "" {
		if err := os.WriteFile(*outFile, []byte(output+"\n"), 0644); err != nil {
			log.Fatalf("cannot write output file: %v", err)
		}
		fmt.Printf("    %s✓%s  Raw output saved to file: %s%s%s\n\n", green, reset, bold, *outFile, reset)
	}
}

// step prints a status line prefix (no newline) with trailing "...".
func step(msg string) {
	fmt.Printf("  %s→%s  %s ... ", cyan, reset, msg)
}

// stepLabel prints a status line without trailing "..." — used as a section header.
func stepLabel(msg string) {
	fmt.Printf("  %s→%s  %s\n", cyan, reset, msg)
}

// done prints the completion checkmark on the same line.
func done() {
	fmt.Printf("%s✓%s\n", green, reset)
}

// spin runs fn in a goroutine while displaying an animated spinner.
func spin(msg string, fn func()) {
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
				fmt.Printf("\r  %s✓%s  %s          \n", green, reset, msg)
				return
			default:
				fmt.Printf("\r  %s%s%s  %s", cyan, frames[i%len(frames)], reset, msg)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()
	fn()
	close(stop)
	wg.Wait()
}

func printSummary(output string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return
	}

	width := 44
	line := strings.Repeat("─", width)
	row := func(label, value, color string) {
		fmt.Printf("  %s%-18s%s %s%s%s\n", gray, label, reset, color, value, reset)
	}

	fmt.Println()
	fmt.Printf("  %s%s%s\n", cyan, line, reset)
	fmt.Printf("  %s%sSummary%s\n", bold, cyan, reset)
	fmt.Printf("  %s%s%s\n", cyan, line, reset)

	if meter, ok := data["meter"].(map[string]interface{}); ok {
		if t, ok := meter["type"].(string); ok {
			row("Meter type:", t, yellow)
		}
		if sn, ok := meter["serial_number"].(string); ok {
			row("Serial number:", sn, bold)
		}
		if v, ok := meter["value"].(map[string]interface{}); ok {
			reading, _ := v["reading"].(string)
			unit, _ := v["unit"].(string)
			row("Reading:", reading+" "+unit, bold)
		}
		if req, ok := meter["requires_confirmation"].(bool); ok {
			if req {
				row("Confirmation:", "⚠  required — digit unreadable", "\033[33m")
			} else {
				row("Confirmation:", "✓  not required", green)
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

	fmt.Printf("  %s%s%s\n\n", cyan, line, reset)
}

func mergeExif(result string, exifData *exifreader.Data, meterType string) string {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(extractJSON(result)), &raw); err != nil {
		raw = map[string]interface{}{"raw": result}
	}

	meter := map[string]interface{}{}
	if meterType != "" {
		meter["type"] = meterType
	}
	if sn, ok := raw["serial_number"]; ok {
		meter["serial_number"] = sn
	}
	if reading, ok := raw["meter_reading"].(map[string]interface{}); ok {
		fullValue, _ := reading["full_value"].(string)
		meter["value"] = map[string]interface{}{
			"reading": fullValue,
			"integer": reading["integer_part"],
			"decimal": reading["decimal_part"],
			"unit":    reading["unit"],
		}
		meter["requires_confirmation"] = strings.Contains(fullValue, "?")
	}

	out := map[string]interface{}{
		"meter": meter,
	}

	if exifData != nil {
		exif := map[string]interface{}{}
		if !exifData.DateTime.IsZero() {
			exif["created_at"] = exifData.DateTime.Format("2006-01-02T15:04:05Z07:00")
		}
		if exifData.HasGPS {
			exif["gps"] = map[string]float64{
				"lat": exifData.Lat,
				"lon": exifData.Lon,
			}
		}
		if len(exif) > 0 {
			out["exif"] = exif
		}
	}

	return printJSON(out)
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		start := strings.Index(s, "\n")
		end := strings.LastIndex(s, "```")
		if start != -1 && end > start {
			return strings.TrimSpace(s[start+1 : end])
		}
	}
	return s
}

func printJSON(data map[string]interface{}) string {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(out)
}

func printHelp(skillsDir string) {
	fmt.Printf("  Reads utility meter photos and extracts readings using a local\n")
	fmt.Printf("  or cloud-based vision model via Ollama. EXIF metadata (date, GPS)\n")
	fmt.Printf("  is preserved in the output but stripped before sending to the model.\n\n")

	fmt.Printf("  %sUsage:%s\n", bold, reset)
	fmt.Printf("    ./ocr [flags] <skill> <image> [model]\n\n")

	fmt.Printf("  %sFlags:%s\n", bold, reset)
	fmt.Printf("    %s-o <file>%s      Write JSON output to file\n", yellow, reset)
	fmt.Printf("    %s-skills <dir>%s  Path to skills directory %s(default: %s)%s\n\n", yellow, reset, gray, skillsDir, reset)

	fmt.Printf("  %sExamples:%s\n", bold, reset)
	fmt.Printf("    %s# Default model (cloud)%s\n", gray, reset)
	fmt.Printf("    ./ocr gas ./example/IMG_3290.HEIC\n")
	fmt.Printf("    ./ocr -o result.json gas ./example/IMG_3290.HEIC\n\n")
	fmt.Printf("    %s# Local models (run on your machine)%s\n", gray, reset)
	fmt.Printf("    ./ocr gas ./example/IMG_3290.HEIC %sgemma4:e4b%s\n", yellow, reset)
	fmt.Printf("    ./ocr gas ./example/IMG_3290.HEIC %sqwen2.5vl:3b%s\n", yellow, reset)
	fmt.Printf("    ./ocr gas ./example/IMG_3290.HEIC %sqwen2.5vl:7b%s\n\n", yellow, reset)
	fmt.Printf("    %s# Cloud models (via Ollama cloud endpoint)%s\n", gray, reset)
	fmt.Printf("    ./ocr gas ./example/IMG_3290.HEIC %sgemma4:31b-cloud%s\n\n", cyan, reset)
	fmt.Printf("  %sAvailable cloud models:%s https://ollama.com/search?c=vision&c=cloud\n\n", bold, reset)
}

func printBanner() {
	lines := []string{
		"  OCR Meter Reader",
		"  Read utility meters from photos",
	}
	width := 40
	fmt.Println(cyan + "  ┌" + strings.Repeat("─", width) + "┐" + reset)
	for _, l := range lines {
		pad := width - len(l) - 1
		fmt.Printf("%s  │%s %s%s%s%s│%s\n", cyan, reset, bold, l, reset, cyan+strings.Repeat(" ", pad), reset)
	}
	fmt.Println(cyan + "  └" + strings.Repeat("─", width) + "┘" + reset)
	fmt.Println()
}

type skillInfo struct {
	file        string
	meterType   string
	description string
}

func listSkills(skillsDir string) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return
	}

	var skills []skillInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(skillsDir, e.Name()))
		if err != nil {
			continue
		}
		_, fields := parseFrontmatter(string(data))
		desc := fields["description"]
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		skills = append(skills, skillInfo{
			file:      strings.TrimSuffix(e.Name(), ".md"),
			meterType: fields["type"],
			description: desc,
		})
	}

	fmt.Printf("  %sAvailable skills:%s\n", bold, reset)
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.Style().Color.Header = text.Colors{text.FgCyan, text.Bold}
	t.Style().Color.Border = text.Colors{text.FgCyan}
	t.Style().Color.Separator = text.Colors{text.FgCyan}
	t.AppendHeader(table.Row{"Skill", "Type", "Description"})
	for _, s := range skills {
		t.AppendRow(table.Row{
			text.Colors{text.FgYellow}.Sprint(s.file),
			s.meterType,
			s.description,
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

func loadSkill(skillsDir, name string) (system, meterType string, err error) {
	path := filepath.Join(skillsDir, name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("cannot load skill %q: %w", name, err)
	}
	system, frontmatter := parseFrontmatter(string(data))
	meterType = frontmatter["type"]
	return system, meterType, nil
}

func parseFrontmatter(content string) (body string, fields map[string]string) {
	fields = map[string]string{}
	if !strings.HasPrefix(content, "---") {
		return content, fields
	}
	rest := content[3:]
	end := strings.Index(rest, "---")
	if end == -1 {
		return content, fields
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), ":"); ok {
			fields[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return strings.TrimSpace(rest[end+3:]), fields
}
