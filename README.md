# AI Meter Reader

A CLI tool that reads utility meter photos and extracts readings using a local or cloud-based vision model via [Ollama](https://ollama.com).

EXIF metadata (date, GPS coordinates) is preserved in the output JSON but automatically stripped from the image before it is sent to the model — no location or author data leaves your machine embedded in the photo.

## Features

- Reads **gas meters** (G4) from JPEG and HEIC photos
- Extracts meter reading and serial number via a vision model
- Strips all EXIF/IPTC metadata from the image before sending to Ollama
- Includes photo date and GPS in the output JSON (read from original, never sent)
- Supports local and cloud Ollama models
- Outputs structured JSON with a human-readable summary
- Optional file output with `-o`

## Requirements

| Tool | Purpose |
|------|---------|
| [Ollama](https://ollama.com) | Runs the vision model locally or connects to cloud |
| [exiftool](https://exiftool.org) | Reads and strips EXIF metadata |
| [sips](https://ss64.com/mac/sips.html) | Converts HEIC to JPEG (macOS built-in) |
| Go 1.23+ | Build the binary |

Install on macOS:
```bash
brew install ollama exiftool
ollama pull gemma4:31b-cloud
```

## Build

```bash
make
```

## Usage

```bash
./bin/ocr [flags] <skill> <image> [model]
```

**Flags:**
- `-o <file>` — write raw JSON output to file

**Examples:**

```bash
# Default model (cloud)
./bin/ocr gas ./example/IMG_3290.HEIC
./bin/ocr -o result.json gas ./example/IMG_3290.HEIC

# Local models (run on your machine)
./bin/ocr gas ./example/IMG_3290.HEIC gemma4:e4b
./bin/ocr gas ./example/IMG_3290.HEIC qwen2.5vl:3b
./bin/ocr gas ./example/IMG_3290.HEIC qwen2.5vl:7b

# Cloud models (via Ollama cloud endpoint)
./bin/ocr gas ./example/IMG_3290.HEIC gemma4:31b-cloud
```

Available cloud vision models: https://ollama.com/search?c=vision&c=cloud

## Output

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
    "gps": {
      "lat": 54.3605277777778,
      "lon": 18.4201361111111
    }
  }
}
```

`requires_confirmation: true` means at least one digit was unreadable (`?`) and the reading should be verified manually.

## Adding a new meter skill

Create `meter-skills/<name>.md` with a YAML frontmatter:

```markdown
---
name: water-meter-ocr
type: water
description: Reads a water meter and extracts the reading and serial number.
---

# Your skill instructions here...
```

Then run:

```bash
./bin/ocr water ./example/photo.jpg
```

## Project structure

```
.
├── src/
│   ├── main.go                   # CLI, orchestration, output formatting
│   └── module/
│       ├── deps/deps.go          # Dependency checker (exiftool, sips)
│       ├── exif/exif.go          # EXIF reader via exiftool
│       ├── heic/heic.go          # HEIC→JPEG converter + metadata stripper
│       └── ollama/ollama.go      # Ollama HTTP client
├── bin/                          # Compiled binaries (git-ignored)
├── example/                      # Example meter photos
└── meter-skills/
    └── gas.md                    # Gas meter skill (prompt + instructions)
```
