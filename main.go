package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed web/templates/*.html
var templatesFS embed.FS

//go:embed web/static/*
var staticFS embed.FS

// canonicalHost is the one hostname we want search engines to index. Requests
// arriving on the raw Render hostname get redirected here so ranking signals
// aren't split across two domains serving identical content.
const canonicalHost = "learncooling.com"

const siteURL = "https://" + canonicalHost

// PageMeta is everything the shared layout needs for the <head>.
type PageMeta struct {
	Title       string
	Description string
	Path        string // canonical path, e.g. "/problems/bad-capacitor"
	SchemaJSON  template.JS
}

func (m PageMeta) Canonical() string { return siteURL + m.Path }

// pageData is what every template receives. Fields not relevant to a given page
// are simply left zero.
type pageData struct {
	Meta         PageMeta
	Year         int
	Modes        []Mode
	Phases       []Phase
	Fans         []Fan
	Faults       []Fault
	Refrigerants []Refrigerant

	// Set on /problems/{slug} only.
	Fault    *Fault
	Related  []*Fault
	RelPhase *Phase
}

// Each page is parsed together with the shared layout so they can each define
// their own "content" block without colliding.
func mustPage(name string) *template.Template {
	return template.Must(template.ParseFS(templatesFS,
		"web/templates/layout.html", "web/templates/"+name))
}

var (
	indexTmpl       = mustPage("index.html")
	problemTmpl     = mustPage("problem.html")
	problemsTmpl    = mustPage("problems.html")
	refrigerantTmpl = mustPage("refrigerants.html")
	heatPumpTmpl    = mustPage("heatpumps.html")
	notFoundTmpl    = mustPage("404.html")
)

// cyclePayload is the full dataset the page needs, served as one JSON document.
type cyclePayload struct {
	Modes        []Mode        `json:"modes"`
	Refrigerants []Refrigerant `json:"refrigerants"`
	Phases       []Phase       `json:"phases"`
	Fans         []Fan         `json:"fans"`
	Faults       []Fault       `json:"faults"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	staticRoot, err := fs.Sub(staticFS, "web/static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticRoot))))

	mux.HandleFunc("GET /api/cycle", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, cyclePayload{
			Modes:        modes,
			Refrigerants: refrigerants,
			Phases:       phases,
			Fans:         fans,
			Faults:       faults,
		})
	})

	// Kept for backwards compatibility — just the phases.
	mux.HandleFunc("GET /api/phases", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, phases)
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Crawlers need /api/ — the home page builds its diagram from /api/cycle, so
	// blocking it would leave Googlebot looking at an empty shell.
	mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: %s/sitemap.xml\n", siteURL)
	})

	mux.HandleFunc("GET /sitemap.xml", handleSitemap)

	mux.HandleFunc("GET /{$}", handleIndex)
	mux.HandleFunc("GET /problems", handleProblems)
	mux.HandleFunc("GET /problems/{$}", handleProblems)
	mux.HandleFunc("GET /problems/{slug}", handleProblem)
	mux.HandleFunc("GET /refrigerants", handleRefrigerants)
	mux.HandleFunc("GET /heat-pumps", handleHeatPumps)

	// Anything else.
	mux.HandleFunc("/", handleNotFound)

	log.Printf("learn-cooling listening on :%s", port)
	if err := http.ListenAndServe(":"+port, logRequests(canonicalRedirect(mux))); err != nil {
		log.Fatal(err)
	}
}

// ---- handlers --------------------------------------------------------------

const homeDesc = "An interactive diagram of the refrigeration cycle. See how your home AC and heat pump move heat, compare refrigerants, and learn what commonly goes wrong."

func handleIndex(w http.ResponseWriter, r *http.Request) {
	render(w, indexTmpl, pageData{
		Meta: PageMeta{
			Title:       "How Air Conditioning Works: The Refrigeration Cycle Explained",
			Description: homeDesc,
			Path:        "/",
			SchemaJSON:  homeSchema(),
		},
		Modes:        modes,
		Phases:       phases,
		Fans:         fans,
		Faults:       faults,
		Refrigerants: refrigerants,
	})
}

func handleProblems(w http.ResponseWriter, r *http.Request) {
	render(w, problemsTmpl, pageData{
		Meta: PageMeta{
			Title:       "AC Troubleshooting: 9 Common Problems and What Causes Them",
			Description: "Why your AC isn't cooling — frozen coils, refrigerant leaks, bad capacitors, dirty coils and more, each explained by what it does to the refrigeration cycle.",
			Path:        "/problems",
			SchemaJSON:  collectionSchema(),
		},
		Faults: faults,
	})
}

func handleProblem(w http.ResponseWriter, r *http.Request) {
	f, ok := faultBySlug[r.PathValue("slug")]
	if !ok {
		handleNotFound(w, r)
		return
	}
	render(w, problemTmpl, pageData{
		Meta: PageMeta{
			Title:       f.MetaTitle,
			Description: f.MetaDesc,
			Path:        "/problems/" + f.Slug,
			SchemaJSON:  problemSchema(f),
		},
		Fault:    f,
		Related:  relatedFaults(f),
		RelPhase: phaseByID(f.Phase),
	})
}

func handleRefrigerants(w http.ResponseWriter, r *http.Request) {
	render(w, refrigerantTmpl, pageData{
		Meta: PageMeta{
			Title:       "AC Refrigerants Compared: R-410A vs R-32 vs R-134a vs Ammonia",
			Description: "What refrigerant actually does, and how R-410A, R-32, R-134a and R-717 (ammonia) compare on pressure, discharge temperature, safety and global-warming impact.",
			Path:        "/refrigerants",
			SchemaJSON:  articleSchema("AC Refrigerants Compared: R-410A vs R-32 vs R-134a vs Ammonia", "/refrigerants", "Refrigerants"),
		},
		Refrigerants: refrigerants,
		Modes:        modes,
	})
}

func handleHeatPumps(w http.ResponseWriter, r *http.Request) {
	render(w, heatPumpTmpl, pageData{
		Meta: PageMeta{
			Title:       "Do Heat Pumps Work in Cold Weather? How the Cycle Reverses",
			Description: "A heat pump is an air conditioner running backwards. How the reversing valve flips the loop, why it can pull heat from freezing air, and what defrost mode is doing.",
			Path:        "/heat-pumps",
			SchemaJSON:  articleSchema("Do Heat Pumps Work in Cold Weather? How the Cycle Reverses", "/heat-pumps", "Heat pumps"),
		},
		Phases: phases,
		Modes:  modes,
	})
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	renderStatus(w, notFoundTmpl, pageData{
		Meta: PageMeta{
			Title:       "Page not found — Learn Cooling",
			Description: "That page doesn't exist.",
			Path:        r.URL.Path,
		},
		Faults: faults,
	})
}

func handleSitemap(w http.ResponseWriter, r *http.Request) {
	paths := []string{"/", "/problems", "/refrigerants", "/heat-pumps"}
	for i := range faults {
		paths = append(paths, "/problems/"+faults[i].Slug)
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprint(w, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`+"\n")
	for _, p := range paths {
		priority := "0.8"
		if p == "/" {
			priority = "1.0"
		}
		fmt.Fprintf(w, "  <url>\n    <loc>%s%s</loc>\n    <changefreq>monthly</changefreq>\n    <priority>%s</priority>\n  </url>\n",
			siteURL, p, priority)
	}
	fmt.Fprint(w, "</urlset>\n")
}

func render(w http.ResponseWriter, t *template.Template, d pageData) {
	renderStatus(w, t, d)
}

func renderStatus(w http.ResponseWriter, t *template.Template, d pageData) {
	d.Year = time.Now().Year()
	if err := t.ExecuteTemplate(w, "layout", d); err != nil {
		log.Printf("template error (%s): %v", d.Meta.Path, err)
	}
}

// ---- structured data -------------------------------------------------------

func website() map[string]any {
	return map[string]any{
		"@type":       "WebSite",
		"@id":         siteURL + "/#website",
		"url":         siteURL + "/",
		"name":        "Learn Cooling",
		"description": homeDesc,
		"inLanguage":  "en",
	}
}

// breadcrumb builds a BreadcrumbList from Home down to the current page.
func breadcrumb(trail ...[2]string) map[string]any {
	items := []any{}
	for i, t := range trail {
		items = append(items, map[string]any{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     t[0],
			"item":     siteURL + t[1],
		})
	}
	return map[string]any{"@type": "BreadcrumbList", "itemListElement": items}
}

func marshalSchema(graph ...any) template.JS {
	b, err := json.Marshal(map[string]any{
		"@context": "https://schema.org",
		"@graph":   graph,
	})
	if err != nil {
		log.Printf("schema marshal: %v", err)
		return template.JS("{}")
	}
	// json.Marshal escapes <, > and & by default, so this is safe inline.
	return template.JS(b)
}

func homeSchema() template.JS {
	return marshalSchema(website(), map[string]any{
		"@type":            "TechArticle",
		"@id":              siteURL + "/#article",
		"isPartOf":         map[string]any{"@id": siteURL + "/#website"},
		"mainEntityOfPage": siteURL + "/",
		"headline":         "How Air Conditioning Works: The Refrigeration Cycle Explained",
		"description":      homeDesc,
		"image":            siteURL + "/static/og.png",
		"inLanguage":       "en",
		"about": []any{
			map[string]any{"@type": "Thing", "name": "Refrigeration cycle"},
			map[string]any{"@type": "Thing", "name": "Air conditioning"},
			map[string]any{"@type": "Thing", "name": "Heat pump"},
		},
	})
}

func collectionSchema() template.JS {
	items := []any{}
	for i := range faults {
		items = append(items, map[string]any{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     faults[i].Question,
			"url":      siteURL + "/problems/" + faults[i].Slug,
		})
	}
	return marshalSchema(website(),
		map[string]any{
			"@type":            "CollectionPage",
			"mainEntityOfPage": siteURL + "/problems",
			"headline":         "AC Troubleshooting: Common Problems and What Causes Them",
			"mainEntity":       map[string]any{"@type": "ItemList", "itemListElement": items},
		},
		breadcrumb([2]string{"Home", "/"}, [2]string{"Troubleshooting", "/problems"}),
	)
}

func problemSchema(f *Fault) template.JS {
	return marshalSchema(website(),
		map[string]any{
			"@type":            "TechArticle",
			"mainEntityOfPage": siteURL + "/problems/" + f.Slug,
			"headline":         f.Question,
			"description":      f.MetaDesc,
			"image":            siteURL + "/static/og.png",
			"inLanguage":       "en",
			"isPartOf":         map[string]any{"@id": siteURL + "/#website"},
		},
		breadcrumb([2]string{"Home", "/"}, [2]string{"Troubleshooting", "/problems"}, [2]string{f.Title, "/problems/" + f.Slug}),
	)
}

func articleSchema(headline, path, crumb string) template.JS {
	return marshalSchema(website(),
		map[string]any{
			"@type":            "TechArticle",
			"mainEntityOfPage": siteURL + path,
			"headline":         headline,
			"image":            siteURL + "/static/og.png",
			"inLanguage":       "en",
			"isPartOf":         map[string]any{"@id": siteURL + "/#website"},
		},
		breadcrumb([2]string{"Home", "/"}, [2]string{crumb, path}),
	)
}

// ---- middleware / util -----------------------------------------------------

// canonicalRedirect 301s the *.onrender.com hostname over to the real domain so
// the two don't compete as duplicate content. It deliberately leaves /healthz
// alone (Render probes the service on its own hostname) and ignores localhost so
// local development still works.
func canonicalRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if i := strings.IndexByte(host, ':'); i >= 0 {
			host = host[:i]
		}
		if r.URL.Path != "/healthz" && strings.HasSuffix(host, ".onrender.com") {
			http.Redirect(w, r, siteURL+r.URL.RequestURI(), http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	json.NewEncoder(w).Encode(v)
}

// logRequests is a tiny middleware so the logs show what's being served.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
