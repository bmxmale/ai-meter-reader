package exif

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"ocr/src/module/check"
)

// Dep implements check.Checker for the exiftool dependency.
type Dep struct{}

func (Dep) CheckDependency() check.Result {
	if _, err := exec.LookPath("exiftool"); err != nil {
		return check.Result{Name: "exiftool", OK: false, Message: `"exiftool" not found in PATH`}
	}
	if err := exec.Command("exiftool", "-ver").Run(); err != nil {
		return check.Result{Name: "exiftool", OK: false, Message: fmt.Sprintf(`"exiftool" found but failed to run: %v`, err)}
	}
	return check.Result{Name: "exiftool", OK: true}
}

type Data struct {
	DateTime time.Time
	Lat      float64
	Lon      float64
	HasGPS   bool
}

type exiftoolEntry struct {
	DateTimeOriginal string  `json:"DateTimeOriginal"`
	CreateDate       string  `json:"CreateDate"`
	GPSLatitude      float64 `json:"GPSLatitude"`
	GPSLongitude     float64 `json:"GPSLongitude"`
	GPSLatitudeRef   string  `json:"GPSLatitudeRef"`
	GPSLongitudeRef  string  `json:"GPSLongitudeRef"`
}

// Read extracts date/time and GPS coordinates from imagePath using exiftool.
func Read(imagePath string) (*Data, error) {
	out, err := exec.Command(
		"exiftool", "-n", "-json",
		"-DateTimeOriginal", "-CreateDate",
		"-GPSLatitude", "-GPSLongitude",
		"-GPSLatitudeRef", "-GPSLongitudeRef",
		imagePath,
	).Output()
	if err != nil {
		return nil, fmt.Errorf("exiftool: %w", err)
	}

	var entries []exiftoolEntry
	if err := json.Unmarshal(out, &entries); err != nil || len(entries) == 0 {
		return nil, fmt.Errorf("exiftool: cannot parse output")
	}

	e := entries[0]
	d := &Data{}

	raw := e.DateTimeOriginal
	if raw == "" {
		raw = e.CreateDate
	}
	if raw != "" {
		if t, err := time.ParseInLocation("2006:01:02 15:04:05", raw, time.Local); err == nil {
			d.DateTime = t
		}
	}

	if e.GPSLatitude != 0 || e.GPSLongitude != 0 {
		lat, lon := e.GPSLatitude, e.GPSLongitude
		if e.GPSLatitudeRef == "S" {
			lat = -lat
		}
		if e.GPSLongitudeRef == "W" {
			lon = -lon
		}
		d.Lat = lat
		d.Lon = lon
		d.HasGPS = true
	}

	return d, nil
}
