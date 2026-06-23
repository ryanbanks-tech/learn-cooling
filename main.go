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

// Phase is one stage of the vapor-compression refrigeration cycle. The numeric
// pressures and temperatures aren't stored here — they're computed in the
// browser from the selected refrigerant and mode, because they depend on both.
type Phase struct {
	ID           string `json:"id"`
	Number       int    `json:"number"`
	Name         string `json:"name"`
	Component    string `json:"component"`
	Role         string `json:"role"`         // compressor | condenser | expansion | evaporator
	PressureSide string `json:"pressureSide"` // low | high | lowHigh | highLow
	Accent       string `json:"accent"`
	Summary      string `json:"summary"`
	Detail       string `json:"detail"`
	CoolLocation string `json:"coolLocation"` // where this happens in cooling mode
	HeatLocation string `json:"heatLocation"` // where this happens in heating mode
	CoolNote     string `json:"coolNote"`     // "in your home" note when cooling
	HeatNote     string `json:"heatNote"`     // "in your home" note when heating
}

// Mode is cooling or heating (heat-pump). It sets the saturation temperatures
// the cycle runs between and explains which way heat is being moved.
type Mode struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	EvapTempC int    `json:"evapTempC"` // evaporator (low-side) saturation temp
	CondTempC int    `json:"condTempC"` // condenser (high-side) saturation temp
	Banner    string `json:"banner"`
}

// SatPoint is one (temperature, saturation pressure) reading for a refrigerant.
type SatPoint struct {
	TempC int     `json:"tempC"`
	Bar   float64 `json:"bar"` // absolute saturation pressure, bar
}

// Refrigerant carries representative real-world properties. SatCurve gives the
// absolute saturation pressure at the temperatures the cycle actually runs at,
// so the browser can show the true high/low pressures for each fluid.
type Refrigerant struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	AltName       string     `json:"altName"`
	Use           string     `json:"use"`
	SatCurve      []SatPoint `json:"satCurve"`
	DischargeTemp int        `json:"dischargeTemp"` // typical compressor discharge gas temp, °C
	GWP           int        `json:"gwp"`           // global warming potential (CO2 = 1)
	Safety        string     `json:"safety"`        // ASHRAE 34 class + plain words
	Color         string     `json:"color"`
	Note          string     `json:"note"`
}

// Fault is something that commonly goes wrong with a home system, tied to the
// phase of the cycle it disrupts.
type Fault struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Icon     string `json:"icon"`
	Phase    string `json:"phase"` // related phase id
	Severity string `json:"severity"`
	Cause    string `json:"cause"`
	Effect   string `json:"effect"`
	Symptoms string `json:"symptoms"`
	Fix      string `json:"fix"`
}

var modes = []Mode{
	{
		ID:        "cooling",
		Name:      "Cooling",
		Icon:      "❄️",
		EvapTempC: 5,
		CondTempC: 45,
		Banner:    "Heat is pulled out of your rooms and dumped outdoors.",
	},
	{
		ID:        "heating",
		Name:      "Heating (heat pump)",
		Icon:      "🔥",
		EvapTempC: -7,
		CondTempC: 45,
		Banner:    "A reversing valve flips the loop: heat is pulled from the cold outdoor air and released into your rooms.",
	},
}

