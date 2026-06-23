(function () {
    "use strict";

    var panel = document.getElementById("panel");
    var svg = document.getElementById("cycle");
    var phases = [];
    var byId = {};
    var currentIndex = -1;

    // Pull the phase data from the Go backend. Single source of truth lives in main.go.
    fetch("/api/phases")
        .then(function (r) { return r.json(); })
        .then(function (data) {
            phases = data;
            data.forEach(function (p) { byId[p.id] = p; });
            wireUp();
        })
        .catch(function (err) {
            console.error("Could not load phases:", err);
            panel.innerHTML = '<div class="panel-empty"><p>Couldn\'t load the cycle data. Try refreshing.</p></div>';
        });

    function wireUp() {
        var nodes = svg.querySelectorAll(".node");
        nodes.forEach(function (node) {
            var id = node.getAttribute("data-phase");
            node.addEventListener("click", function () { select(id); });
            node.addEventListener("keydown", function (e) {
                if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    select(id);
                }
            });
        });

        // Arrow keys move between stages once one is open.
        document.addEventListener("keydown", function (e) {
            if (currentIndex < 0) return;
            if (e.key === "ArrowRight") { step(1); }
            else if (e.key === "ArrowLeft") { step(-1); }
        });
    }

    function step(dir) {
        var next = (currentIndex + dir + phases.length) % phases.length;
        select(phases[next].id);
    }

    function select(id) {
        var p = byId[id];
        if (!p) return;
        currentIndex = phases.findIndex(function (x) { return x.id === id; });

        svg.querySelectorAll(".node").forEach(function (n) {
            n.classList.toggle("active", n.getAttribute("data-phase") === id);
        });

        var prev = phases[(currentIndex - 1 + phases.length) % phases.length];
        var nxt = phases[(currentIndex + 1) % phases.length];

        panel.innerHTML =
            '<div class="panel-card">' +
                '<span class="badge" style="background:' + p.accent + '">Stage ' + p.number + ' of 4</span>' +
                "<h2>" + esc(p.name) + "</h2>" +
                '<p class="component">' + esc(p.component) + " · " + esc(p.location) + "</p>" +
                '<div class="flow">' +
                    "<span>" + esc(p.stateIn) + "</span>" +
                    '<span class="arrow-i">→</span>' +
                    "<span>" + esc(p.stateOut) + "</span>" +
                "</div>" +
                '<div class="facts">' +
                    fact("Pressure", p.pressure) +
                    fact("Temperature", p.temp) +
                "</div>" +
                '<p class="detail">' + esc(p.detail) + "</p>" +
                '<p class="home-note"><strong>In your home:</strong> ' + esc(p.homeNote) + "</p>" +
                '<div class="panel-nav">' +
                    '<button data-go="' + prev.id + '">← ' + esc(prev.name) + "</button>" +
                    '<button data-go="' + nxt.id + '">' + esc(nxt.name) + " →</button>" +
                "</div>" +
            "</div>";

        panel.querySelectorAll("[data-go]").forEach(function (b) {
            b.addEventListener("click", function () { select(b.getAttribute("data-go")); });
        });
    }

    function fact(k, v) {
        return '<div class="fact"><span class="k">' + esc(k) + '</span><span class="v">' + esc(v) + "</span></div>";
    }

    function esc(s) {
        return String(s)
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;");
    }
})();
