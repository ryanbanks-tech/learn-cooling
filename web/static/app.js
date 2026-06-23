(function () {
    "use strict";

    var svg = document.getElementById("cycle");
    var panel = document.getElementById("panel");
    var modeToggle = document.getElementById("mode-toggle");
    var refToggle = document.getElementById("ref-toggle");
    var modeBanner = document.getElementById("mode-banner");
    var refCard = document.getElementById("ref-card");
    var faultsEl = document.getElementById("faults");

    var data = null;            // { modes, refrigerants, phases, faults }
    var modeById = {};
    var refById = {};
    var phaseById = {};
    var SUPERHEAT = 5;          // °C of superheat we assume leaving the evaporator

    var state = { mode: "cooling", ref: "r410a", phase: null };

    fetch("/api/cycle")
        .then(function (r) { return r.json(); })
        .then(function (d) {
            data = d;
            d.modes.forEach(function (m) { modeById[m.id] = m; });
            d.refrigerants.forEach(function (r) { refById[r.id] = r; });
            d.phases.forEach(function (p) { phaseById[p.id] = p; });
            init();
        })
        .catch(function (err) {
            console.error("Could not load cycle data:", err);
            panel.innerHTML = '<div class="panel-empty"><p>Couldn\'t load the data. Try refreshing.</p></div>';
        });

    function init() {
        buildToggles();
        buildFaults();
        wireNodes();

        document.addEventListener("keydown", function (e) {
            if (!state.phase) return;
            if (e.key === "ArrowRight") stepPhase(1);
            else if (e.key === "ArrowLeft") stepPhase(-1);
        });

        render();
    }

    // ---- controls ----------------------------------------------------------

    function buildToggles() {
        data.modes.forEach(function (m) {
            var b = document.createElement("button");
            b.className = "seg" + (m.id === state.mode ? " active" : "");
            b.type = "button";
            b.dataset.mode = m.id;
            b.textContent = m.icon + " " + m.name;
            b.addEventListener("click", function () { state.mode = m.id; render(); });
            modeToggle.appendChild(b);
        });

        data.refrigerants.forEach(function (r) {
            var b = document.createElement("button");
            b.className = "seg" + (r.id === state.ref ? " active" : "");
            b.type = "button";
            b.dataset.ref = r.id;
            b.textContent = r.name;
            b.addEventListener("click", function () { state.ref = r.id; render(); });
            refToggle.appendChild(b);
        });
    }

    function syncToggles() {
        modeToggle.querySelectorAll(".seg").forEach(function (b) {
            b.classList.toggle("active", b.dataset.mode === state.mode);
        });
        refToggle.querySelectorAll(".seg").forEach(function (b) {
            b.classList.toggle("active", b.dataset.ref === state.ref);
        });
    }

    // ---- diagram -----------------------------------------------------------

    function wireNodes() {
        svg.querySelectorAll(".node").forEach(function (node) {
            var id = node.getAttribute("data-phase");
            node.addEventListener("click", function () { state.phase = id; render(); });
            node.addEventListener("keydown", function (e) {
                if (e.key === "Enter" || e.key === " ") { e.preventDefault(); state.phase = id; render(); }
            });
        });
    }

    function stepPhase(dir) {
        var i = data.phases.findIndex(function (p) { return p.id === state.phase; });
        state.phase = data.phases[(i + dir + data.phases.length) % data.phases.length].id;
        render();
    }

    // ---- physics helpers ---------------------------------------------------

    function barAt(ref, tempC) {
        var pt = ref.satCurve.find(function (p) { return p.tempC === tempC; });
        return pt ? pt.bar : null;
    }

    // Show absolute bar plus the gauge psi a technician reads on the manifold.
    function fmtPressure(bar) {
        if (bar == null) return "—";
        var psig = Math.round((bar - 1.01325) * 14.5038);
        return bar.toFixed(1) + " bar (" + psig + " psig)";
    }

    function lowBar(ref, mode) { return barAt(ref, mode.evapTempC); }
    function highBar(ref, mode) { return barAt(ref, mode.condTempC); }

    // Temperatures and pressures a given phase runs between, for this ref + mode.
    function phaseNumbers(phase, ref, mode) {
        var evap = mode.evapTempC, cond = mode.condTempC;
        var disc = ref.dischargeTemp + (mode.id === "heating" ? 15 : 0); // hotter in heating
        var lo = fmtPressure(lowBar(ref, mode));
        var hi = fmtPressure(highBar(ref, mode));

        switch (phase.role) {
            case "compressor":
                return { temp: (evap + SUPERHEAT) + " °C  →  " + disc + " °C", pressure: lo + "  →  " + hi };
            case "condenser":
                return { temp: disc + " °C  →  " + cond + " °C", pressure: hi };
            case "expansion":
                return { temp: cond + " °C  →  " + evap + " °C", pressure: hi + "  →  " + lo };
            case "evaporator":
                return { temp: evap + " °C  →  " + (evap + SUPERHEAT) + " °C", pressure: lo };
        }
        return { temp: "—", pressure: "—" };
    }

    // ---- render ------------------------------------------------------------

    function render() {
        var mode = modeById[state.mode];
        var ref = refById[state.ref];

        syncToggles();

        // mode banner + diagram theming
        modeBanner.textContent = mode.icon + "  " + mode.banner;
        document.body.classList.toggle("heating", state.mode === "heating");
        document.getElementById("loc-condenser").textContent =
            state.mode === "heating" ? "Indoors" : "Outdoors";
        document.getElementById("loc-evaporator").textContent =
            state.mode === "heating" ? "Outdoors" : "Indoors";

        renderRefCard(ref, mode);
        if (state.phase) renderPanel(phaseById[state.phase], ref, mode);
    }

    function renderRefCard(ref, mode) {
        var lo = fmtPressure(lowBar(ref, mode));
        var hi = fmtPressure(highBar(ref, mode));
        var disc = ref.dischargeTemp + (mode.id === "heating" ? 15 : 0);
        var gwp = ref.gwp === 0
            ? '<span class="gwp-zero">0 — none</span>'
            : ref.gwp.toLocaleString();

        refCard.innerHTML =
            '<div class="ref-head" style="border-color:' + ref.color + '">' +
                "<div>" +
                    "<h3>" + esc(ref.name) + ' <span class="ref-alt">' + esc(ref.altName) + "</span></h3>" +
                    '<p class="ref-use">' + esc(ref.use) + "</p>" +
                "</div>" +
            "</div>" +
            '<p class="ref-note">' + esc(ref.note) + "</p>" +
            '<div class="ref-stats">' +
                stat("Low side (" + mode.evapTempC + " °C)", lo) +
                stat("High side (" + mode.condTempC + " °C)", hi) +
                stat("Compressor discharge", "~" + disc + " °C") +
                stat("Global warming potential", gwp) +
                stat("Safety class", esc(ref.safety)) +
            "</div>";
    }

    function stat(k, v) {
        return '<div class="rstat"><span class="k">' + k + '</span><span class="v">' + v + "</span></div>";
    }

    function renderPanel(p, ref, mode) {
        var i = data.phases.findIndex(function (x) { return x.id === p.id; });
        var prev = data.phases[(i - 1 + data.phases.length) % data.phases.length];
        var next = data.phases[(i + 1) % data.phases.length];
        var nums = phaseNumbers(p, ref, mode);
        var loc = state.mode === "heating" ? p.heatLocation : p.coolLocation;
        var note = state.mode === "heating" ? p.heatNote : p.coolNote;
        var noteLabel = state.mode === "heating" ? "In heating mode" : "In your home";

        svg.querySelectorAll(".node").forEach(function (n) {
            n.classList.toggle("active", n.getAttribute("data-phase") === p.id);
        });

        panel.innerHTML =
            '<div class="panel-card">' +
                '<span class="badge" style="background:' + p.accent + '">Stage ' + p.number + " of 4</span>" +
                "<h2>" + esc(p.name) + "</h2>" +
                '<p class="component">' + esc(p.component) + " · " + esc(loc) + "</p>" +
                '<div class="facts">' +
                    fact("Pressure", nums.pressure) +
                    fact("Temperature", nums.temp) +
                "</div>" +
                '<p class="detail">' + esc(p.detail) + "</p>" +
                '<p class="home-note"><strong>' + noteLabel + ":</strong> " + esc(note) + "</p>" +
                '<div class="panel-nav">' +
                    '<button data-go="' + prev.id + '">← ' + esc(prev.name) + "</button>" +
                    '<button data-go="' + next.id + '">' + esc(next.name) + " →</button>" +
                "</div>" +
            "</div>";

        panel.querySelectorAll("[data-go]").forEach(function (b) {
            b.addEventListener("click", function () { state.phase = b.getAttribute("data-go"); render(); });
        });
    }

    function fact(k, v) {
        return '<div class="fact"><span class="k">' + esc(k) + '</span><span class="v">' + esc(v) + "</span></div>";
    }

    // ---- faults ------------------------------------------------------------

    function buildFaults() {
        data.faults.forEach(function (f) {
            var card = document.createElement("button");
            card.type = "button";
            card.className = "fault";
            card.setAttribute("aria-expanded", "false");
            card.innerHTML =
                '<div class="fault-head">' +
                    '<span class="fault-icon">' + f.icon + "</span>" +
                    '<span class="fault-title">' + esc(f.title) + "</span>" +
                    '<span class="fault-sev sev-' + sevClass(f.severity) + '">' + esc(f.severity) + "</span>" +
                "</div>" +
                '<div class="fault-body">' +
                    detail("Cause", f.cause) +
                    detail("What it does", f.effect) +
                    detail("You'd notice", f.symptoms) +
                    detail("The fix", f.fix) +
                "</div>";
            card.addEventListener("click", function () {
                var open = card.classList.toggle("open");
                card.setAttribute("aria-expanded", open ? "true" : "false");
            });
            faultsEl.appendChild(card);
        });
    }

    function detail(k, v) {
        return '<p class="fdetail"><strong>' + k + ":</strong> " + esc(v) + "</p>";
    }

    function sevClass(s) {
        if (s.indexOf("DIY") === 0) return "diy";
        if (s.indexOf("Call") === 0) return "pro";
        return "mixed";
    }

    // ---- util --------------------------------------------------------------

    function esc(s) {
        return String(s)
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;");
    }
})();