// phases lists the four stages in cycle order. They're the single source of
// truth for the diagram, the side panel, and /api/phases.
var phases = []Phase{
	{
		ID:           "compression",
		Number:       1,
		Name:         "Compression",
		Component:    "Compressor",
		Role:         "compressor",
		PressureSide: "lowHigh",
		Accent:       "#ef4444",
		Summary:      "The pump that drives the whole loop. It squeezes the refrigerant gas, spiking its pressure and temperature.",
		Detail:       "The compressor pulls in the cool, low-pressure gas returning from the evaporator and compresses it. Squeezing the gas forces its molecules together, which raises both its pressure and its temperature dramatically — it leaves as a hot, high-pressure gas. Think of it as the heart of the system: nothing moves around the loop without it.",
		CoolLocation: "Outdoor unit",
		HeatLocation: "Outdoor unit",
		CoolNote:     "This is the part that hums and uses most of the electricity. When people say the AC 'kicked on,' they're hearing the compressor start.",
		HeatNote:     "It works harder in heating: the colder the outdoor air, the bigger the squeeze it has to make, so the discharge gas runs even hotter than in cooling.",
	},
	{
		ID:           "condensation",
		Number:       2,
		Name:         "Condensation",
		Component:    "Condenser coil",
		Role:         "condenser",
		PressureSide: "high",
		Accent:       "#f59e0b",
		Summary:      "The refrigerant dumps its heat and condenses back into a liquid. This is the hot end of the loop.",
		Detail:       "The hot gas flows through the condenser coil while a fan blows air across it. The refrigerant gives up its heat and, as it cools, condenses from a gas into a warm, high-pressure liquid. This is the step that actually delivers heat to wherever you want it.",
		CoolLocation: "Outdoors",
		HeatLocation: "Indoors — this is what warms your home",
		CoolNote:     "That warm air blowing out the top of the outdoor unit? That's literally the heat from inside your home being thrown away.",
		HeatNote:     "In heat-pump mode this coil is indoors. The 'waste heat' is now the whole point — it's what blows warm air into your rooms.",
	},
	{
		ID:           "expansion",
		Number:       3,
		Name:         "Expansion",
		Component:    "Expansion valve",
		Role:         "expansion",
		PressureSide: "highLow",
		Accent:       "#38bdf8",
		Summary:      "A tiny nozzle drops the pressure, and the refrigerant turns ice-cold.",
		Detail:       "The warm liquid is forced through a very small opening — the expansion valve or metering device. On the far side the pressure suddenly drops, and that drop makes the refrigerant expand and its temperature plummet. It emerges as a cold, low-pressure mix of liquid and vapor, ready to soak up heat.",
		CoolLocation: "Feeds the indoor coil",
		HeatLocation: "Feeds the outdoor coil",
		CoolNote:     "Same physics as an aerosol can getting cold when you spray it: let a pressurized fluid expand and it chills fast.",
		HeatNote:     "In heating it has to chill the refrigerant below the freezing outdoor air so it can still absorb heat — that's why heat pumps need defrost cycles.",
	},
	{
		ID:           "evaporation",
		Number:       4,
		Name:         "Evaporation",
		Component:    "Evaporator coil",
		Role:         "evaporator",
		PressureSide: "low",
		Accent:       "#22d3ee",
		Summary:      "The cold coil absorbs heat from the air around it. This is the cold end of the loop.",
		Detail:       "A blower pushes air across the cold evaporator coil. The refrigerant absorbs heat from that air, which makes it boil and evaporate back into a low-pressure gas. Whatever air passes the coil comes out colder. The gas then heads back to the compressor and the cycle repeats.",
		CoolLocation: "Indoors",
		HeatLocation: "Outdoors — it steals heat from cold air",
		CoolNote:     "This is the only step you actually feel. It also dehumidifies — that's the water that drips from the indoor unit's drain line.",
		HeatNote:     "Surprisingly, even 0 °C air has heat in it. Outdoors, this coil pulls that heat out — which is why a heat pump can warm your house using cold air.",
	},
}

