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
| **Drag** | turn the camera (in *Look* mode a click captures the pointer for FPS-style look) |
| **Left-click** | on a highlighted node — remove it (and its edges) from the graph; a drag turns instead |
| **W A S D** | move · **Q/E** down/up · **Shift** boost |
| **scroll** | fly speed |
| **Esc** | release the pointer |
| **Space** | re-heat — random scatter, then re-settle |
| **X** | spread — push every node straight out from the centre (keeping its direction), then re-settle |
| **1 2 3** | highlight depth — light up 1, 2 or 3 hops of connections from the aimed node |
| **H** | hide *all* panels |

Two switches, top-right:

- **Autopilot on/off** (default **off**) — when on, the camera flies itself after
  10 s idle: an establishing overview from outside, then in through the dense
  core, dodging nodes as it goes. Off keeps it wherever you leave it.
- **Mouse: Look / Inspect** (default **Inspect**) — *Inspect* frees the cursor to
  *be* the crosshair, so you hover nodes to inspect them without moving (drag to
  turn, WASD to fly); *Look* captures the pointer and turns the camera about a
  fixed centre crosshair ⊕.

**Re-heat vs. spread.** `Space` scatters the nodes randomly onto a shell and lets
the forces re-sort from scratch. `X` instead keeps each node's *direction* from
the centre and only pushes it straight outward, so the angular structure is
preserved while the graph inflates and re-settles — usually into cleaner
groupings. The push distance is captured on the **first** `X` (the furthest
node's distance × `CFG.spreadMult`) and reused on every later `X`, so each press
adds the same fixed step outward rather than inflating an already-spread graph by
an ever-growing amount.

Look is free — the camera rotates about its own axes (no fixed up, no pole
limit), so it feels the same whichever way you face, fitting a graph with no
real up or down.

The aimed node lights up white; thick **green** edges (with arrowheads) are the
ones it calls, thick **orange** the ones that call it — the arrows point the
call direction. Each highlighted edge is a solid cylinder + cone (real geometry,
so *every* edge shows a line — no dropped segments). Its neighbours brighten and
nearby nodes show their name.

Press **1 / 2 / 3** (or set `CFG.hlDepth`) to fan the highlight out that many
**hops** — 2nd- and 3rd-order callers/callees, both directions at once — each hop
drawn dimmer and thinner (`CFG.hlFalloff`). Tune the edge glow with `CFG.hlBright`
and the arrowhead size with `CFG.coneSize`.

Set `CFG.focusLabels = true` for a focus-reading label mode: names show normally
until you highlight a node, then **only that node and its highlighted connections
keep their names** — at any distance — so you can read exactly who it talks to
(it follows `CFG.hlDepth`, so `1`/`2`/`3` widens the labelled set).

**Left-click a highlighted node to delete it** (its edges go too). Every survivor
keeps its position and the camera holds still — the layout just eases the gap
shut. A click that drags past a few pixels turns the camera instead, so you never
delete by accident while flying.

## Notes

- Colours: cool = defined functions (by source file), warm amber = external
  callees. Nodes rest dim so the crosshair highlight pops; size scales with
  degree, so hubs read as the giants.
- Layout: nodes start spread over a large sphere and are pulled together while
  repulsion ramps up as the sim cools, so connected groups clump before spacing
  is enforced. Connected nodes attract along edges (`CFG.spring`), and that pull
  grows *super-linearly* the more an edge is stretched (`CFG.springStretch`), so
  far-apart connected nodes snap back hard; everything is drawn toward the centre
  (`CFG.gravity`) and repels its neighbours — nudge spring / springStretch up and
  gravity down for tighter, better-separated groups. **External** callees are
  tethered more loosely (`CFG.extSpring` weakens their caller's spring), so
  repulsion carries the calls-to-the-outside out to the rim — a balanced spring,
  so it never drifts the whole graph the way a one-sided outward push would. Every
  knob in the `CFG` block at the top of `graph3d.js` is documented inline: what it
  does, the effect of raising or lowering it, and the formula that reads it.
- Targeting is **GPU picking**: the nodes are re-rendered, each in a colour that
  encodes its index, into a 1×1 buffer at the cursor (`camera.setViewOffset`);
  reading that one pixel back gives exactly the node whose real geometry is drawn
  there — pixel-precise, correct depth for free, no projection maths to drift, no
  glow to over-shoot. Coordinates are canvas-relative (`getBoundingClientRect`),
  so a mis-positioned canvas can't offset it.
- Self-contained: `graph3d.html` (shell + CDN three.js r136 + bloom) and
  `graph3d.js` (everything else). No build step.
- `window.__graph3d` exposes `load()`, `loadUrl(url)`, `loadText(name, text)`,
  `cfg` (the live `CFG` tunables — mutate e.g. `__graph3d.cfg.hlBright` to tweak on
  the fly) plus debug hooks: `state()`, `dbg()`, `extent()` (live max/mean node distance
  from the centroid + the current X-spread step — handy for tuning the `CFG`
  force knobs), `focus(i, depth)` (force the highlight onto node `i` and report
  its multi-hop fan-out), `simulate(steps, dt)`, `overview()`, and `lookAtHub()`
  (park the camera on the highest-degree node).
