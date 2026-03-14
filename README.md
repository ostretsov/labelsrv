# labelsrv

A configuration-driven label rendering server. Define labels as YAML templates, render them to PDF via HTTP or CLI. Watch the demo on YouTube:

[![Watch the demo](https://img.youtube.com/vi/sGFVaIbe_ns/hqdefault.jpg)](https://youtu.be/sGFVaIbe_ns)

## What it does

- Serves a REST API that accepts JSON input and returns PDF labels
- Templates are YAML files that describe layout, inputs, and constants
- Supports text, barcodes (Code128, QR, EAN13), images, lines, rectangles, and text boxes
- Conditional element visibility based on input values
- Unicode and Cyrillic text out of the box (FreeSans font embedded)
- Auto-generated OpenAPI spec and interactive docs at `/docs`

## Quick start

```sh
# Create a test directory and download the demo template
mkdir /tmp/labelsrv-test && cd /tmp/labelsrv-test
wget https://raw.githubusercontent.com/ostretsov/labelsrv/refs/heads/main/demo/labels/demo.yaml

# Run the server with the current directory as the labels source
docker run --rm -p 8080:8080 -v ./:/labels ostretsov/labelsrv:v1.0.2 dev

# Render a label to PDF using the API and save it to /tmp/demo.pdf
curl -X POST "http://localhost:8080/labels/demo?format=pdf" \
  -H "Content-Type: application/json" \
  -d '{
    "barcode": "TRK1234561789US",
    "delivery_address": "John Doe\n\n123 Main St, Anytown, USA 12345",
    "label_creation_datetime": "2026-03-14 15:12:36",

    "line1_description": "1 x ASICS Men'\''s DYNABLAST 5 Running Shoes",
    "line1_hs_code": "640290",
    "line1_value_usd": "99.95",
    "line1_weight_kg": "0.3",

    "total_weight_kg": "0.4",
    "total_value_usd": "299.71"
  }' \
  -o /tmp/demo.pdf
```

## CLI

```
labelsrv serve     Start the HTTP server
labelsrv dev       Start the server with hot-reload on template changes
labelsrv render    Render a template to PDF from the command line
labelsrv validate  Validate a template file
labelsrv version   Print version and exit
```

### Flags

| Command | Flag | Default | Description |
|---|---|---|---|
| `serve`, `dev` | `--port`, `-p` | `8080` | Port to listen on |
| `serve`, `dev` | `--labels` | `labels` | Directory with template files |
| `serve`, `dev`, `render` | `--fonts` | _(none)_ | Directory with extra TTF fonts |
| `render` | `--output`, `-o` | `label.pdf` | Output PDF path |

```sh
labelsrv render labels/demo.yaml data/example.json -o out.pdf
labelsrv validate labels/demo.yaml
labelsrv serve --port 9090 --labels /etc/labels
```

## HTTP API

### `POST /labels/{template}`

Render a label. Request body is a JSON object with input values.

**JSON response** (default):
```sh
curl -X POST http://localhost:8080/labels/demo \
  -H 'Content-Type: application/json' \
  -d '{"item_name": "Widget"}'
# → {"pdf": "<base64>"}
```

**PDF response** (add `?format=pdf` or `Accept: application/pdf`):
```sh
curl -X POST 'http://localhost:8080/labels/demo?format=pdf' \
  -H 'Content-Type: application/json' \
  -d '{"item_name": "Widget"}' \
  --output label.pdf
```

### Other endpoints

| Endpoint | Description |
|---|---|
| `GET /health` | Server status and list of loaded templates |
| `GET /docs` | Interactive API docs (ReDoc) |
| `GET /openapi.json` | OpenAPI 3.0 specification |

## Template format

Templates are YAML (or JSON) files placed in the labels directory.

```yaml
name: my-label        # unique name, used in the API path

size:
  width: 4in          # units: in, mm, cm, pt
  height: 6in

inputs:  
  tracking_number:
    type: string
    required: true

layout:
  - id: title
    type: text
    value: "SHIPMENT"
    x: 4
    y: 4
    font_size: 18
    font_style: bold
  
  - id: barcode
    type: barcode
    source: input
    key: tracking_number
    barcode_type: code128
    x: 6
    y: 6
    width: 90
    height: 10
```

### Layout element types

#### `text`

Single or multi-line text. Set `max_width` to enable wrapping.

```yaml
- id: label
  type: text
  source: input      # input | constant | (omit for literal value)
  key: recipient
  x: 4
  y: 20
  font_size: 12
  font_style: bold   # bold | italic | bold-italic
  font_family: FreeSans
  color: "#333333"
  max_width: 90      # enables word wrap
  visible_if: "has(recipient)"
```

#### `textbox`

A rectangle container with wrapped text inside. Use `border_color` for a visible border; omit it for an invisible box (useful for layout debugging when you want to see the box bounds without printing a visible outline).

```yaml
- id: notes
  type: textbox
  source: input
  key: delivery_notes
  x: 4
  y: 80
  width: 93
  height: 30
  padding: 3           # inner spacing (mm), default 0
  font_size: 9
  border_color: "#1A237E"   # border color — omit for no border
  fill_color: "#E8EAF6"
  text_color: "#1A237E"
  clip: true               # clip text to box bounds (default false)
```

#### `barcode`

```yaml
- id: tracking_bc
  type: barcode
  source: input
  key: tracking_number
  barcode_type: code128   # code128 | qr | ean13 | code39
  x: 4
  y: 50
  width: 90
  height: 20
  border_color: "#000000"  # optional border — omit for no border
```

#### `rect`

```yaml
- id: header_bg
  type: rect
  x: 0
  y: 0
  width: 101.6
  height: 15
  fill_color: "#1A237E"   # background fill
  color: "#000000"        # border color (omit for no border)
  line_width: 0.5
```

#### `line`

```yaml
- id: separator
  type: line
  x: 0
  y: 20
  x2: 101.6
  y2: 20
  line_width: 0.3
  color: "#BDBDBD"
```

#### `image`

```yaml
- id: logo
  type: image
  src: /path/to/logo.png
  x: 4
  y: 4
  width: 30
  height: 10
  border_color: "#CCCCCC"  # optional border — omit for no border
```

### Value sources

| `source` | Behaviour |
|---|---|
| `input` | Reads from the JSON request field named by `key` |
| `constant` | Reads from the template's `constants` block by `key` |
| _(omit)_ | Uses the literal `value` field |

### Conditional visibility (`visible_if`)

Elements can be shown or hidden based on input values:

```yaml
visible_if: "has(tracking_number)"      # present and non-empty
visible_if: "missing(tracking_number)"  # absent or empty
visible_if: "empty(field)"             # absent or empty string
visible_if: "not_empty(field)"         # present and non-empty
visible_if: "eq(status, shipped)"      # equals value
visible_if: "ne(status, pending)"      # not equal
```

### Field reference

| Field | Types | Description |
|---|---|---|
| `id` | all | Required. Unique element ID |
| `type` | all | Required. Element type |
| `x`, `y` | all | Position in mm |
| `width`, `height` | barcode, rect, image, textbox | Size in mm |
| `x2`, `y2` | line | End position in mm |
| `source` | text, barcode, image, textbox | `input`, `constant`, or omit |
| `key` | text, barcode, image, textbox | Input or constant name |
| `value` | text, textbox | Literal text value |
| `font_size` | text, textbox | Size in pt (default 12) |
| `font_style` | text, textbox | `bold`, `italic`, `bold-italic` |
| `font_family` | text, textbox | Font name (default `FreeSans`); extra fonts loaded via `--fonts` |
| `align` | textbox | `L`, `C`, or `R` (default `L`) |
| `max_width` | text | Wrapping width in mm |
| `color` | text, line, rect | Hex stroke or text color |
| `border_color` | textbox, barcode, image | Hex border color — set to show border, omit for none |
| `fill_color` | rect, textbox | Hex fill color |
| `text_color` | textbox | Hex text color (separate from border) |
| `line_width` | line, rect, textbox | Stroke width in mm (default 0.3) |
| `padding` | textbox | Inner spacing in mm (default 0) |
| `clip` | textbox | Clip text to box bounds (default `false`) |
| `barcode_type` | barcode | `code128`, `qr`, `ean13`, `code39` |
| `src` | image | File path |
| `visible_if` | all | Conditional expression |

