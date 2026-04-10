package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"ocr/src/module/deps"
	exif "ocr/src/module/exif"
	"ocr/src/module/heic"
	"ocr/src/module/skills"
	"ocr/src/module/ui"
	ollama "ocr/src/module/ollama"
)

const defaultSkillsDir = "meter-skills"

func main() {
	outFile := flag.String("o", "", "write JSON output to file")
	skillsDir := flag.String("skills", defaultSkillsDir, "path to meter skills directory")
	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		ui.PrintBanner()
		ui.PrintHelp(*skillsDir)
		ui.ListSkills(*skillsDir)
		deps.Print(deps.Verify(*skillsDir, ""))
		os.Exit(1)
	}

	skillName := args[0]
	imagePath := args[1]

	results := deps.Verify(*skillsDir, imagePath)
	if deps.Fatal(results) {
		deps.Print(results)
		os.Exit(1)
	}

	var model string
	if len(args) >= 3 {
		model = args[2]
	}

	// Step 1: read EXIF from original file before any modification.
	ui.Step("Reading EXIF metadata")
	exifData, _ := exif.Read(imagePath)
	ui.Done()

	// Step 2: produce a stripped JPEG copy for Ollama.
	ui.Step("Preparing image (stripping metadata)")
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
	ui.Done()

	// Step 3: load skill.
	ui.Step("Loading skill: " + skillName)
	system, meterType, err := skills.Load(*skillsDir, skillName)
	if err != nil {
		log.Fatal(err)
	}
	ui.Done()

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
	ui.Spin("Analyzing image with "+modelName, func() {
		result, err = client.Generate(system, "Analyze this image.", cleanPath)
	})
	if err != nil {
		log.Fatal(err)
	}

	output := buildOutput(result, exifData, meterType)

	ui.StepLabel("Processed data:")
	fmt.Println()

	ui.PrintSummary(output)

	if *outFile != "" {
		if err := os.WriteFile(*outFile, []byte(output+"\n"), 0644); err != nil {
			log.Fatalf("cannot write output file: %v", err)
		}
		fmt.Printf("    %s✓%s  Raw output saved to file: %s%s%s\n\n", ui.Green, ui.Reset, ui.Bold, *outFile, ui.Reset)
	}
}

func buildOutput(result string, exifData *exif.Data, meterType string) string {
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
		exifOut := map[string]interface{}{}
		if !exifData.DateTime.IsZero() {
			exifOut["created_at"] = exifData.DateTime.Format("2006-01-02T15:04:05Z07:00")
		}
		if exifData.HasGPS {
			exifOut["gps"] = map[string]float64{
				"lat": exifData.Lat,
				"lon": exifData.Lon,
			}
		}
		if len(exifOut) > 0 {
			out["exif"] = exifOut
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
