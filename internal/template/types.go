package template

type Size struct {
	Width  string `yaml:"width" json:"width"`
	Height string `yaml:"height" json:"height"`
}

type InputField struct {
	Type        string `yaml:"type" json:"type"`
	Required    bool   `yaml:"required" json:"required"`
	Description string `yaml:"description" json:"description"`
	MaxLength   int    `yaml:"max_length" json:"max_length"`
	Pattern     string `yaml:"pattern" json:"pattern"`
}

type Constant struct {
	Type        string `yaml:"type" json:"type"`
	Value       string `yaml:"value" json:"value"`
	Locked      bool   `yaml:"locked" json:"locked"`
	Description string `yaml:"description" json:"description"`
}

type LayoutElement struct {
	ID          string  `yaml:"id" json:"id"`
	Type        string  `yaml:"type" json:"type"`     // text, barcode, image, line, rect
	Source      string  `yaml:"source" json:"source"` // input, constant (empty = literal)
	Key         string  `yaml:"key" json:"key"`
	Value       string  `yaml:"value" json:"value"`
	X           float64 `yaml:"x" json:"x"`                 // mm
	Y           float64 `yaml:"y" json:"y"`                 // mm
	X2          float64 `yaml:"x2" json:"x2"`               // mm — line end X
	Y2          float64 `yaml:"y2" json:"y2"`               // mm — line end Y
	Width       float64 `yaml:"width" json:"width"`         // mm
	Height      float64 `yaml:"height" json:"height"`       // mm
	FontSize    float64 `yaml:"font_size" json:"font_size"` // pt
	FontFamily  string  `yaml:"font_family" json:"font_family"`
	FontStyle   string  `yaml:"font_style" json:"font_style"` // bold, italic, bold-italic
	Align       string  `yaml:"align" json:"align"`
	MaxWidth    float64 `yaml:"max_width" json:"max_width"`
	BarcodeType string  `yaml:"barcode_type" json:"barcode_type"`
	Src         string  `yaml:"src" json:"src"`
	Color       string  `yaml:"color" json:"color"`               // hex stroke/text color e.g. "#FF0000"
	BorderColor string  `yaml:"border_color" json:"border_color"` // hex border color for textbox, barcode, image
	FillColor   string  `yaml:"fill_color" json:"fill_color"`     // hex fill color for rect/textbox
	TextColor   string  `yaml:"text_color" json:"text_color"`     // hex text color for textbox
	LineWidth   float64 `yaml:"line_width" json:"line_width"`     // mm stroke width
	Padding     float64 `yaml:"padding" json:"padding"`           // mm inner padding for textbox
	Clip        bool    `yaml:"clip" json:"clip"`                 // clip text to textbox bounds
	VisibleIf   string  `yaml:"visible_if" json:"visible_if"`
}

type Template struct {
	Name      string                `yaml:"name" json:"name"`
	Size      Size                  `yaml:"size" json:"size"`
	Inputs    map[string]InputField `yaml:"inputs" json:"inputs"`
	Constants map[string]Constant   `yaml:"constants" json:"constants"`
	Layout    []LayoutElement       `yaml:"layout" json:"layout"`
}