var refrigerants = []Refrigerant{
	{
		ID:            "r410a",
		Name:          "R-410A",
		AltName:       "Puron",
		Use:           "The standard in U.S. home AC & heat pumps (2000s–early 2020s).",
		SatCurve:      []SatPoint{{-7, 6.6}, {5, 9.3}, {45, 27.2}},
		DischargeTemp: 80,
		GWP:           2088,
		Safety:        "A1 — non-toxic, non-flammable",
		Color:         "#38bdf8",
		Note:          "Runs at high pressure. A blend being phased down because of its high global-warming potential.",
	},
	{
		ID:            "r32",
		Name:          "R-32",
		AltName:       "difluoromethane",
		Use:           "Newer split systems — the main replacement for R-410A.",
		SatCurve:      []SatPoint{{-7, 6.8}, {5, 9.5}, {45, 28.5}},
		DischargeTemp: 95,
		GWP:           675,
		Safety:        "A2L — mildly flammable",
		Color:         "#34d399",
		Note:          "About a third of R-410A's warming impact, but it runs noticeably hotter at the compressor.",
	},
	{
		ID:            "r134a",
		Name:          "R-134a",
		AltName:       "tetrafluoroethane",
		Use:           "Car AC, refrigerators, and chillers (older equipment).",
		SatCurve:      []SatPoint{{-7, 2.4}, {5, 3.5}, {45, 11.6}},
		DischargeTemp: 70,
		GWP:           1430,
		Safety:        "A1 — non-toxic, non-flammable",
		Color:         "#fbbf24",
		Note:          "Low, easy-to-handle pressures and a gentle discharge temp — but a high GWP, so it's being phased out.",
	},
	{
		ID:            "r717",
		Name:          "R-717",
		AltName:       "ammonia (NH₃)",
		Use:           "Industrial refrigeration — cold storage, food plants, ice rinks.",
		SatCurve:      []SatPoint{{-7, 3.2}, {5, 5.2}, {45, 17.8}},
		DischargeTemp: 120,
		GWP:           0,
		Safety:        "B2L — toxic, mildly flammable",
		Color:         "#a78bfa",
		Note:          "Extremely efficient with zero warming impact — but toxic and corrosive to copper, so it's never used in homes, and it runs very hot.",
	},
}

