package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"github.com/chubin/wttr.in/internal/domain"
	"github.com/chubin/wttr.in/internal/weather"
)

const defaultHTMLLocation = "Berlin"

// rainyKeywords are matched (case-insensitively) against the weather
// description to decide whether to show the rainy background video.
var rainyKeywords = []string{"rain", "drizzle", "shower", "thunder", "sleet", "snow"}

type htmlPageData struct {
	Location    string
	Country     string
	Description string
	TempC       string
	FeelsLikeC  string
	Humidity    string
	WindKmph    string
	VideoURL    string
	Error       string
}

var htmlPageTemplate = template.Must(template.New("htmlPage").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Weather for {{.Location}}</title>
	<style>
		html, body { margin: 0; height: 100%; font-family: sans-serif; color: #fff; }
		.background-video {
			position: fixed; top: 0; left: 0;
			width: 100%; height: 100%;
			object-fit: cover; z-index: -1;
		}
		.card {
			max-width: 360px;
			margin: 10vh auto 0;
			padding: 24px 28px;
			background: rgba(0, 0, 0, 0.45);
			border-radius: 12px;
			backdrop-filter: blur(4px);
		}
		.card h1 { margin: 0 0 4px; font-size: 1.6em; }
		.card .desc { margin: 0 0 16px; opacity: 0.85; }
		.card .temp { font-size: 3em; margin: 0; }
		.card dl { display: grid; grid-template-columns: auto 1fr; gap: 4px 12px; margin-top: 16px; }
		.card dt { opacity: 0.7; }
		.card dd { margin: 0; }
	</style>
</head>
<body>
	{{if .VideoURL}}
	<video class="background-video" autoplay muted loop playsinline>
		<source src="{{.VideoURL}}" type="video/mp4">
	</video>
	{{end}}

	<div class="card">
		<h1>{{.Location}}{{if .Country}}, {{.Country}}{{end}}</h1>
		{{if .Error}}
			<p class="desc">{{.Error}}</p>
		{{else}}
			<p class="desc">{{.Description}}</p>
			<p class="temp">{{.TempC}}&deg;C</p>
			<dl>
				<dt>Feels like</dt><dd>{{.FeelsLikeC}}&deg;C</dd>
				<dt>Humidity</dt><dd>{{.Humidity}}%</dd>
				<dt>Wind</dt><dd>{{.WindKmph}} km/h</dd>
			</dl>
		{{end}}
	</div>
</body>
</html>
`))

// isRainy classifies a weather description as rainy/snowy vs. sunny/clear.
func isRainy(description string) bool {
	desc := strings.ToLower(description)
	for _, kw := range rainyKeywords {
		if strings.Contains(desc, kw) {
			return true
		}
	}
	return false
}

// newHTMLPageHandler returns a handler that resolves a location (from the
// URL path /html/<location> or the ?location= query parameter, defaulting
// to defaultHTMLLocation), fetches its current weather via the existing
// WeatherService pipeline, and renders a page with a full-screen background
// video (sunny/rainy) plus a card summarizing the payload.
func newHTMLPageHandler(ws *weather.WeatherService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		locationName := strings.TrimPrefix(r.URL.Path, "/html/")
		locationName = strings.Trim(locationName, "/")

		if locationName == "" {
			locationName = r.URL.Query().Get("location")
		}
		if locationName == "" {
			locationName = defaultHTMLLocation
		}

		data := htmlPageData{Location: locationName}

		loc, err := ws.Locator.GetLocation(locationName)
		if err != nil {
			data.Error = "Could not find that location."
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = htmlPageTemplate.Execute(w, data)
			return
		}
		data.Location = loc.Name
		data.Country = loc.Country

		weatherBytes, err := ws.Weatherer.GetWeather(loc.Latitude, loc.Longitude, "en")
		if err != nil {
			data.Error = "Weather data is currently unavailable."
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = htmlPageTemplate.Execute(w, data)
			return
		}

		var parsed domain.Weather
		if jsonErr := json.Unmarshal(weatherBytes, &parsed); jsonErr != nil || len(parsed.CurrentCondition) == 0 {
			data.Error = "Weather data is currently unavailable."
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = htmlPageTemplate.Execute(w, data)
			return
		}

		current := parsed.CurrentCondition[0]
		description := ""
		if len(current.WeatherDesc) > 0 {
			description = strings.TrimSpace(current.WeatherDesc[0].Value)
		}

		data.Description = description
		data.TempC = current.TempC
		data.FeelsLikeC = current.FeelsLikeC
		data.Humidity = current.Humidity
		data.WindKmph = current.WindspeedKmph

		if isRainy(description) {
			data.VideoURL = "/files/videos/raining.mp4"
		} else {
			data.VideoURL = "/files/videos/sunny.mp4"
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = htmlPageTemplate.Execute(w, data)
	}
}
