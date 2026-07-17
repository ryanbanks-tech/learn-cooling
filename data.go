package main

import (
	"fmt"
	"strings"
)

// This file is the content layer: the cycle data and the page copy. Everything
// the site knows about refrigeration lives here, and the handlers in main.go
// just present it.

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

// Fan is one of the air-movers in the system. The cycle only works if air is
// pushed across the coils, and that's what the fans do.
type Fan struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Location string `json:"location"`
	Type     string `json:"type"`
	Job      string `json:"job"`
	Detail   string `json:"detail"`
	Phase    string `json:"phase"` // the cycle stage this fan serves
}

// Fault is something that commonly goes wrong with a home system. Each one is
// both a card on the home page and a standalone page at /problems/{slug},
// because "why is my AC frozen" is what people actually search for.
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

	// Page-specific copy for /problems/{slug}.
	Slug      string   `json:"slug"`
	Question  string   `json:"question"`  // the H1 — phrased the way people search
	MetaTitle string   `json:"metaTitle"` // <title>
	MetaDesc  string   `json:"metaDesc"`
	Answer    string   `json:"answer"` // short, direct answer up top
	Steps     []string `json:"steps"`
	Related   []string `json:"related"` // other fault IDs
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

var fans = []Fan{
	{
		ID:       "condenser-fan",
		Name:     "Condenser fan",
		Icon:     "🌀",
		Location: "Outdoor unit",
		Type:     "Axial (propeller) fan",
		Job:      "Pulls outdoor air across the hot condenser coil so the refrigerant can dump its heat.",
		Detail:   "It's the big fan you see spinning on top of the outdoor unit, blowing warm air straight up and out. The more clean air it moves, the better the refrigerant condenses back into a liquid. If it slows or stops — a worn motor or a dead capacitor — the high-side pressure spikes, the compressor overheats, and the system can trip on a safety switch.",
		Phase:    "condensation",
	},
	{
		ID:       "blower-fan",
		Name:     "Indoor blower",
		Icon:     "🌬️",
		Location: "Indoor unit / air handler",
		Type:     "Centrifugal 'squirrel-cage' blower",
		Job:      "Pushes your home's air across the cold evaporator coil and out through the ducts.",
		Detail:   "This is the fan you actually hear inside and feel at the vents. It draws warm room air in through the return, forces it over the cold evaporator where it's chilled and dehumidified, and sends it back to your rooms. Choke its airflow — a dirty filter, closed vents, or a tired motor — and the coil gets so cold it freezes over.",
		Phase:    "evaporation",
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

		Slug:      "dirty-air-filter",
		Question:  "Why does a dirty air filter stop your AC from cooling?",
		MetaTitle: "Dirty AC Air Filter: Symptoms, Frozen Coils, and the 5-Minute Fix",
		MetaDesc:  "A clogged filter starves your evaporator coil of airflow until it ices over. Here's what to look for, why it freezes, and how to fix it yourself.",
		Answer:    "A clogged filter chokes the airflow across your evaporator coil. With too little warm air passing over it, the refrigerant can't absorb enough heat to boil off, so the coil keeps getting colder — until the moisture condensing on it freezes solid and blocks the airflow completely. It's the most common AC fault there is, and the cheapest to prevent.",
		Steps: []string{
			"Turn the system off at the thermostat before pulling anything.",
			"Slide the filter out and hold it up to a light. If you can't see light through it, it's long overdue.",
			"Note the size printed on the cardboard frame (e.g. 20x25x1) and fit the replacement with the airflow arrow pointing toward the unit.",
			"If the coil is already iced over, run the fan alone for 2–4 hours to thaw it before you cool again.",
			"Set a monthly phone reminder to check it. Most filters want replacing every 1–3 months.",
		},
		Related: []string{"frozen-coil", "dirty-evaporator"},
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

		Slug:      "refrigerant-leak",
		Question:  "Why is my AC low on refrigerant?",
		MetaTitle: "AC Refrigerant Leak: Symptoms, Causes, and Why Topping Off Fails",
		MetaDesc:  "Refrigerant is never consumed — if you're low, it leaked. The symptoms, what low charge does to your compressor, and the only real fix.",
		Answer:    "Because it leaked. Refrigerant runs in a sealed loop and is never consumed, so a system that's low has a hole somewhere. Low charge drops the low-side pressure, which makes the evaporator run colder than designed (and ice up), while starving the compressor of the gas flow that normally carries its heat away.",
		Steps: []string{
			"Listen for hissing or bubbling near the refrigerant lines.",
			"Look for ice on the copper suction line or the indoor coil.",
			"Check fittings for an oily film — escaping refrigerant carries compressor oil out with it, so oil marks the leak.",
			"Call a licensed tech. They'll find the leak with dye or an electronic sniffer, repair it, then weigh in an exact charge.",
			"Refuse a 'top-off' with no leak search. It's money vented into the atmosphere, and it's illegal to knowingly vent refrigerant.",
		},
		Related: []string{"frozen-coil", "weak-compressor"},
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

		Slug:      "dirty-condenser-coil",
		Question:  "What happens when your outdoor AC coil gets dirty?",
		MetaTitle: "Dirty Condenser Coil: Why Your AC Struggles on the Hottest Days",
		MetaDesc:  "A dirt-caked outdoor coil can't dump heat, so pressure climbs and the compressor strains. Symptoms, the risk, and how to clean it safely.",
		Answer:    "The condenser coil's entire job is dumping your home's heat into the outdoor air. Caked with dirt, grass, and cottonwood, it can't — so high-side pressure and temperature climb. The compressor strains against that pressure, burns extra electricity, overheats, and can trip its high-pressure safety switch. It's why so many units die on the hottest day of the year.",
		Steps: []string{
			"Kill the power at the disconnect box beside the unit and at the breaker. Confirm it's dead before touching anything.",
			"Clear leaves, grass, and weeds away; keep about 2 ft of clearance on all sides.",
			"Rinse the coil from the inside out with a plain garden hose so debris pushes back out the way it came.",
			"Never use a pressure washer — it flattens the aluminum fins and makes the problem permanent.",
			"If the coil is greasy or matted deep between the fins, have a tech chemically clean it.",
		},
		Related: []string{"weak-compressor", "bad-capacitor"},
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

		Slug:      "dirty-evaporator-coil",
		Question:  "Why does a dirty indoor coil kill your cooling?",
		MetaTitle: "Dirty Evaporator Coil: Symptoms, Musty Smells, and the Fix",
		MetaDesc:  "Dust coating the indoor coil insulates it so it can't absorb heat. Why it happens, the musty 'dirty sock' smell, and why it's a pro job.",
		Answer:    "Dust that slips past a cheap filter settles on the permanently damp indoor coil and mats into a felt-like blanket. That layer insulates the coil, so the refrigerant can't absorb heat from your air — cooling drops off, the coil may ice, and the damp grime grows the mildew behind that musty 'dirty sock' smell.",
		Steps: []string{
			"Check the filter first — a dirty coil nearly always means the filter failed or was missing.",
			"Shine a flashlight at the coil behind the air-handler panel if you can reach it, but don't start scrubbing.",
			"Have a tech pull and chemically clean it. The fins are razor-thin and crush easily, and a damaged coil leaks refrigerant.",
			"Upgrade to a filter that actually seals in its track — air bypassing around the edges is what fouls coils.",
			"Keep the condensate drain clear so the coil isn't sitting in standing water.",
		},
		Related: []string{"dirty-filter", "frozen-coil"},
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

		Slug:      "frozen-evaporator-coil",
		Question:  "Why is my AC's evaporator coil frozen?",
		MetaTitle: "Why Is My AC Frozen? Ice on the Coil, Explained and Fixed",
		MetaDesc:  "Ice on your AC coil is a symptom, not the disease — it's almost always low airflow or low refrigerant. How to thaw it and find the real cause.",
		Answer:    "Ice on your coil is a symptom, not the disease. It's almost always one of two things: not enough air moving across the coil (dirty filter, weak blower, closed vents), or not enough refrigerant (a leak). Either way the coil runs colder than designed, so the moisture condensing on it freezes instead of draining away — and the ice then blocks the airflow completely, making it worse.",
		Steps: []string{
			"Set the thermostat's cooling to OFF but switch the fan to ON. Room-temperature air melts the ice fastest.",
			"Wait 2–4 hours; a thick block can take most of a day. Never chip or scrape the ice — you'll puncture the coil.",
			"Put towels down. A surprising amount of melt water comes off, and the drain pan can overflow.",
			"While it thaws, change the filter and open every supply vent in the house.",
			"Restart cooling. If it freezes again, you almost certainly have a refrigerant leak — that one needs a tech.",
		},
		Related: []string{"dirty-filter", "low-charge"},
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

		Slug:      "compressor-failure",
		Question:  "How do you know if your AC compressor is failing?",
		MetaTitle: "AC Compressor Failure: Symptoms, Causes, and Repair or Replace",
		MetaDesc:  "The compressor drives the whole refrigeration loop. The signs it's dying, what actually kills it, and the repair-or-replace math.",
		Answer:    "The compressor is the pump that drives the entire loop, so when it weakens or seizes there's no pressure difference and no heat moves anywhere — you get no cooling at all. Tell-tale signs are humming without starting, hard-start clicking, and tripped breakers. It's the most expensive part in the system, and it rarely dies of old age alone: leaks, dirty coils, and dead capacitors kill it first.",
		Steps: []string{
			"Note exactly what it does: hums but won't start, clicks, trips the breaker, or runs but never cools.",
			"Insist the tech checks the capacitor first — a cheap part fails far more often and mimics compressor death exactly.",
			"Ask them to ohm/megger the compressor windings before condemning it. 'It's the compressor' should be proven, not assumed.",
			"Get the repair-or-replace math in writing. A compressor swap often approaches the price of a new system, especially on an R-410A unit now being phased down.",
			"Whatever you decide, fix the root cause — the leak or the filthy coil — or the replacement dies the same way.",
		},
		Related: []string{"bad-capacitor", "low-charge"},
	},
	{
		ID:       "bad-capacitor",
		Title:    "Failed capacitor",
		Icon:     "🔋",
		Phase:    "compression",
		Severity: "Call a pro",
		Cause:    "Capacitors store and release the electrical jolt that starts and runs the compressor and the fan motors. Heat and age make them weaken, bulge, and fail — it's one of the single most common AC breakdowns, especially in a heat wave.",
		Effect:   "Without the capacitor's boost, a motor can't get going or keeps stalling. The outdoor fan may sit dead still while the unit just hums, or the compressor strains to start, pulls huge current, and trips the breaker.",
		Symptoms: "A humming outdoor unit with a fan that won't spin (sometimes it'll start if you nudge a blade — a classic sign), a clicking or buzzing relay, no cooling on the hottest days, or a visibly swollen, domed top on the capacitor.",
		Fix:      "It's a cheap part, but it holds a dangerous charge even with the power off. A pro discharges it safely and installs one with the exact microfarad (µF) and voltage rating.",

		Slug:      "bad-capacitor",
		Question:  "What are the symptoms of a bad AC capacitor?",
		MetaTitle: "Bad AC Capacitor Symptoms: Humming Unit, Fan Won't Spin",
		MetaDesc:  "A failed capacitor is one of the most common AC breakdowns. The classic humming-but-no-fan sign, why it happens, and why you shouldn't DIY it.",
		Answer:    "The classic sign is an outdoor unit that hums while its fan sits dead still — and then starts spinning if you nudge a blade with a stick. Capacitors store the electrical kick that gets the compressor and fan motors turning. When heat and age make them fail, the motors can't start: they stall, draw enormous current, and trip the breaker. A swollen, domed top on the capacitor is a dead giveaway.",
		Steps: []string{
			"Look at the top of the capacitor (a silver can in the outdoor unit's electrical panel). Domed or bulging means it's finished.",
			"Note the classic test: the unit hums, the fan won't spin, but a nudge with a stick starts it turning.",
			"Do not touch it. A capacitor holds a lethal charge for a long time even with the power off and the disconnect pulled.",
			"Have a tech discharge it safely and fit one matching the exact µF and voltage rating printed on the old can.",
			"It's a cheap part and a fast job — usually the best-value repair on the whole unit.",
		},
		Related: []string{"weak-compressor", "dirty-condenser"},
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

		Slug:      "expansion-valve-problems",
		Question:  "What happens when the expansion valve sticks?",
		MetaTitle: "AC Expansion Valve (TXV) Problems: Symptoms and Fixes",
		MetaDesc:  "A stuck metering device either starves the evaporator or floods the compressor. How to tell the two apart — and from a refrigerant leak.",
		Answer:    "The metering device controls exactly how much refrigerant sprays into the evaporator. Stuck restricted, it starves the coil and freezes it — mimicking a low charge almost perfectly. Stuck open, it floods liquid straight through to the compressor, which is built to pump gas, not liquid, and can be destroyed by the slugging.",
		Steps: []string{
			"Know that the symptoms mimic low refrigerant. Don't let anyone simply add refrigerant to 'see if it helps'.",
			"Ask the tech to compare superheat and subcooling readings — that pairing is what separates a TXV fault from an actual leak.",
			"Expect the filter-drier to be replaced too; debris or moisture is usually what fouled the valve.",
			"Valve replacement means recovering and recharging the refrigerant, so it's a half-day job, not a quick swap.",
			"If liquid has been slugging back, have the compressor evaluated at the same time.",
		},
		Related: []string{"frozen-coil", "low-charge"},
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

		Slug:      "overcharged-refrigerant",
		Question:  "Can you put too much refrigerant in an AC?",
		MetaTitle: "Overcharged AC: Why More Refrigerant Makes Cooling Worse",
		MetaDesc:  "More refrigerant isn't better. Overcharging raises head pressure, floods the compressor, and cuts capacity. The signs and the fix.",
		Answer:    "Yes — and more is emphatically not better. Excess refrigerant raises the high-side pressure and leaves liquid where the system expects gas, so it can flood back to the compressor and damage it. Counter-intuitively, both cooling capacity and efficiency drop while your energy bill climbs.",
		Steps: []string{
			"Suspect it if cooling got noticeably worse right after someone 'topped off' the system.",
			"Have a tech read head pressure and subcooling against the manufacturer's chart — that's the proof.",
			"The charge must match the exact weight on the unit's data plate, not a gauge-pressure guess.",
			"Ask them to recover the excess into a cylinder. Venting it to the air is illegal.",
			"If the top-off was chasing a leak, the leak still has to be found and repaired.",
		},
		Related: []string{"weak-compressor", "low-charge"},
	},
}

