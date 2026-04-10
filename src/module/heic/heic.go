package heic

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"ocr/src/module/check"
)

// IsHEIC reports whether the file has a HEIC/HEIF extension.
func IsHEIC(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".heic" || ext == ".heif"
}

// Dep implements check.Checker for HEIC/sips dependency.
type Dep struct {
	ImagePath string
}

func (d Dep) CheckDependency() check.Result {
	if d.ImagePath != "" && IsHEIC(d.ImagePath) && runtime.GOOS != "darwin" {
		return check.Result{Name: "heic support", OK: false, Message: "HEIC/HEIF conversion requires sips (macOS only) — convert the image to JPEG first"}
	}
	if runtime.GOOS != "darwin" {
		return check.Result{}
	}
	// On macOS: verify sips only when image is HEIC or type is unknown.
	if d.ImagePath != "" && !IsHEIC(d.ImagePath) {
		return check.Result{}
	}
	if _, err := exec.LookPath("sips"); err != nil {
		return check.Result{Name: "sips", OK: false, Message: `"sips" not found in PATH`}
	}
	return check.Result{Name: "sips", OK: true}
}

// ToJPG converts a HEIC file to a metadata-free JPEG using sips + exiftool.
// Returns the path to a temporary file. The caller must remove it when done.
// Only supported on macOS (requires sips).
func ToJPG(src string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("HEIC/HEIF conversion requires sips (macOS only) — convert the image to JPEG first")
	}
	dst := tempJPGPath(src)

	cmd := exec.Command("sips", "-s", "format", "jpeg", src, "--out", dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("sips conversion failed: %w\n%s", err, out)
	}

	if err := stripMetadata(dst); err != nil {
		os.Remove(dst)
		return "", err
	}

	return dst, nil
}

// StripJPG creates a metadata-free JPEG copy of a JPEG file using exiftool.
// Returns the path to a temporary file. The caller must remove it when done.
func StripJPG(src string) (string, error) {
	dst := tempJPGPath(src)

	if out, err := exec.Command("exiftool", "-all=", "-o", dst, src).CombinedOutput(); err != nil {
		return "", fmt.Errorf("exiftool strip failed: %w\n%s", err, out)
	}

	return dst, nil
}

// stripMetadata removes all metadata from a JPEG file in-place using exiftool.
func stripMetadata(path string) error {
	if out, err := exec.Command("exiftool", "-all=", "-overwrite_original", path).CombinedOutput(); err != nil {
		return fmt.Errorf("exiftool strip failed: %w\n%s", err, out)
	}
	return nil
}

func tempJPGPath(src string) string {
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	return filepath.Join(os.TempDir(), base+".clean.jpg")
}
