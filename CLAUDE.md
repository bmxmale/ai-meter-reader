# CLAUDE.md — ai-meter-reader

## Co to jest

CLI tool `ocr` do odczytu liczników użytkowych (gaz, docelowo woda/prąd) ze zdjęć JPEG i HEIC.
Używa Ollama (lokalnie lub cloud) jako modelu vision. EXIF (data, GPS) jest czytany z oryginału
i dodawany do outputu, ale nigdy nie trafia do modelu — zdjęcie jest strippowane przed wysłaniem.

## Build & Run

```bash
make
./bin/ocr gas ./example/IMG_3290.HEIC
./bin/ocr -o result.json gas ./example/IMG_3290.HEIC gemma4:e4b
```

Wymaga: `ollama` (localhost:11434), `exiftool`, `sips` (macOS built-in).

## Architektura

| Plik | Odpowiedzialność |
|------|-----------------|
| `src/main.go` | CLI (flag, args), orchestracja kroków, formatowanie outputu, spinner, summary |
| `src/module/ollama/ollama.go` | HTTP klient do Ollama `/api/generate` — tylko transport |
| `src/module/exif/exif.go` | Odczyt EXIF via `exiftool -n -json` |
| `src/module/heic/heic.go` | Konwersja HEIC→JPEG (`sips`) + strip metadanych (`exiftool -all=`) |
| `src/module/deps/deps.go` | Weryfikacja zależności systemowych przed uruchomieniem |
| `meter-skills/<name>.md` | Skill = system prompt + frontmatter (name, type, description) |

## Format skilla (meter-skills/)

```markdown
---
name: gas-meter-ocr
type: gas
description: Krótki opis używany w tabeli helpowej.
---

Treść skilla (system prompt dla modelu)...
```

`type` → `meter.type` w outputcie JSON.

## Format outputu JSON

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

- `requires_confirmation: true` gdy `?` występuje w `reading`
- Odpowiedź Ollamy może być w markdown (` ```json ``` `) — `extractJSON()` to obsługuje
- `-o <file>` zapisuje raw JSON do pliku; terminal pokazuje tylko summary

## Przepływ danych

```
oryginał (HEIC/JPG)
  ├─ exiftool → exifData (pamięć)
  ├─ sips [+ exiftool -all=] → /tmp/*.clean.jpg → Ollama
  └─ mergeExif(ollamaResponse, exifData) → JSON output
```

## Domyślny model

`gemma4:31b-cloud` (zdefiniowany w `module/ollama/ollama.go` jako `DefaultOllamaModel`)

## Dodawanie nowego typu licznika

1. Utwórz `meter-skills/<name>.md` z frontmatterem
2. Opcjonalnie dodaj moduł w `module/` jeśli potrzebna specjalna logika parsowania
