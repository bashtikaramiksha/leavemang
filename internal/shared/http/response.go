package http

import (
	"html/template"
	"net/http"
)

// RenderHTML parses and executes template files with data.
func RenderHTML(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}, filenames ...string) {
	tmpl, err := template.ParseFiles(filenames...)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(statusCode)
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// IsHTMX checks if the request is an HTMX request.
func IsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
