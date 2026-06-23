package main

import (
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"
)

//go:embed web/templates/*.html
var templatesFS embed.FS

//go:embed web/static/*
var staticFS embed.FS

// Phase is one stage of the vapor-compression refrigeration cycle.
type Phase struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	Name      string `json:"name"`
	Component string `json:"component"`
	Location  string `json:"location"`
	StateIn   string `json:"stateIn"`
	StateOut  string `json:"stateOut"`
	Pressure  string `json:"pressure"`
	Temp      string `json:"temp"`
	Summary   string `json:"summary"`
	Detail    string `json:"detail"`
	HomeNote  string `json:"homeNote"`
	Accent    string `json:"accent"`
}

// phases describes the four stages refrigerant moves through, in cycle order.
// This is the single source of truth — the diagram, the side panel, and the
// /api/phases endpoint are all driven from here.
var phases = []Phase{
	{
		ID:        "compression",
		Number:    1,
		Name:      "Compression",
		Component: "Compressor",
		Location:  "Outdoor unit",
		StateIn:   "Cool, low-pressure gas",
		StateOut:  "Hot, high-pressure gas",
		Pressure:  "Low → High",
		Temp:      "~10°C → ~80°C",
		Summary:   "The pump that drives the whole loop. It squeezes the refrigerant gas, spiking its pressure and temperature.",
		Detail:    "The compressor pulls in the cool, low-pressure gas returning from the evaporator and compresses it. Squeezing the gas forces its molecules together, which raises both its pressure and its temperature dramatically — it leaves as a hot, high-pressure gas. Think of it as the heart of the system: nothing moves around the loop without it.",
		HomeNote:  "This is the part that hums and uses most of the electricity. When people say the AC 'kicked on,' they're hearing the compressor start.",
		Accent:    "#ef4444",
	},
	{
		ID:        "condensation",
		Number:    2,
		Name:      "Condensation",
		Component: "Condenser coil",
		Location:  "Outdoor unit",
		StateIn:   "Hot, high-pressure gas",
		StateOut:  "Warm, high-pressure liquid",
		Pressure:  "High",
		Temp:      "~80°C → ~40°C",
		Summary:   "The refrigerant dumps its heat outdoors and condenses back into a liquid.",
		Detail:    "The hot gas flows through the condenser coils while a fan blows outside air across them. The refrigerant gives up its heat to the outdoor air and, as it cools, condenses from a gas into a warm, high-pressure liquid. This is the step that actually moves your home's heat to the outside world.",
		HomeNote:  "That warm air blowing out the top of the unit outside your house? That's literally the heat from inside your home being thrown away.",
		Accent:    "#f59e0b",
	},
	{
		ID:        "expansion",
		Number:    3,
		Name:      "Expansion",
		Component: "Expansion valve",
		Location:  "Indoor unit",
		StateIn:   "Warm, high-pressure liquid",
		StateOut:  "Cold, low-pressure mist",
		Pressure:  "High → Low",
		Temp:      "~40°C → ~5°C",
		Summary:   "A tiny nozzle drops the pressure, and the refrigerant turns ice-cold.",
		Detail:    "The warm liquid is forced through a very small opening — the expansion valve or metering device. On the far side the pressure suddenly drops, and that drop makes the refrigerant expand and its temperature plummet. It emerges as a cold, low-pressure mix of liquid and vapor, ready to soak up heat.",
		HomeNote:  "Same physics as an aerosol can getting cold when you spray it: let a pressurized fluid expand and it chills fast.",
		Accent:    "#38bdf8",
	},
	{
		ID:        "evaporation",
		Number:    4,
		Name:      "Evaporation",
		Component: "Evaporator coil",
		Location:  "Indoor unit",
		StateIn:   "Cold, low-pressure mist",
		StateOut:  "Cool, low-pressure gas",
		Pressure:  "Low",
		Temp:      "~5°C → ~10°C",
		Summary:   "The cold coil absorbs heat from your room's air — this is where cooling happens.",
		Detail:    "A blower pushes warm room air across the cold evaporator coil. The refrigerant absorbs that heat, which makes it boil and evaporate back into a low-pressure gas. The air that comes out the vents is now cooler (and drier, because moisture condenses on the cold coil). The gas heads back to the compressor and the cycle repeats.",
		HomeNote:  "This is the only step you actually feel. It also dehumidifies — that's the water that drips from the indoor unit's drain line.",
		Accent:    "#22d3ee",
	},
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	tmpl := template.Must(template.ParseFS(templatesFS, "web/templates/*.html"))

	staticRoot, err := fs.Sub(staticFS, "web/static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticRoot))))

	mux.HandleFunc("/api/phases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		json.NewEncoder(w).Encode(phases)
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := struct {
			Phases []Phase
			Year   int
		}{
			Phases: phases,
			Year:   time.Now().Year(),
		}
		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			log.Printf("template error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	})

	log.Printf("learn-cooling listening on :%s", port)
	if err := http.ListenAndServe(":"+port, logRequests(mux)); err != nil {
		log.Fatal(err)
	}
}

// logRequests is a tiny middleware so the logs show what's being served.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
