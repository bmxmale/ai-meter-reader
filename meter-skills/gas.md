---
name: gas-meter-ocr
type: gas
description: "Analyzes a photo of a G4 gas meter and displays a structured JSON with meter reading and serial number. Use this skill whenever the user provides a gas meter photo and wants to extract readings, when the user mentions \"licznik gazu\" (Polish: gas meter), \"odczyt\" (reading), \"numer seryjny\" (serial number), or asks to describe/analyze a gas meter image."
---

# Gas Meter OCR (G4) — Extract Reading and Serial Number

Your task: read a G4 gas meter image and display the extracted data as JSON.

## Meter display format

G4 meters have exactly **8 display positions**:
- **5 integer digits** (white/black drums) — whole cubic meters
- **3 decimal digits** (red-framed drums) — liters

The split is always at position 5 from the left. Leading zeros are shown on the display (e.g. `0 3 1 7 8`).

## Core rule — when in doubt, use `?`

If you cannot read a digit with confidence, write `?` in its place. Never guess. Never interpolate from context. A `?` in the output is useful data; a wrong digit is worse than nothing.

This applies to:
- Any digit in the meter display that is obscured, blurry, or partially visible (especially the last decimal digit which may be mid-rotation)
- Any digit in the serial number that is hard to read

## Output format

Display the result as JSON in the chat.

```json
{
  "meter_reading": {
    "integer_part": "03178",
    "decimal_part": "829",
    "full_value": "03178.829",
    "unit": "m³"
  },
  "serial_number": "02534331"
}
```

Rules for the fields:

- `integer_part` — exactly 5 digits, preserve leading zeros as shown on the display (e.g. `"03178"`, not `"3178"`)
- `decimal_part` — exactly 3 digits from the red-framed section; if a digit is unreadable, use `?` (e.g. `"82?"`)
- `full_value` — `integer_part + "." + decimal_part` with any `?` preserved (e.g. `"03178.82?"`)
- `unit` — always `"m³"` for G4 gas meters
- `serial_number` — the serial number as printed on the meter label, typically prefixed with `Nr`; use `?` for unreadable characters

Omit any field you cannot determine at all.

## How to read the image

1. **Display digits**: Read all 8 positions left to right. The first 5 are the integer part, the last 3 (in the red frame) are the decimal part.
2. **Serial number**: Look for a label on the meter body, often prefixed with `Nr`. Prefer the human-readable number over the barcode.
3. **Confidence check**: For each digit, ask — am I certain? The last decimal digit is often mid-rotation and may be ambiguous — use `?` if so.

## What to exclude

Do not include: manufacturer name, group/class (G4, etc.), technical specs (Qmax, pmax, etc.), certification marks, model number, production year, or any other data. Only `meter_reading` and `serial_number`.