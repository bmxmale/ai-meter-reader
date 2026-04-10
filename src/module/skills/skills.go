package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ocr/src/module/check"
)

// Info holds metadata about a single meter skill.
type Info struct {
	File        string
	MeterType   string
	Description string
}

// Dep implements check.Checker for the meter-skills directory.
type Dep struct {
	Dir string
}

func (s Dep) CheckDependency() check.Result {
	info, err := os.Stat(s.Dir)
	if err != nil || !info.IsDir() {
		return check.Result{Name: "skills", OK: false, Message: fmt.Sprintf("skills directory %q not found", s.Dir)}
	}
	return check.Result{Name: "skills", OK: true}
}

// List returns metadata for all skills found in dir.
func List(dir string) []Info {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var result []Info
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		_, fields := parseFrontmatter(string(data))
		desc := fields["description"]
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		result = append(result, Info{
			File:        strings.TrimSuffix(e.Name(), ".md"),
			MeterType:   fields["type"],
			Description: desc,
		})
	}
	return result
}

// Load reads a skill file and returns its system prompt and meter type.
func Load(dir, name string) (system, meterType string, err error) {
	path := filepath.Join(dir, name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("cannot load skill %q: %w", name, err)
	}
	system, frontmatter := parseFrontmatter(string(data))
	return system, frontmatter["type"], nil
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
