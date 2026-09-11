package server

import (
	"html/template"
	"net/http"
	"strings"
)

const defaultHTMLLocation = "Berlin"

var htmlPageTemplate = template.Must(template.New("htmlPage").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>Weather for {{.Location}}</title>
</head>
<body>
	<h1>Weather for {{.Location}}</h1>
	<section id="current">
		<p>Current conditions: TODO</p>
	</section>
	<section id="forecast">
		<p>Forecast: TODO</p>
	</section>
</body>
</html>
`))

// htmlPageHandler returns a bare HTML skeleton for a single location, taken
// either from the URL path (/html/<location>) or the ?location= query
// parameter, falling back to defaultHTMLLocation. It exists to test the
// HTML-page flow independently of the full weather rendering pipeline.
func htmlPageHandler(w http.ResponseWriter, r *http.Request) {
	location := strings.TrimPrefix(r.URL.Path, "/html/")
	location = strings.Trim(location, "/")

	if location == "" {
		location = r.URL.Query().Get("location")
	}

	if location == "" {
		location = defaultHTMLLocation
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_ = htmlPageTemplate.Execute(w, struct{ Location string }{Location: location})
}
