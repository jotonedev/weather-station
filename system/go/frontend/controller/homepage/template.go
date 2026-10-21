package homepage

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

var indexTmpl = template.Must(template.ParseFS(templateFS, "templates/index.html"))