var faults = []Fault{
	{
		ID:       "dirty-filter",
		Title:    "Dirty air filter",
		Icon:     "🌫️",
		Phase:    "evaporation",
		Severity: "DIY",
		Cause:    "The filter clogs with dust and chokes the airflow across the indoor coil.",
		Effect:   "Too little warm air reaches the cold evaporator, so the refrigerant can't pick up enough heat to fully boil. The coil keeps getting colder until moisture on it freezes into ice — which blocks the airflow even more.",
		Symptoms: "Weak airflow from the vents, ice on the indoor coil or copper lines, long run times, and creeping energy bills.",
		Fix:      "Check the filter monthly and replace or wash it every 1–3 months. This is the single most common AC problem and the easiest to prevent.",
	},
	{
		ID:       "low-charge",
		Title:    "Low refrigerant (a leak)",
		Icon:     "💧",
		Phase:    "evaporation",
		Severity: "Call a pro",
		Cause:    "A leak somewhere in the sealed loop. Refrigerant is never 'used up' — if you're low, it escaped.",
		Effect:   "Less refrigerant means the low-side pressure drops, so the evaporator runs colder than designed and can freeze. Meanwhile the compressor runs hot with too little gas flowing through to cool it, and can eventually burn out.",
		Symptoms: "Warm air, hissing or bubbling sounds, ice on the suction line, and the classic 'it's just not as cold as it used to be.'",
		Fix:      "A pro has to find and seal the leak, then recharge to spec. Just 'topping it off' wastes refrigerant and the leak comes back.",
	},
	{
		ID:       "dirty-condenser",
		Title:    "Dirty condenser coil",
		Icon:     "🍂",
		Phase:    "condensation",
		Severity: "DIY / pro",
		Cause:    "The outdoor coil gets caked with dirt, grass clippings, cottonwood, and leaves.",
		Effect:   "The coil can't dump its heat to the outside air, so the high-side pressure and temperature climb. The compressor strains against that high pressure, wastes energy, overheats, and may trip its high-pressure safety switch.",
		Symptoms: "A very hot outdoor unit, high bills, breakers that trip on hot days, and weak cooling exactly when you need it most.",
		Fix:      "Gently rinse the outdoor coil with a hose (power off) and keep about 2 ft of clearance around the unit.",
	},
	{
		ID:       "dirty-evaporator",
		Title:    "Dirty evaporator coil",
		Icon:     "🧼",
		Phase:    "evaporation",
		Severity: "Call a pro",
		Cause:    "Dust slips past a cheap or missing filter over months and coats the indoor coil.",
		Effect:   "The grime acts like a blanket, insulating the coil so it can't absorb heat from your air. Cooling capacity drops and, like a dirty filter, the coil can ice over.",
		Symptoms: "Weak cooling even though everything is running, a musty smell, and ice on the coil.",
		Fix:      "Have a technician clean the coil — it's behind the unit's panels and easy to damage. Then use a good filter to keep it clean.",
	},
	{
		ID:       "frozen-coil",
		Title:    "Frozen evaporator coil",
		Icon:     "🧊",
		Phase:    "evaporation",
		Severity: "DIY first aid + pro",
		Cause:    "Almost always a symptom of something else: low airflow (dirty filter or weak blower) or low refrigerant.",
		Effect:   "Ice encases the coil and blocks the airflow completely, so cooling stops and the melt water can overflow the drain pan and leak.",
		Symptoms: "Visible ice on the coil or lines, water pooling around the indoor unit, and no cold air at all.",
		Fix:      "Switch cooling off and run just the fan to thaw it (a few hours), then fix the root cause — change the filter or call a pro for the leak.",
	},
	{
		ID:       "weak-compressor",
		Title:    "Failing compressor",
		Icon:     "💔",
		Phase:    "compression",
		Severity: "Call a pro",
		Cause:    "Age, an electrical fault, chronic overheating, or liquid refrigerant slugging back and damaging it.",
		Effect:   "The pump that drives the entire loop weakens or stops. With no pressure difference, no heat moves anywhere. This is the most expensive part to fail.",
		Symptoms: "Humming but not starting, hard-start clicking, tripping breakers, or simply no cooling at all.",
		Fix:      "A pro must replace it — often costly enough that a whole new unit makes more sense. Prevent it by fixing leaks and dirty coils early, since those kill compressors.",
	},
	{
		ID:       "stuck-valve",
		Title:    "Stuck metering device",
		Icon:     "🚪",
		Phase:    "expansion",
		Severity: "Call a pro",
		Cause:    "Debris, moisture freezing inside it, or a failed sensing bulb on the expansion valve.",
		Effect:   "Stuck closed/restricted, it starves the evaporator (which then freezes, just like low charge). Stuck open, it floods too much liquid through, which slugs back and can wreck the compressor.",
		Symptoms: "Poor cooling, frost showing up in odd spots, and unusual compressor noise.",
		Fix:      "Needs a technician to diagnose and replace the valve and clear whatever fouled it.",
	},
	{
		ID:       "overcharge",
		Title:    "Too much refrigerant",
		Icon:     "📈",
		Phase:    "compression",
		Severity: "Call a pro",
		Cause:    "Someone added more refrigerant than the system is rated for — more isn't better.",
		Effect:   "Excess liquid raises the high-side pressure and can flood back to the compressor. Counter-intuitively, capacity and efficiency actually drop.",
		Symptoms: "High energy bills, high head pressure on the gauges, and compressor wear over time.",
		Fix:      "A pro reclaims the excess to bring the charge back to the exact spec on the unit's label.",
	},
}

// cyclePayload is the full dataset the page needs, served as one JSON document.
type cyclePayload struct {
	Modes        []Mode        `json:"modes"`
	Refrigerants []Refrigerant `json:"refrigerants"`
	Phases       []Phase       `json:"phases"`
	Faults       []Fault       `json:"faults"`
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

	mux.HandleFunc("/api/cycle", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, cyclePayload{
			Modes:        modes,
			Refrigerants: refrigerants,
			Phases:       phases,
			Faults:       faults,
		})
	})

	// Kept for backwards compatibility — just the phases.
	mux.HandleFunc("/api/phases", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, phases)
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
			Faults []Fault
			Year   int
		}{
			Phases: phases,
			Faults: faults,
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
