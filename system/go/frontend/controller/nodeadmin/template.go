package nodeadmin

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

var nodesTmpl = template.Must(template.ParseFS(templateFS, "templates/nodes.html"))
