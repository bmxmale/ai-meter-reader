# CLAUDE.md — ai-meter-reader

## What is this

CLI tool `ocr` for reading utility meter photos (gas, water/electricity planned) from JPEG and HEIC images.
Uses Ollama (local or cloud) as the vision model. EXIF data (date, GPS) is read from the original
and included in the output, but never sent to the model — the image is stripped before sending.

## Build & Run

```bash
make
./bin/ocr gas ./example/IMG_3290.HEIC
./bin/ocr -o result.json gas ./example/IMG_3290.HEIC gemma4:e4b
```

`make` builds natively — `bin/ocr` for the current platform (macOS arm64 or Linux amd64).

Requires: `ollama` (localhost:11434), `exiftool`. `sips` required only on macOS for HEIC/HEIF files.

## Architecture

| File | Responsibility |
|------|----------------|
| `src/main.go` | CLI (flags, args), step orchestration, `buildOutput` |
| `src/module/ui/ui.go` | All terminal output: spinner, step, banner, help, summary, listSkills |
| `src/module/skills/skills.go` | Loading and listing skills (`Load`, `List`, `parseFrontmatter`) + `Dep` |
| `src/module/ollama/ollama.go` | Ollama HTTP client for `/api/generate` — transport only |
| `src/module/exif/exif.go` | EXIF reader via `exiftool -n -json` + `Dep` |
| `src/module/heic/heic.go` | HEIC→JPEG conversion (`sips`) + metadata stripping + `Dep` |
| `src/module/deps/deps.go` | Dependency verification — iterates over `[]check.Checker` |
| `src/module/check/check.go` | Shared `Checker` interface and `Result` type |
| `meter-skills/<name>.md` | Skill = system prompt + frontmatter (name, type, description) |

## Dependency check system

Each module implements the `check.Checker` interface:

```go
type Checker interface {
    CheckDependency() Result
}
```

`deps.Verify(skillsDir, imagePath)` iterates over a list of checkers:
- `exif.Dep{}` — verifies `exiftool`
- `heic.Dep{ImagePath}` — verifies `sips` (macOS + HEIC only), or returns an error for HEIC on non-darwin
- `skills.Dep{Dir}` — verifies the skills directory exists

## Skill format (meter-skills/)

```markdown
---
name: gas-meter-ocr
type: gas
description: "Short description shown in the help table."
---

Skill body (system prompt for the model)...
```

`type` → `meter.type` in JSON output. Values containing colons in `description` must be quoted.

## JSON output format

```json
{
  "meter": {
    "type": "gas",
    "serial_number": "02534331",
    "requires_confirmation": true,
    "value": {
      "reading": "03178.82?",
      "integer": "03178",
      "decimal": "82?",
      "unit": "m³"
    }
  },
  "exif": {
    "created_at": "2023-11-01T11:11:43+01:00",
    "gps": { "lat": 54.36, "lon": 18.42 }
  }
}
```

- `requires_confirmation: true` when `?` appears in `reading`
- Ollama may respond with markdown-fenced JSON (` ```json ``` `) — `extractJSON()` handles this
- `-o <file>` writes raw JSON to a file; the terminal shows only the summary

## Data flow

```
original (HEIC/JPG)
  ├─ exif.Read() → exifData (in memory)
  ├─ heic.ToJPG() / heic.StripJPG() → /tmp/*.clean.jpg → Ollama
  └─ buildOutput(ollamaResponse, exifData, meterType) → JSON output
```

## Default model

`gemma4:31b-cloud` (defined in `module/ollama/ollama.go` as `DefaultOllamaModel`)

## Adding a new meter type

1. Create `meter-skills/<name>.md` with frontmatter
2. Optionally add a module under `module/` if custom parsing logic is needed
3. If the module requires an external dependency — add a `Dep` implementing `check.Checker` and register it in `deps.Verify`
