# graph3d — call graphs in 3D

A force-directed 3D viewer for the graphs the metacompiler emits (`-callgraph`,
`-cfgraph`, `-render`). Nodes glow, edges web between them, and the camera flies
itself through the graph on a smooth spline — or you take the stick and fly by
hand.

![a call graph rendered as a 3D galaxy]

## Run it

Two ways:

**Quick (drag-and-drop).** Open the file directly and drop a graph onto it:

```sh
open viz/graph3d.html          # macOS  (or just double-click it)
```

Then drag a `.dot` anywhere onto the window. Needs an internet connection the
first time — three.js is pulled from a CDN.

**With a URL parameter.** Serve the folder and pass the graph via `?src=`:

```sh
cd viz && python3 -m http.server 8777
# then open http://localhost:8777/graph3d.html?src=../my-callgraph.dot
```

(`?src=` uses `fetch`, which the browser blocks over `file://`, so it only works
when served — hence the tiny server. Drag-and-drop works either way.)

## Make a graph to view

```sh
# whole-codebase call graph (what the screenshot shows)
./mec languages/kotlin-to-llvm-ir.abnf App.kt -i src/ -callgraph app.dot

# control-flow graph of one function
./mec languages/java-interpreter.abnf Foo.java -cfgraph foo.dot
```

Any `digraph` with `"a" -> "b"` edges loads. A plain `{ "nodes":[...],
"links":[...] }` JSON works too. Recognized DOT extras: `subgraph cluster_N`
(one color per file), `[style=dashed]` (external callees, drawn amber),
`[label="N"]` on edges (call-site counts).

Big graphs are the point — the Android/Compose sample is ~1250 nodes / ~1800
edges and flies smoothly. (For flat, printable output, keep using Graphviz
`dot`/`sfdp` on the same `.dot`; this viewer is the fly-through companion.)

## Controls

| | |
|---|---|
| **Click** | look around — in *Look* mode this captures the pointer (proper FPS capture) |
| **W A S D** | move · **Q/E** down/up · **Shift** boost |
| **scroll** | fly speed |
| **Esc** | release the pointer |
| **Space** | re-heat the layout (re-settle) |
| **H** | hide the help |

Two switches, top-right:

- **Autopilot on/off** — when on, the camera flies itself after 10 s idle: an
  establishing overview from outside, then in through the dense core, dodging
  nodes as it goes. Off keeps it wherever you leave it.
- **Mouse: Look / Inspect** — *Look* captures the pointer and turns the camera
  (fixed centre crosshair ⊕); *Inspect* frees the cursor to *be* the crosshair,
  so you can hover nodes to inspect them without moving (drag to turn, WASD to
  fly).

Look is free — the camera rotates about its own axes (no fixed up, no pole
limit), so it feels the same whichever way you face, fitting a graph with no
real up or down.

The aimed node lights up white; thick **green** edges (with arrowheads) are the
ones it calls, thick **orange** the ones that call it — the arrows point the
call direction. Its neighbours brighten and nearby nodes show their name.

## Notes

- Colours: cool = defined functions (by source file), warm amber = external
  callees. Nodes rest dim so the crosshair highlight pops; size scales with
  degree, so hubs read as the giants.
- Layout: nodes start spread over a large sphere and are pulled together while
  repulsion ramps up as the sim cools, so connected groups clump before spacing
  is enforced (tunable via `CFG.initSpread` / `repMin` / `warmup`).
- Self-contained: `graph3d.html` (shell + CDN three.js r136 + bloom + fat lines)
  and `graph3d.js` (everything else). No build step.
- `window.__graph3d` exposes `load()`, `loadUrl(url)`, `loadText(name, text)`
  plus debug hooks: `state()`, `simulate(steps, dt)`, `overview()`, and
  `lookAtHub()` (park the camera on the highest-degree node).