// SevClass maps the severity wording to a CSS class. Mirrors sevClass() in app.js.
func (f Fault) SevClass() string {
	switch {
	case strings.HasPrefix(f.Severity, "DIY"):
		return "diy"
	case strings.HasPrefix(f.Severity, "Call"):
		return "pro"
	default:
		return "mixed"
	}
}

// BarAt returns the saturation pressure at a temperature on the curve, or 0.
func (r Refrigerant) BarAt(tempC int) float64 {
	for _, p := range r.SatCurve {
		if p.TempC == tempC {
			return p.Bar
		}
	}
	return 0
}

// PressureAt formats the saturation pressure the way a tech reads it: absolute
// bar plus the gauge psi shown on a manifold.
func (r Refrigerant) PressureAt(tempC int) string {
	bar := r.BarAt(tempC)
	if bar == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f bar (%.0f psig)", bar, (bar-1.01325)*14.5038)
}

// GWPLabel renders the global-warming potential for display.
func (r Refrigerant) GWPLabel() string {
	if r.GWP == 0 {
		return "0 — none"
	}
	return fmt.Sprintf("%d× CO₂", r.GWP)
}

// faultBySlug / faultByID give the handlers cheap lookups.
var faultBySlug = map[string]*Fault{}
var faultByID = map[string]*Fault{}

func init() {
	for i := range faults {
		faultBySlug[faults[i].Slug] = &faults[i]
		faultByID[faults[i].ID] = &faults[i]
	}
}

// relatedFaults resolves a fault's Related IDs into real faults.
func relatedFaults(f *Fault) []*Fault {
	out := make([]*Fault, 0, len(f.Related))
	for _, id := range f.Related {
		if r, ok := faultByID[id]; ok {
			out = append(out, r)
		}
	}
	return out
}

// phaseByID finds the cycle stage a fault or fan refers to.
func phaseByID(id string) *Phase {
	for i := range phases {
		if phases[i].ID == id {
			return &phases[i]
		}
	}
	return nil
}
