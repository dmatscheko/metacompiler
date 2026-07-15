/* graph3d.js - a 3D force-directed viewer for the metacompiler's .dot graphs.
 *
 * Loads the DOT emitted by -callgraph / -cfgraph / -render (or a plain
 * {nodes,links} JSON), lays it out with a grid-accelerated force simulation,
 * and renders glowing instanced nodes + additive edges with bloom.
 *
 * Two camera modes:
 *   AUTO - flies a Catmull-Rom (bezier-equivalent) spline that wanders through
 *          the graph. A soft repulsion from nearby nodes keeps it from clipping
 *          straight through them.
 *   FLY  - click to grab the pointer, then WASD + mouse-look + Q/E to fly by
 *          hand. Any steering input switches to FLY; after 10s idle it eases
 *          back into AUTO from wherever you left off.
 *
 * Needs THREE (r136 global build) loaded first; bloom is optional and degrades
 * gracefully to a plain render if the postprocessing addons fail to load.
 */
(function () {
  'use strict';

  // ---- tunables -----------------------------------------------------------
  // Every knob below lists: WHAT it controls · the EFFECT of raising / lowering
  // it · the FORMULA that reads it. The layout is a simulated-annealed force sim:
  // each tick sums repulsion + edge springs + gravity into an acceleration,
  // integrates it with damping into velocity/position, then cools `alpha` (which
  // scales all forces) a little toward alphaMin. See layoutTick() for the maths.
  var CFG = {
    // ---- force layout ----
    restLen: 9,          // preferred edge length, in world units - the separation the
                         //   spring pulls connected nodes toward. ↑ = looser, airier
                         //   graph; ↓ = tighter. Also sizes the repulsion grid cell
                         //   (restLen*1.8) and the initial scatter shell.
    repulsion: 240,      // node-node repulsion strength once fully cooled. Every nearby
                         //   pair pushes apart with force repulsion/dist². ↑ = more
                         //   personal space, the graph inflates; ↓ = nodes pack tight.
    repMin: 0.12,        // repulsion scale while still HOT, ramping to 1.0 as it cools:
                         //   repScale = repMin + (1-repMin)*cooled  (cooled: 0 hot → 1 cold).
                         //   Low = springs & gravity win early so groups CLUMP before
                         //   spacing kicks in. ↑ toward 1 = even spacing from the start
                         //   (less clumping); ↓ = a stronger initial huddle.
    spring: 0.05,        // edge spring stiffness: the attraction between CONNECTED nodes.
                         //   f = spring*(dist-restLen), applied along the edge. ↑ = tighter,
                         //   more compact clusters (can cramp); ↓ = edges slacken and
                         //   connected groups drift apart.
    springStretch: 0.6,  // super-linear EXTRA pull when an edge is stretched past restLen:
                         //   the spring force is multiplied by (1 + springStretch*stretch/
                         //   restLen), where stretch = dist-restLen (only while stretched).
                         //   So the further apart two connected nodes are, the HARDER they
                         //   snap together. ↑ = long bridges pull taut and clusters ball up
                         //   tightly; 0 = a plain linear Hooke spring (pull grows only
                         //   linearly with distance).
    gravity: 0.001,      // every node's pull toward the origin: accel += -pos*gravity. This
                         //   is the ONLY force on UNconnected nodes, so it sets how far loose
                         //   pieces drift. ↑ = the whole graph hugs the centre (groups
                         //   overlap); ↓ = groups fly apart and separate (loners wander off).
    damping: 0.86,       // fraction of velocity kept per tick: vel = (vel+accel)*damping -
                         //   i.e. friction. ↓ (≈0.7) = sluggish, settles fast but stiff; ↑
                         //   toward 1 = springy, keeps jiggling and can oscillate.
    maxStep: 9,          // hard clamp on how far a node may move in ONE tick (world units) -
                         //   the anti-explosion guard when forces spike (e.g. a fresh
                         //   scatter). ↓ = calmer but slower to settle; ↑ = snaps into place
                         //   faster, risking overshoot.
    alphaDecay: 0.988,   // annealing cooldown per tick: alpha *= alphaDecay. alpha scales ALL
                         //   forces and drives the repulsion ramp. ↓ (≈0.97) = cools fast,
                         //   freezes sooner (less settling); ↑ toward 1 = simmers longer for
                         //   a cleaner layout, but takes longer to get there.
    alphaMin: 0.015,     // floor alpha never cools below, so the sim keeps breathing gently
                         //   forever instead of freezing solid. ↑ = more idle drift/life;
                         //   ↓ toward 0 = the graph comes fully to rest.
    warmup: 320,         // layout ticks run synchronously BEFORE the first frame, so the graph
                         //   opens already settled (scatter → contract → settle). ↑ = a better
                         //   first look but a longer load pause; ↓ = loads faster but opens
                         //   mid-collapse.
    initSpread: 5,       // initial scatter size, as a multiple of the natural radius: nodes
                         //   start on a shell of restLen*cbrt(N)*initSpread+40, then contract.
                         //   ↑ = starts more exploded (helps big graphs untangle); ↓ = starts
                         //   compact (can trap knots).
    spreadMult: 2.0,     // X-key spread amount: the FIRST X captures a push distance of
                         //   (furthest-node distance * spreadMult); every later X moves each
                         //   node that SAME stored distance further out along its direction.
                         //   ↑ = each X inflates the graph more; ↓ = gentler nudges.

    // ---- appearance ----
    bg: 0x05060c,        // background + fog colour (near-black).
    nodeBase: 1.0,       // base node radius. The drawn radius grows with degree:
                         //   nodeBase*(0.7 + 1.7*sqrt(deg/maxDeg)). ↑ = bigger balls overall.
    fog: 0.0022,         // exponential fog density - a depth cue fading distant nodes into the
                         //   background. ↑ = murkier, shorter sight line; ↓ = clearer, deeper.

    // ---- camera: fly (manual) ----
    flySpeed: 80,        // WASD / Q / E move speed, world units/sec (before scroll & boost).
    flyBoost: 5,         // Shift multiplier on flySpeed for a fast dash.
    mouseSens: 0.0022,   // radians of camera turn per pixel of mouse motion. ↑ = twitchier look.

    // ---- camera: auto (autopilot spline) ----
    autoSpeed: 24,       // travel speed along the flight spline, world units/sec.
    autoLook: 0.5,       // how hard the camera turns toward the flight tangent (slerp
                         //   fraction). ↑ = snappier aim; ↓ = lazier, driftier aim.
    lookSmooth: 0.08,    // low-pass on the look TARGET itself (A.look eased toward the tangent).
                         //   ↓ = smoother, wider turns; ↑ = hugs the curve more tightly.
    maxTurn: 1.0,        // hard cap on turn rate, rad/sec, so sharp curvature or a node-dodge
                         //   can't yank the view around. ↓ = calmer camera.
    avoidRadius: 10,     // autopilot only dodges nodes within this distance (ones it would
                         //   actually clip). ↑ = gives every node a wider berth.
    avoidStrength: 3.0,  // per-node dodge magnitude; soft, falling to 0 at avoidRadius.
    maxAvoid: 4,         // hard cap on the total dodge offset (world units): a nudge through
                         //   the gaps, never a shove out of the graph.
    avoidSmooth: 0.07,   // how fast the dodge offset eases in/out (lower = softer).
    idleMs: 10000,       // ms of no input before autopilot (when ON) resumes flying itself.

    // ---- appearance: focus highlight & labels ----
    nodeDim: 0.55,       // resting node brightness (colour * nodeDim), dimmed so the white
                         //   focus highlight pops. ↑ = brighter resting nodes (less contrast).
    lineRadius: 0.32,    // world radius of a highlighted edge's cylinder (the fat line). ↑ =
                         //   chunkier highlight lines.
    labelMax: 130,       // most node-name labels drawn at once (nearest-to-camera win). ↑ = more
                         //   names on screen but more clutter and DOM cost.
    labelDist: 4.5       // show a node's name when it's within labelDist*coreRadius of the
                         //   camera. ↑ = names appear from further away.
  };

  // ---- DOM ----------------------------------------------------------------
  var $ = function (id) { return document.getElementById(id); };
  var elHud = $('hud'), elMode = $('mode'), elLabel = $('label'),
      elCross = $('crosshair'), elFile = $('fileInput');

  if (typeof THREE === 'undefined') {
    if (elHud) elHud.innerHTML =
      '<b>three.js failed to load.</b><br>This viewer pulls three.js r136 from a CDN, ' +
      'so it needs an internet connection the first time.';
    return;
  }

  // ---- three.js scaffolding ----------------------------------------------
  var renderer = new THREE.WebGLRenderer({ antialias: true, powerPreference: 'high-performance' });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
  renderer.setSize(window.innerWidth, window.innerHeight);
  renderer.setClearColor(CFG.bg, 1);
  if ('outputEncoding' in renderer) renderer.outputEncoding = THREE.sRGBEncoding;
  document.body.appendChild(renderer.domElement);

  var scene = new THREE.Scene();
  scene.fog = new THREE.FogExp2(CFG.bg, CFG.fog);

  var camera = new THREE.PerspectiveCamera(65, window.innerWidth / window.innerHeight, 0.1, 6000);
  camera.position.set(0, 0, 120);

  // Optional bloom. If the addons didn't load, fall back to a plain render.
  var composer = null, bloom = null;
  (function setupBloom() {
    if (!THREE.EffectComposer || !THREE.RenderPass || !THREE.UnrealBloomPass) return;
    try {
      composer = new THREE.EffectComposer(renderer);
      composer.addPass(new THREE.RenderPass(scene, camera));
      bloom = new THREE.UnrealBloomPass(
        new THREE.Vector2(window.innerWidth, window.innerHeight), 0.62, 0.7, 0.2);
      composer.addPass(bloom);
    } catch (e) { composer = null; }
  })();

  // ---- graph state --------------------------------------------------------
  var nodes = [];        // {id,label,line,cluster,external,deg}
  var links = [];        // {s,t,w} (indices into nodes)
  var clusterFiles = []; // cluster index -> file label
  var N = 0, L = 0;

  // Layout arrays (structure-of-arrays for speed / low GC).
  var px, py, pz, vx, vy, vz;
  var alpha = 1, graphRadius = 60, coreRadius = 40, viewRadius = 60;
  var spreadDist = 0;    // X-key push distance, captured once per graph (0 = not yet)

  // Render objects.
  var nodeMesh = null, edgeLines = null, edgePos = null;
  var nodeGeom = new THREE.IcosahedronGeometry(1, 1);
  var coneGeom = new THREE.ConeGeometry(0.95, 2.8, 12);  // arrowhead; points +Y by default
  var nodeColors = []; // vivid THREE.Color per node (used when highlighted)
  var baseCol = [];    // darkened resting color per node
  var nodeRadius = []; // per node
  var scaleMul = null; // per-node scale multiplier (focus/neighbours bump it)
  var edgeCol = null, edgeColBase = null; // live + resting edge vertex colors
  var adjOut = [], adjIn = [];            // per-node outgoing / incoming link indices

  // GPU picking: a second instanced mesh of the nodes, each drawn in a colour
  // that encodes its index, rendered into a 1x1 buffer at the cursor. Reading
  // that pixel back gives EXACTLY the node whose real geometry is under the
  // cursor - no projection maths to get wrong, no bloom glow, correct depth for
  // free (the nearest node writes the pixel). This is the whole point: it can't
  // be "at the wrong place" because it reads what was actually drawn there.
  var pickScene = new THREE.Scene();
  var pickTarget = new THREE.WebGLRenderTarget(1, 1, { minFilter: THREE.NearestFilter, magFilter: THREE.NearestFilter });
  var pickMesh = null, pickBuf = new Uint8Array(4), _pickCol = new THREE.Color();

  // focus highlight state (the one node under the crosshair + its neighbourhood)
  var focusIdx = -1, hlNodes = [], hlEdges = [];
  var COL_FOCUS = new THREE.Color(1.0, 1.0, 1.0);   // the targeted node
  var COL_OUT = new THREE.Color(0.30, 1.0, 0.65);   // edges it calls (outgoing)
  var COL_IN = new THREE.Color(1.0, 0.5, 0.2);      // edges that call it (incoming)

  // Highlighted edges: each is drawn as a solid CYLINDER (the fat line) plus a
  // CONE (the arrowhead) - both InstancedMesh of real geometry. This is the same
  // mechanism as the arrowheads, which always render correctly; the old
  // LineSegments2 fat lines quietly dropped view-aligned / near-plane segments,
  // so some highlighted edges showed only their cone. Solid geometry can't do
  // that: every edge in the list gets a cylinder, no matter its orientation.
  var HL_MAX = 4000;                                   // per direction: cylinder/cone instance cap
  var cylGeom = new THREE.CylinderGeometry(1, 1, 1, 6); // unit cylinder along +Y (radius 1, height 1)
  var hlOut = null, hlIn = null;
  function makeHighlight(color) {
    var lmat = new THREE.MeshBasicMaterial({ color: color, toneMapped: false });
    var cyl = new THREE.InstancedMesh(cylGeom, lmat, HL_MAX);
    cyl.frustumCulled = false; cyl.count = 0;
    scene.add(cyl);
    var cmat = new THREE.MeshBasicMaterial({ color: color, toneMapped: false });
    var cones = new THREE.InstancedMesh(coneGeom, cmat, HL_MAX);
    cones.frustumCulled = false; cones.count = 0;
    scene.add(cones);
    return { cyl: cyl, cones: cones, edges: [] };
  }
  hlOut = makeHighlight(new THREE.Color(0.25, 1.0, 0.55));   // outgoing: green
  hlIn = makeHighlight(new THREE.Color(1.0, 0.5, 0.12));     // incoming: orange

  // ---- DOT / JSON parsing -------------------------------------------------
  function unesc(s) { return s.replace(/\\"/g, '"'); }

  function parseDot(text) {
    var map = Object.create(null);   // id -> node (null-proto: ids like "toString" are legal)
    var order = [];
    var lnk = [];
    var cfiles = [];
    var cur = -1;           // current cluster index (-1 = top level)

    function ensure(id) {
      var n = map[id];
      if (!n) { n = { id: id, label: id, line: 0, cluster: -1, external: true }; map[id] = n; order.push(n); }
      return n;
    }

    var reSub = /subgraph\s+cluster_(\d+)\s*\{/;
    var reEdge = /^\s*"((?:[^"\\]|\\.)*)"\s*->\s*"((?:[^"\\]|\\.)*)"\s*(?:\[([^\]]*)\])?\s*;?\s*$/;
    var reNode = /^\s*"((?:[^"\\]|\\.)*)"\s*(?:\[([^\]]*)\])?\s*;?\s*$/;
    var reLabel = /label\s*=\s*"((?:[^"\\]|\\.)*)"/;
    var reLine = /\\nL(\d+)/;

    var rows = text.split(/\r?\n/);
    for (var i = 0; i < rows.length; i++) {
      var line = rows[i];
      var t = line.trim();
      if (!t) continue;

      var m = line.match(reSub);
      if (m) { cur = parseInt(m[1], 10); continue; }
      if (t === '}') { cur = -1; continue; }

      // A cluster's own label line: "label=\"/path/File.kt\";"
      if (cur >= 0 && /^label\s*=/.test(t)) {
        var lm = t.match(reLabel);
        if (lm) cfiles[cur] = unesc(lm[1]);
        continue;
      }
      // Skip graph-level directives (digraph, rankdir, node [..], edge [..]).
      if (/^(digraph|graph|rankdir|node|edge|label|fontname|bgcolor)\b/.test(t)) continue;

      var e = line.match(reEdge);
      if (e) {
        var s = unesc(e[1]), d = unesc(e[2]);
        var w = 1;
        if (e[3]) { var wl = e[3].match(reLabel); if (wl) { var pv = parseInt(wl[1], 10); if (pv > 0) w = pv; } }
        ensure(s); ensure(d);
        lnk.push({ s: s, t: d, w: w });
        continue;
      }

      var nd = line.match(reNode);
      if (nd) {
        var name = unesc(nd[1]);
        var attrs = nd[2] || '';
        var node = ensure(name);
        if (cur >= 0) { node.cluster = cur; node.external = false; }
        if (/style\s*=\s*dashed/.test(attrs)) node.external = true;
        var la = attrs.match(reLabel);
        if (la) { var lm2 = la[1].match(reLine); if (lm2) node.line = parseInt(lm2[1], 10); }
      }
    }

    // Resolve link endpoints to node indices.
    var idx = Object.create(null); for (var k = 0; k < order.length; k++) idx[order[k].id] = k;
    var outLinks = [];
    for (var j = 0; j < lnk.length; j++) {
      var a = idx[lnk[j].s], b = idx[lnk[j].t];
      if (typeof a !== 'number' || typeof b !== 'number' || a === b) continue;
      outLinks.push({ s: a, t: b, w: lnk[j].w });
    }
    return { nodes: order, links: outLinks, clusterFiles: cfiles };
  }

  function parseJSON(text) {
    var g = JSON.parse(text);
    var order = g.nodes.map(function (n, i) {
      return {
        id: n.id != null ? String(n.id) : String(i),
        label: n.label != null ? String(n.label) : (n.id != null ? String(n.id) : String(i)),
        line: n.line || 0,
        cluster: n.cluster != null ? n.cluster : (n.group != null ? n.group : -1),
        external: !!n.external
      };
    });
    var idx = Object.create(null); order.forEach(function (n, i) { idx[n.id] = i; });
    var raw = g.links || g.edges || [];
    var outLinks = [];
    raw.forEach(function (e) {
      var a = idx[String(e.source != null ? e.source : e.s)];
      var b = idx[String(e.target != null ? e.target : e.t)];
      if (a === undefined || b === undefined || a === b) return;
      outLinks.push({ s: a, t: b, w: e.w || e.weight || 1 });
    });
    return { nodes: order, links: outLinks, clusterFiles: g.clusterFiles || [] };
  }

  // ---- a pretty synthetic graph, shown until a file is dropped ------------
  function sampleGraph() {
    var order = [], lnk = [], cfiles = [];
    var C = 7, id = 0;
    for (var c = 0; c < C; c++) {
      cfiles.push('module_' + String.fromCharCode(65 + c) + '.src');
      var size = 10 + ((Math.random() * 12) | 0);
      var base = order.length;
      for (var i = 0; i < size; i++) {
        order.push({ id: 'n' + (id++), label: 'fn_' + c + '_' + i, line: 1 + ((Math.random() * 200) | 0), cluster: c, external: false });
      }
      // dense-ish intra-cluster wiring
      for (var a = base; a < order.length; a++) {
        var m = 1 + ((Math.random() * 3) | 0);
        for (var e = 0; e < m; e++) {
          var b = base + ((Math.random() * size) | 0);
          if (b !== a) lnk.push({ s: order[a].id, t: order[b].id, w: 1 + ((Math.random() * 4) | 0) });
        }
      }
    }
    // a few inter-cluster bridges
    for (var x = 0; x < C * 4; x++) {
      var u = (Math.random() * order.length) | 0, v = (Math.random() * order.length) | 0;
      if (u !== v) lnk.push({ s: order[u].id, t: order[v].id, w: 1 });
    }
    // some shared external leaves
    for (var g = 0; g < 10; g++) {
      var ext = { id: 'ext' + g, label: 'builtin_' + g, line: 0, cluster: -1, external: true };
      order.push(ext);
      var hits = 2 + ((Math.random() * 6) | 0);
      for (var h = 0; h < hits; h++) {
        var w2 = (Math.random() * (order.length - 10)) | 0;
        lnk.push({ s: order[w2].id, t: ext.id, w: 1 });
      }
    }
    var idx = Object.create(null); order.forEach(function (n, i) { idx[n.id] = i; });
    var outLinks = lnk.map(function (e) { return { s: idx[e.s], t: idx[e.t], w: e.w }; })
      .filter(function (e) { return e.s !== e.t; });
    return { nodes: order, links: outLinks, clusterFiles: cfiles };
  }

  // ---- colors -------------------------------------------------------------
  function clusterColor(c) {
    if (c < 0) return new THREE.Color().setHSL(0.07, 0.68, 0.55); // external callees: warm amber
    var hue = (0.55 + c * 0.61803398875) % 1;                     // defined: start cool, spread by golden ratio
    return new THREE.Color().setHSL(hue, 0.6, 0.6);
  }

  // ---- build render objects for a freshly parsed graph --------------------
  function buildGraph(g) {
    // tear down old
    if (nodeMesh) { scene.remove(nodeMesh); nodeMesh.material.dispose(); nodeMesh = null; }
    if (edgeLines) { scene.remove(edgeLines); edgeLines.geometry.dispose(); edgeLines.material.dispose(); edgeLines = null; }
    if (pickMesh) { pickScene.remove(pickMesh); pickMesh.material.dispose(); pickMesh = null; }

    nodes = g.nodes; links = g.links; clusterFiles = g.clusterFiles || [];
    N = nodes.length; L = links.length;
    if (N === 0) return;

    // degrees
    for (var i = 0; i < N; i++) nodes[i].deg = 0;
    for (var e = 0; e < L; e++) { nodes[links[e].s].deg++; nodes[links[e].t].deg++; }
    var maxDeg = 1;
    for (i = 0; i < N; i++) maxDeg = Math.max(maxDeg, nodes[i].deg);

    // per-node vivid color (for highlights) + darkened resting color + radius/scale
    nodeColors = []; baseCol = []; nodeRadius = [];
    scaleMul = new Float32Array(N); scaleMul.fill(1);
    adjOut = new Array(N); adjIn = new Array(N);
    for (i = 0; i < N; i++) {
      nodeColors[i] = clusterColor(nodes[i].cluster);
      baseCol[i] = nodeColors[i].clone().multiplyScalar(CFG.nodeDim);
      nodeRadius[i] = CFG.nodeBase * (0.7 + 1.7 * Math.sqrt(nodes[i].deg / maxDeg));
      adjOut[i] = []; adjIn[i] = [];
    }
    for (e = 0; e < L; e++) { adjOut[links[e].s].push(e); adjIn[links[e].t].push(e); }
    focusIdx = -1; hlNodes.length = 0; hlEdges.length = 0;
    spreadDist = 0;                                     // re-capture the X push for this new graph

    // layout arrays + initial scatter over a LARGE sphere shell
    px = new Float32Array(N); py = new Float32Array(N); pz = new Float32Array(N);
    vx = new Float32Array(N); vy = new Float32Array(N); vz = new Float32Array(N);
    scatterNodes();

    // node instanced mesh
    var nmat = new THREE.MeshBasicMaterial({ toneMapped: false });
    nodeMesh = new THREE.InstancedMesh(nodeGeom, nmat, N);
    nodeMesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
    for (i = 0; i < N; i++) nodeMesh.setColorAt(i, baseCol[i]);
    if (nodeMesh.instanceColor) nodeMesh.instanceColor.needsUpdate = true;
    scene.add(nodeMesh);

    // pick mesh: same geometry, SHARES nodeMesh's instanceMatrix (so identical
    // positions/sizes every frame for free), each instance coloured by its index
    // (i+1, so 0 = "nothing"). Rendered to a 1x1 buffer for GPU picking.
    // white material + per-instance instanceColor (set below) => the shader
    // outputs exactly the encoded id colour (InstancedMesh multiplies the white
    // base by instanceColor; no vertexColors, which would need a vertex-colour
    // attribute the icosahedron doesn't have and would render black).
    var pmat = new THREE.MeshBasicMaterial({ color: 0xffffff, toneMapped: false, fog: false });
    pickMesh = new THREE.InstancedMesh(nodeGeom, pmat, N);
    pickMesh.instanceMatrix = nodeMesh.instanceMatrix;      // share the buffer
    pickMesh.frustumCulled = false;
    for (i = 0; i < N; i++) { var id = i + 1; _pickCol.setRGB(((id >> 16) & 255) / 255, ((id >> 8) & 255) / 255, (id & 255) / 255); pickMesh.setColorAt(i, _pickCol); }
    pickMesh.instanceColor.needsUpdate = true;
    pickScene.add(pickMesh);

    // edges: additive line segments, brighter at the caller, dim at the callee.
    // edgeColBase is the resting look; edgeCol is what's drawn (focus recolors it).
    var eg = new THREE.BufferGeometry();
    edgePos = new Float32Array(L * 6);
    edgeCol = new Float32Array(L * 6);
    edgeColBase = new Float32Array(L * 6);
    for (e = 0; e < L; e++) {
      var cs = baseCol[links[e].s], ct = baseCol[links[e].t];
      var o = e * 6;
      edgeColBase[o] = cs.r * 0.9; edgeColBase[o + 1] = cs.g * 0.9; edgeColBase[o + 2] = cs.b * 0.9;
      edgeColBase[o + 3] = ct.r * 0.45; edgeColBase[o + 4] = ct.g * 0.45; edgeColBase[o + 5] = ct.b * 0.45;
    }
    edgeCol.set(edgeColBase);
    eg.setAttribute('position', new THREE.BufferAttribute(edgePos, 3));
    eg.setAttribute('color', new THREE.BufferAttribute(edgeCol, 3));
    var emat = new THREE.LineBasicMaterial({
      vertexColors: true, transparent: true, opacity: 0.5,
      blending: THREE.AdditiveBlending, depthWrite: false, toneMapped: false
    });
    edgeLines = new THREE.LineSegments(eg, emat);
    edgeLines.frustumCulled = false;
    scene.add(edgeLines);

    // warm the layout up so the first frame isn't a random cloud
    for (i = 0; i < CFG.warmup; i++) layoutTick();
    updateGraphRadius();

    // start outside, flying inward
    graphRadius = Math.max(graphRadius, 40);
    // Start outside, framing most of the graph for an establishing overview,
    // then fly in. Frame by viewRadius, not the outlier-dominated max.
    var halfFov = THREE.MathUtils.degToRad(camera.fov * 0.5);
    var frameDist = viewRadius / Math.sin(halfFov) * 1.1;
    camera.position.set(0, viewRadius * 0.35, frameDist);
    var fwd = new THREE.Vector3(0, 0, 0).sub(camera.position).normalize();
    camera.lookAt(0, 0, 0);            // face the graph from the overview spot
    seedAuto(camera.position, fwd);    // prime the spline in case autopilot is turned on
    mode = autopilot ? 'auto' : 'fly'; // autopilot off (default) -> park here, let the user drive

    updateHud();
  }

  // Scatter nodes over a large sphere shell, give them room, and re-heat the
  // sim. Used at load AND on re-heat (Space): connected groups then clump as
  // repulsion ramps up while cooling. Re-heat blows the graph apart to re-sort
  // - not compress it - which is the whole point of the ramp.
  function scatterNodes() {
    var R0 = CFG.restLen * Math.cbrt(N) * CFG.initSpread + 40;
    for (var i = 0; i < N; i++) {
      var yy = 1 - (i + 0.5) / N * 2;                 // -1..1, evenly stratified
      var rad = Math.sqrt(Math.max(0, 1 - yy * yy));
      var th = i * 2.399963229728653;                 // golden angle -> even spiral on the shell
      var rr = R0 * (0.9 + Math.random() * 0.2);       // slight radial jitter
      px[i] = Math.cos(th) * rad * rr;
      py[i] = yy * rr;
      pz[i] = Math.sin(th) * rad * rr;
      vx[i] = vy[i] = vz[i] = 0;
    }
    alpha = 1;
  }

  // Spread (X): unlike re-heat's random scatter, this keeps every node's
  // DIRECTION from the centre and only pushes it outward, so the angular
  // structure (which clusters sit where) is preserved while the graph inflates
  // and re-sorts.
  //
  // The push distance is captured ONCE, on the first X for this graph: the
  // furthest node's distance from the centre * spreadMult. Every later X reuses
  // that same stored distance, so each press adds a FIXED amount to every node's
  // distance from the centre - the graph inflates by equal steps, instead of by
  // an ever-growing multiple of an already-spread graph (which would run away).
  // spreadDist resets to 0 on load (buildGraph), so a new graph re-captures.
  function spreadNodes() {
    if (N === 0) return;
    var cx = 0, cy = 0, cz = 0, i;
    for (i = 0; i < N; i++) { cx += px[i]; cy += py[i]; cz += pz[i]; }
    cx /= N; cy /= N; cz /= N;
    if (spreadDist === 0) {                             // first X: measure & remember the push
      var maxD = 0;
      for (i = 0; i < N; i++) {
        var ex = px[i] - cx, ey = py[i] - cy, ez = pz[i] - cz;
        var d = Math.sqrt(ex * ex + ey * ey + ez * ez);
        if (d > maxD) maxD = d;
      }
      spreadDist = maxD * CFG.spreadMult;
    }
    var push = spreadDist;                              // same additive push on every press
    for (i = 0; i < N; i++) {
      var dx = px[i] - cx, dy = py[i] - cy, dz = pz[i] - cz;
      var dd = Math.sqrt(dx * dx + dy * dy + dz * dz);
      if (dd < 1e-4) continue;                          // a node exactly at the centre has no direction
      var k = (dd + push) / dd;                         // same direction, dd + push out from centre
      px[i] = cx + dx * k; py[i] = cy + dy * k; pz[i] = cz + dz * k;
      vx[i] = vy[i] = vz[i] = 0;
    }
    alpha = 1;                                          // re-heat so the forces re-settle it
  }

  // ---- force layout (grid-accelerated repulsion) --------------------------
  var ax = null, ay = null, az = null;
  function layoutTick() {
    if (!ax || ax.length !== N) { ax = new Float32Array(N); ay = new Float32Array(N); az = new Float32Array(N); }
    ax.fill(0); ay.fill(0); az.fill(0);

    // spatial hash so repulsion is ~O(N) instead of O(N^2)
    var cell = CFG.restLen * 1.8;
    var grid = {};
    for (var i = 0; i < N; i++) {
      var key = ((px[i] / cell) | 0) + ',' + ((py[i] / cell) | 0) + ',' + ((pz[i] / cell) | 0);
      (grid[key] || (grid[key] = [])).push(i);
    }

    // Repulsion ramps from repMin (hot) to full (cooled), so early on springs and
    // gravity win and groups clump; later, repulsion enforces even spacing.
    var repScale = CFG.repMin + (1 - CFG.repMin) * (1 - (alpha - CFG.alphaMin) / (1 - CFG.alphaMin));
    var repK = CFG.repulsion * repScale;

    // repulsion within the 27-cell neighborhood
    for (i = 0; i < N; i++) {
      var cx = (px[i] / cell) | 0, cy = (py[i] / cell) | 0, cz = (pz[i] / cell) | 0;
      for (var dx = -1; dx <= 1; dx++) for (var dy = -1; dy <= 1; dy++) for (var dz = -1; dz <= 1; dz++) {
        var bucket = grid[(cx + dx) + ',' + (cy + dy) + ',' + (cz + dz)];
        if (!bucket) continue;
        for (var bi = 0; bi < bucket.length; bi++) {
          var j = bucket[bi];
          if (j <= i) continue;
          var ddx = px[i] - px[j], ddy = py[i] - py[j], ddz = pz[i] - pz[j];
          var d2 = ddx * ddx + ddy * ddy + ddz * ddz;
          if (d2 < 0.01) { ddx = (Math.random() - 0.5); ddy = (Math.random() - 0.5); ddz = (Math.random() - 0.5); d2 = 0.01; }
          var inv = repK / d2;
          var d = Math.sqrt(d2);
          var fx = ddx / d * inv, fy = ddy / d * inv, fz = ddz / d * inv;
          ax[i] += fx; ay[i] += fy; az[i] += fz;
          ax[j] -= fx; ay[j] -= fy; az[j] -= fz;
        }
      }
    }

    // Spring attraction along edges. A plain Hooke spring, f = spring*(dl-restLen),
    // already pulls harder the more an edge is stretched - but only LINEARLY. The
    // springStretch knob adds a super-linear boost while stretched, multiplying the
    // force by (1 + springStretch*stretch/restLen), so far-apart connected nodes are
    // yanked back together disproportionately hard (collapsing long bridges, balling
    // clusters up). A compressed edge (dl < restLen) keeps the plain linear push-out.
    for (var e = 0; e < L; e++) {
      var s = links[e].s, t = links[e].t;
      var ex = px[t] - px[s], ey = py[t] - py[s], ez = pz[t] - pz[s];
      var dl = Math.sqrt(ex * ex + ey * ey + ez * ez) || 0.0001;
      var stretch = dl - CFG.restLen;
      var f = CFG.spring * stretch;
      if (stretch > 0) f *= 1 + CFG.springStretch * stretch / CFG.restLen;
      var gx = ex / dl * f, gy = ey / dl * f, gz = ez / dl * f;
      ax[s] += gx; ay[s] += gy; az[s] += gz;
      ax[t] -= gx; ay[t] -= gy; az[t] -= gz;
    }

    // gravity toward origin + integrate
    var g = CFG.gravity, damp = CFG.damping, mx = CFG.maxStep;
    for (i = 0; i < N; i++) {
      var accx = (ax[i] - px[i] * g) * alpha;
      var accy = (ay[i] - py[i] * g) * alpha;
      var accz = (az[i] - pz[i] * g) * alpha;
      vx[i] = (vx[i] + accx) * damp;
      vy[i] = (vy[i] + accy) * damp;
      vz[i] = (vz[i] + accz) * damp;
      var sp = Math.sqrt(vx[i] * vx[i] + vy[i] * vy[i] + vz[i] * vz[i]);
      if (sp > mx) { var k = mx / sp; vx[i] *= k; vy[i] *= k; vz[i] *= k; }
      px[i] += vx[i]; py[i] += vy[i]; pz[i] += vz[i];
    }

    if (alpha > CFG.alphaMin) { alpha *= CFG.alphaDecay; if (alpha < CFG.alphaMin) alpha = CFG.alphaMin; }
  }

  var _dists = null;
  function updateGraphRadius() {
    if (!_dists || _dists.length !== N) _dists = new Float32Array(N);
    var m = 1;
    for (var i = 0; i < N; i++) {
      var d = px[i] * px[i] + py[i] * py[i] + pz[i] * pz[i];
      _dists[i] = Math.sqrt(d);
      if (d > m) m = d;
    }
    graphRadius = Math.sqrt(m);
    // coreRadius = radius of the DENSE body (a percentile of node distances),
    // ignoring the diffuse shell of low-degree stragglers the layout flings out.
    // The autocam lives inside this, so it stays where the graph actually is.
    _dists.sort();
    coreRadius = _dists[(N * 0.55) | 0] || graphRadius;
    if (coreRadius < 20) coreRadius = 20;
    // viewRadius = most of the graph (ignores the ~15% most-flung outliers that
    // would otherwise blow up framing). Used only to place the overview camera.
    viewRadius = _dists[(N * 0.85) | 0] || graphRadius;
    if (viewRadius < coreRadius * 1.2) viewRadius = coreRadius * 1.2;
  }

  // ---- push node/edge positions into the GPU buffers ----------------------
  var _m = new THREE.Matrix4(), _q = new THREE.Quaternion(), _s = new THREE.Vector3(), _p = new THREE.Vector3();
  function updateRender() {
    if (!nodeMesh) return;
    for (var i = 0; i < N; i++) {
      var r = nodeRadius[i] * scaleMul[i];
      _p.set(px[i], py[i], pz[i]); _s.set(r, r, r);
      _m.compose(_p, _q, _s);
      nodeMesh.setMatrixAt(i, _m);
    }
    nodeMesh.instanceMatrix.needsUpdate = true;

    var ep = edgePos;
    for (var e = 0; e < L; e++) {
      var s = links[e].s, t = links[e].t, o = e * 6;
      ep[o] = px[s]; ep[o + 1] = py[s]; ep[o + 2] = pz[s];
      ep[o + 3] = px[t]; ep[o + 4] = py[t]; ep[o + 5] = pz[t];
    }
    edgeLines.geometry.attributes.position.needsUpdate = true;

    if (focusIdx >= 0) updateHighlightGeom();           // keep the fat edges on their moving endpoints
  }

  // ---- camera: shared state ----------------------------------------------
  var mode = 'fly';                  // 'auto' (autopilot flying) | 'fly' (user in control)
  var autopilot = false;             // may the camera fly itself when idle? (off by default)
  var lookMode = 'cursor';           // 'capture' (pointer-lock look) | 'cursor' (hover to inspect - default)
  var lastInput = -1e9;
  var keys = {};
  var flyVel = new THREE.Vector3();
  var speedMul = 1;
  var cursorX = window.innerWidth / 2, cursorY = window.innerHeight / 2;
  // free-look: mouse rotates the camera about its OWN axes (no fixed up), so you
  // can look past the poles and the response is the same whichever way you face.
  var _AX_Y = new THREE.Vector3(0, 1, 0), _AX_X = new THREE.Vector3(1, 0, 0), _qTmp = new THREE.Quaternion();

  // ---- camera: AUTO spline ------------------------------------------------
  var A = {
    p0: new THREE.Vector3(), p1: new THREE.Vector3(), p2: new THREE.Vector3(), p3: new THREE.Vector3(),
    t: 0, offset: new THREE.Vector3(), look: new THREE.Vector3(0, 0, -1)
  };
  // A Camera (not a plain Object3D): Object3D.lookAt() orients a generic
  // object's +Z at the target, but a Camera's -Z - which is what we need, since
  // we copy this quaternion onto the real camera (cameras look down -Z).
  var _dummy = new THREE.Camera();

  function nextWaypoint(out) {
    // Pick a well-connected node - a tournament of a few samples biases toward
    // the hubs in the dense core - and aim near it with a modest jitter, so the
    // path threads the busy interior. Now and then dart across the core.
    var i = (Math.random() * N) | 0;
    for (var s = 0; s < 3; s++) { var j = (Math.random() * N) | 0; if (nodes[j].deg > nodes[i].deg) i = j; }
    out.set(px[i], py[i], pz[i]);
    var jitter = coreRadius * (0.08 + Math.random() * 0.22);
    out.x += (Math.random() - 0.5) * jitter;
    out.y += (Math.random() - 0.5) * jitter;
    out.z += (Math.random() - 0.5) * jitter;
    if (Math.random() < 0.35) out.multiplyScalar(-0.75); // dart across the core
    var len = out.length(), cap = coreRadius * 1.05;     // keep waypoints within the dense body
    if (len > cap) out.multiplyScalar(cap / len);
    return out;
  }

  function seedAuto(pos, dir) {
    var seg = coreRadius * 0.8 + 10;
    A.p1.copy(pos);
    A.p0.copy(pos).addScaledVector(dir, -seg);
    A.p2.copy(pos).addScaledVector(dir, seg);
    nextWaypoint(A.p3);
    A.t = 0; A.offset.set(0, 0, 0); A.look.copy(dir).normalize();
  }

  function catmull(p0, p1, p2, p3, t, out) {
    var t2 = t * t, t3 = t2 * t;
    out.x = 0.5 * (2 * p1.x + (-p0.x + p2.x) * t + (2 * p0.x - 5 * p1.x + 4 * p2.x - p3.x) * t2 + (-p0.x + 3 * p1.x - 3 * p2.x + p3.x) * t3);
    out.y = 0.5 * (2 * p1.y + (-p0.y + p2.y) * t + (2 * p0.y - 5 * p1.y + 4 * p2.y - p3.y) * t2 + (-p0.y + 3 * p1.y - 3 * p2.y + p3.y) * t3);
    out.z = 0.5 * (2 * p1.z + (-p0.z + p2.z) * t + (2 * p0.z - 5 * p1.z + 4 * p2.z - p3.z) * t2 + (-p0.z + 3 * p1.z - 3 * p2.z + p3.z) * t3);
    return out;
  }

  var _pos = new THREE.Vector3(), _ahead = new THREE.Vector3(), _tan = new THREE.Vector3(), _avoid = new THREE.Vector3();
  var _tmpLook = new THREE.Vector3();
  function autoStep(dt) {
    var segLen = _ahead.copy(A.p2).sub(A.p1).length() || 1;
    A.t += CFG.autoSpeed * dt / segLen;
    while (A.t >= 1) {
      A.t -= 1;
      A.p0.copy(A.p1); A.p1.copy(A.p2); A.p2.copy(A.p3);
      nextWaypoint(A.p3);
    }
    catmull(A.p0, A.p1, A.p2, A.p3, A.t, _pos);
    catmull(A.p0, A.p1, A.p2, A.p3, Math.min(A.t + 0.03, 1), _ahead);
    _tan.copy(_ahead).sub(_pos);
    if (_tan.lengthSq() < 1e-6) _tan.set(0, 0, -1);
    _tan.normalize();

    // Node avoidance: a SOFT nudge away from nodes we'd otherwise clip, with a
    // linear falloff (0 at avoidRadius, max when touching) and a small absolute
    // cap. It steers the camera through the gaps; it must never shove it out of
    // the graph, so the cap is a few node-radii, not a fraction of graphRadius.
    _avoid.set(0, 0, 0);
    var ar = CFG.avoidRadius, r2 = ar * ar;
    for (var i = 0; i < N; i++) {
      var dx = _pos.x - px[i], dy = _pos.y - py[i], dz = _pos.z - pz[i];
      var d2 = dx * dx + dy * dy + dz * dz;
      if (d2 < r2 && d2 > 1e-4) {
        var d = Math.sqrt(d2);
        var w = (ar - d) / ar;                        // 1 touching -> 0 at the rim
        var f = CFG.avoidStrength * w * w / d;         // dx*f yields a unit push * strength
        _avoid.x += dx * f; _avoid.y += dy * f; _avoid.z += dz * f;
      }
    }
    if (_avoid.length() > CFG.maxAvoid) _avoid.setLength(CFG.maxAvoid);
    A.offset.lerp(_avoid, CFG.avoidSmooth);

    camera.position.copy(_pos).add(A.offset);
    // Look along the direction of travel (forward flight), but smooth the look
    // TARGET (low-pass the tangent) and cap the turn RATE, so sharp spline
    // curvature - or a node dodge - can never yank the camera around fast.
    A.look.lerp(_tan, CFG.lookSmooth);
    if (A.look.lengthSq() < 1e-6) A.look.copy(_tan);
    A.look.normalize();
    _dummy.position.copy(camera.position);
    _dummy.lookAt(_tmpLook.copy(camera.position).add(A.look));
    var ang = camera.quaternion.angleTo(_dummy.quaternion);
    if (ang > 1e-5) {
      var turn = Math.min(ang * CFG.autoLook, CFG.maxTurn * dt) / ang;
      camera.quaternion.slerp(_dummy.quaternion, turn);
    }
  }

  // ---- camera: FLY --------------------------------------------------------
  // The camera quaternion is owned by the mouse handler; here we only translate
  // along the camera's OWN axes (forward/right/up all relative to where it looks),
  // so movement feels identical whichever way you're facing.
  var _fwd = new THREE.Vector3(), _right = new THREE.Vector3(), _up = new THREE.Vector3(), _acc = new THREE.Vector3();
  function flyStep(dt) {
    _fwd.set(0, 0, -1).applyQuaternion(camera.quaternion);
    _right.set(1, 0, 0).applyQuaternion(camera.quaternion);
    _up.set(0, 1, 0).applyQuaternion(camera.quaternion);
    _acc.set(0, 0, 0);
    if (keys['w']) _acc.add(_fwd);
    if (keys['s']) _acc.sub(_fwd);
    if (keys['d']) _acc.add(_right);
    if (keys['a']) _acc.sub(_right);
    if (keys['e']) _acc.add(_up);
    if (keys['q']) _acc.sub(_up);
    if (_acc.lengthSq() > 0) _acc.normalize();
    var speed = CFG.flySpeed * speedMul * (keys['shift'] ? CFG.flyBoost : 1);
    _acc.multiplyScalar(speed);
    // frame-rate independent easing toward the target velocity
    var k = 1 - Math.pow(0.0015, dt);
    flyVel.lerp(_acc, k);
    camera.position.addScaledVector(flyVel, dt);
  }

  // ---- input --------------------------------------------------------------
  function now() { return performance.now(); }

  function registerSteer() {
    lastInput = now();
    if (mode !== 'fly') enterFly();
  }
  function enterFly() {
    mode = 'fly';
    flyVel.set(0, 0, 0);   // keep the current orientation; the mouse owns it from here
    updateHud();
  }
  function enterAuto() {
    if (!autopilot) return;            // never take over when autopilot is off
    mode = 'auto';
    var dir = camera.getWorldDirection(new THREE.Vector3());
    seedAuto(camera.position, dir);
    if (document.pointerLockElement) document.exitPointerLock();
    updateHud();
  }

  var MOVEKEYS = { w: 1, a: 1, s: 1, d: 1, q: 1, e: 1, shift: 1 };
  window.addEventListener('keydown', function (ev) {
    var k = ev.key.toLowerCase();
    if (k === ' ') { scatterNodes(); ev.preventDefault(); return; } // re-heat: random scatter & re-sort
    if (k === 'x') { spreadNodes(); ev.preventDefault(); return; }  // spread: push out radially, keep direction
    if (k === 'h') { document.body.classList.toggle('hideui'); return; }
    if (MOVEKEYS[k]) { keys[k] = true; registerSteer(); }
  });
  window.addEventListener('keyup', function (ev) { keys[ev.key.toLowerCase()] = false; });

  // Two mouse modes (toggled from the dashboard):
  //   capture - click grabs the pointer (proper FPS capture); moving the captured
  //             mouse turns the camera; a fixed centre crosshair is the aim point.
  //   cursor  - the visible cursor IS the crosshair; hovering picks the node under
  //             it (no camera turn); click-drag turns; WASD moves. Good for
  //             inspecting connections without flying.
  var dragging = false;
  function rotateCam(dx, dy) {
    // Post-multiply by rotations about canonical axes => LOCAL-frame yaw/pitch:
    // no pole clamp, no gimbal, free look in any direction.
    _qTmp.setFromAxisAngle(_AX_Y, -dx * CFG.mouseSens); camera.quaternion.multiply(_qTmp);
    _qTmp.setFromAxisAngle(_AX_X, -dy * CFG.mouseSens); camera.quaternion.multiply(_qTmp);
    camera.quaternion.normalize();
    registerSteer();
  }
  renderer.domElement.addEventListener('mousedown', function (ev) {
    dragging = true; document.body.classList.add('grabbing');
    if (lookMode === 'capture' && renderer.domElement.requestPointerLock) renderer.domElement.requestPointerLock();
    if (lookMode === 'capture') registerSteer();       // cursor-mode: a plain click shouldn't hijack from autopilot
    ev.preventDefault();
  });
  window.addEventListener('mouseup', function () { dragging = false; document.body.classList.remove('grabbing'); });
  document.addEventListener('pointerlockchange', updateCrosshair);
  window.addEventListener('mousemove', function (ev) {
    cursorX = ev.clientX; cursorY = ev.clientY;         // always track for cursor-mode focus
    var locked = document.pointerLockElement === renderer.domElement;
    var turn = (lookMode === 'capture') ? (locked || dragging) : dragging;
    if (turn) rotateCam(ev.movementX || 0, ev.movementY || 0);
  });
  window.addEventListener('wheel', function (ev) {
    speedMul *= ev.deltaY < 0 ? 1.1 : 0.9;
    speedMul = Math.max(0.15, Math.min(6, speedMul));
    registerSteer();
  }, { passive: true });

  function updateCrosshair() {
    var locked = document.pointerLockElement === renderer.domElement;
    elCross.style.display = (lookMode === 'capture') ? 'block' : 'none';
    document.body.classList.toggle('cursormode', lookMode === 'cursor');
    if (!locked) { dragging = false; document.body.classList.remove('grabbing'); }
  }

  // dashboard switches
  var elTgAuto = $('tgAuto'), elTgMouse = $('tgMouse');
  if (elTgAuto) elTgAuto.addEventListener('click', function () {
    autopilot = !autopilot;
    elTgAuto.classList.toggle('on', autopilot);
    elTgAuto.textContent = 'Autopilot: ' + (autopilot ? 'on' : 'off');
    if (!autopilot && mode === 'auto') enterFly();
    lastInput = now();
  });
  if (elTgMouse) elTgMouse.addEventListener('click', function () {
    lookMode = (lookMode === 'capture') ? 'cursor' : 'capture';
    elTgMouse.classList.toggle('on', lookMode === 'cursor');
    elTgMouse.textContent = 'Mouse: ' + (lookMode === 'capture' ? 'Fly' : 'Inspect');
    if (lookMode === 'cursor' && document.pointerLockElement) document.exitPointerLock();
    updateCrosshair();
  });
  updateCrosshair();

  // ---- focus highlight + node labels --------------------------------------
  // The node nearest the crosshair (screen centre) becomes the "focus": it lights
  // up white, its OUTGOING call edges turn green and INCOMING edges orange, and
  // its direct neighbours brighten too. Nearby nodes get their name drawn on them.
  var labelLayer = document.createElement('div');
  labelLayer.id = 'labelLayer';
  document.body.appendChild(labelLayer);
  var labelPool = [];
  function getLabel(k) {
    while (labelPool.length <= k) {
      var d = document.createElement('div'); d.className = 'nlabel';
      labelLayer.appendChild(d); labelPool.push(d);
    }
    return labelPool[k];
  }

  function setEdgeColor(e, col) {
    // Boost past 1.0 so the additive line blooms in its own colour (green/orange)
    // instead of washing to white next to the bright focus node.
    var o = e * 6, b = 1.9;
    edgeCol[o] = col.r * b; edgeCol[o + 1] = col.g * b; edgeCol[o + 2] = col.b * b;
    edgeCol[o + 3] = col.r * b; edgeCol[o + 4] = col.g * b; edgeCol[o + 5] = col.b * b;
  }
  function markNeighbour(i) {
    if (i === focusIdx) return;
    nodeMesh.setColorAt(i, nodeColors[i]);          // full vivid (brighter than base, not white)
    if (scaleMul[i] < 1.25) scaleMul[i] = 1.25;
    hlNodes.push(i);
  }
  function applyFocus(nf) {
    var a, i, e, o, c, k;
    // restore whatever was highlighted last frame
    for (a = 0; a < hlNodes.length; a++) { i = hlNodes[a]; nodeMesh.setColorAt(i, baseCol[i]); scaleMul[i] = 1; }
    for (a = 0; a < hlEdges.length; a++) { e = hlEdges[a]; o = e * 6; for (c = 0; c < 6; c++) edgeCol[o + c] = edgeColBase[o + c]; }
    hlNodes.length = 0; hlEdges.length = 0;
    hlOut.edges.length = 0; hlIn.edges.length = 0;
    focusIdx = nf;
    if (nf >= 0) {
      nodeMesh.setColorAt(nf, COL_FOCUS); scaleMul[nf] = 1.4; hlNodes.push(nf);
      var lo = adjOut[nf], li = adjIn[nf];
      // Each highlighted edge gets a solid cylinder (fat line) + cone (arrowhead)
      // built below; its thin base edge is also recoloured as a subtle underglow.
      for (k = 0; k < lo.length; k++) {
        markNeighbour(links[lo[k]].t); setEdgeColor(lo[k], COL_OUT); hlEdges.push(lo[k]);
        hlOut.edges.push(lo[k]);
      }
      for (k = 0; k < li.length; k++) {
        markNeighbour(links[li[k]].s); setEdgeColor(li[k], COL_IN); hlEdges.push(li[k]);
        hlIn.edges.push(li[k]);
      }
    }
    if (nodeMesh.instanceColor) nodeMesh.instanceColor.needsUpdate = true;
    edgeLines.geometry.attributes.color.needsUpdate = true;
    updateHighlightGeom();
  }

  // Rebuild the highlight geometry - a cylinder (fat line) and a cone (arrowhead)
  // per focus edge - from the current node positions. Redone each frame so they
  // track the moving endpoints; only the focus node's edges, so a few dozen
  // instances. Solid geometry ALWAYS renders, so every listed edge shows a line.
  function updateHighlightGeom() {
    updateHighlightLine(hlOut);
    updateHighlightLine(hlIn);
  }
  var _cUp = new THREE.Vector3(0, 1, 0), _cd = new THREE.Vector3(), _cq = new THREE.Quaternion(),
      _cpos = new THREE.Vector3(), _cscale = new THREE.Vector3(1, 1, 1), _cm = new THREE.Matrix4();
  function updateHighlightLine(fl) {
    var n = Math.min(fl.edges.length, HL_MAX);
    if (n === 0) { fl.cyl.count = 0; fl.cones.count = 0; return; }
    for (var k = 0; k < n; k++) {
      var e = fl.edges[k], s = links[e].s, t = links[e].t;
      var sx = px[s], sy = py[s], sz = pz[s], tx = px[t], ty = py[t], tz = pz[t];
      _cd.set(tx - sx, ty - sy, tz - sz);
      var len = _cd.length() || 1e-3; _cd.multiplyScalar(1 / len);
      _cq.setFromUnitVectors(_cUp, _cd);              // orient +Y along source -> target
      // fat line: a thin cylinder spanning source -> target (centred at midpoint)
      _cpos.set((sx + tx) * 0.5, (sy + ty) * 0.5, (sz + tz) * 0.5);
      _cscale.set(CFG.lineRadius, len, CFG.lineRadius);
      _cm.compose(_cpos, _cq, _cscale);
      fl.cyl.setMatrixAt(k, _cm);
      // arrowhead: a cone just outside the target node, same orientation
      var back = nodeRadius[t] + 2.0;
      _cpos.set(tx - _cd.x * back, ty - _cd.y * back, tz - _cd.z * back);
      _cscale.set(1, 1, 1);
      _cm.compose(_cpos, _cq, _cscale);
      fl.cones.setMatrixAt(k, _cm);
    }
    fl.cyl.count = n; fl.cyl.instanceMatrix.needsUpdate = true;
    fl.cones.count = n; fl.cones.instanceMatrix.needsUpdate = true;
  }

  // GPU pick: render the pick mesh's 1x1 region at (aimX, aimY) into pickTarget
  // and read the pixel back. camera.setViewOffset squeezes exactly that one
  // screen pixel to fill the 1x1 buffer, so the readback is the node actually
  // drawn under the cursor - or -1 for the background. Coordinates are plain CSS
  // pixels (clientX/Y), the same space setViewOffset wants, so there is nothing
  // to misalign; devicePixelRatio, FOV and perspective are all already baked
  // into the camera that draws it.
  function pickAt(aimX, aimY, w, h) {
    if (!pickMesh) return -1;
    camera.setViewOffset(w, h, aimX | 0, aimY | 0, 1, 1);
    renderer.setRenderTarget(pickTarget);
    renderer.setClearColor(0x000000, 1);
    renderer.render(pickScene, camera);          // auto-clears the 1x1 target first
    renderer.setRenderTarget(null);
    renderer.setClearColor(CFG.bg, 1);
    camera.clearViewOffset();
    renderer.readRenderTargetPixels(pickTarget, 0, 0, 1, 1, pickBuf);
    var id = (pickBuf[0] << 16) | (pickBuf[1] << 8) | pickBuf[2];
    return (id > 0 && id <= N) ? id - 1 : -1;
  }

  var _pv = new THREE.Vector3(), labelCands = [];
  function updateOverlays() {
    if (!nodeMesh || N === 0) { elLabel.style.display = 'none'; return; }
    // Work in the CANVAS's own coordinate space, not the viewport's. The cursor
    // is clientX/Y (viewport-relative) but the render fills the canvas, which is
    // NOT guaranteed to sit at (0,0) - a stray in-flow panel can push it down.
    // getBoundingClientRect gives the canvas's on-screen box, so we subtract its
    // origin for picking and add it back for the (viewport-positioned) labels.
    var rect = renderer.domElement.getBoundingClientRect();
    var cw = rect.width, ch = rect.height;
    if (cw < 1 || ch < 1) return;
    // aim point (canvas-local): the moving cursor in inspect mode, the centre otherwise
    var aimX = (lookMode === 'cursor') ? (cursorX - rect.left) : cw * 0.5;
    var aimY = (lookMode === 'cursor') ? (cursorY - rect.top) : ch * 0.5;
    var best = pickAt(aimX, aimY, cw, ch);
    if (best !== focusIdx) applyFocus(best);
    // nearby-node name labels: project only the nodes close to the camera
    var labD = coreRadius * CFG.labelDist, labD2 = labD * labD;
    var camx = camera.position.x, camy = camera.position.y, camz = camera.position.z;
    labelCands.length = 0;
    for (var i = 0; i < N; i++) {
      var dx = px[i] - camx, dy = py[i] - camy, dz = pz[i] - camz, wd = dx * dx + dy * dy + dz * dz;
      if (wd < labD2) {
        _pv.set(px[i], py[i], pz[i]).project(camera);
        if (_pv.z <= 1) labelCands.push({ i: i, sx: (_pv.x * 0.5 + 0.5) * cw + rect.left, sy: (-_pv.y * 0.5 + 0.5) * ch + rect.top, wd: wd });
      }
    }
    renderFocusLabel(cw, ch, rect.left, rect.top);
    renderNearLabels(labD);
  }

  function renderFocusLabel(w, h, ox, oy) {
    if (focusIdx < 0) { elLabel.style.display = 'none'; return; }
    _pv.set(px[focusIdx], py[focusIdx], pz[focusIdx]).project(camera);
    var n = nodes[focusIdx];
    var where = n.external ? 'external' : (clusterFiles[n.cluster] ? baseName(clusterFiles[n.cluster]) : 'cluster ' + n.cluster);
    elLabel.style.display = 'block';
    elLabel.style.left = ((_pv.x * 0.5 + 0.5) * w + ox + 16) + 'px';
    elLabel.style.top = ((-_pv.y * 0.5 + 0.5) * h + oy - 10) + 'px';
    elLabel.innerHTML = '<b>' + escapeHtml(n.id) + '</b>' + (n.line ? ' :' + n.line : '') +
      '<span class="dim"> &middot; ' + escapeHtml(where) + ' &middot; ' + n.deg + ' link' + (n.deg === 1 ? '' : 's') + '</span>';
  }

  function renderNearLabels(fade) {
    labelCands.sort(function (a, b) { return a.wd - b.wd; });
    var shown = 0;
    for (var c = 0; c < labelCands.length && shown < CFG.labelMax; c++) {
      var lc = labelCands[c];
      if (lc.i === focusIdx) continue;               // the focus has its own bigger label
      var d = getLabel(shown); shown++;
      var name = nodes[lc.i].id;
      if (d._name !== name) { d.textContent = name; d._name = name; }
      d.style.left = (lc.sx + 7) + 'px';
      d.style.top = (lc.sy - 7) + 'px';
      var op = (1 - Math.sqrt(lc.wd) / fade) * 0.85;
      d.style.opacity = (op < 0 ? 0 : op).toFixed(2);
      d.style.display = 'block';
    }
    for (var k = shown; k < labelPool.length; k++) labelPool[k].style.display = 'none';
  }

  function baseName(p) { var i = Math.max(p.lastIndexOf('/'), p.lastIndexOf('\\')); return i >= 0 ? p.slice(i + 1) : p; }
  function escapeHtml(s) { return s.replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }

  // ---- HUD ----------------------------------------------------------------
  function updateHud() {
    elMode.textContent = mode === 'auto' ? 'AUTO' : 'FLY';
    // keep the `panel` class! setting className = 'fly' dropped it, which lost
    // position:fixed, so the bar fell into flow and pushed the canvas down 34px
    // - offsetting every cursor pick by that much until the panel was hidden.
    elMode.className = 'panel ' + (mode === 'auto' ? 'auto' : 'fly');
    var files = clusterFiles.filter(function (x) { return x; }).length;
    elHud.innerHTML =
      '<b>' + N + '</b> nodes &nbsp; <b>' + L + '</b> edges' +
      (files ? ' &nbsp; <b>' + files + '</b> file' + (files > 1 ? 's' : '') : '');
  }

  // ---- file loading -------------------------------------------------------
  function loadText(name, text) {
    try {
      var g = /\.json$/i.test(name) ? parseJSON(text) : parseDot(text);
      if (!g.nodes.length) { flash('No nodes found in ' + name); return; }
      buildGraph(g);
      flash('Loaded ' + baseName(name) + '  (' + g.nodes.length + ' nodes, ' + g.links.length + ' edges)');
    } catch (err) { flash('Failed to parse ' + name + ': ' + err.message); }
  }
  function readFile(file) {
    var fr = new FileReader();
    fr.onload = function () { loadText(file.name, String(fr.result)); };
    fr.readAsText(file);
  }
  function loadUrl(url) {
    flash('Fetching ' + url + ' ...');
    fetch(url).then(function (r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.text(); })
      .then(function (t) { loadText(url, t); })
      .catch(function (e) { flash('Could not fetch ' + url + ' (' + e.message + '). Drag the file in instead.'); });
  }
  elFile.addEventListener('change', function () { if (elFile.files[0]) readFile(elFile.files[0]); });
  window.addEventListener('dragover', function (e) { e.preventDefault(); document.body.classList.add('dragging'); });
  window.addEventListener('dragleave', function (e) { if (e.target === document.body) document.body.classList.remove('dragging'); });
  window.addEventListener('drop', function (e) {
    e.preventDefault(); document.body.classList.remove('dragging');
    if (e.dataTransfer.files[0]) readFile(e.dataTransfer.files[0]);
  });

  var flashT = null;
  function flash(msg) {
    var f = $('flash'); f.textContent = msg; f.style.opacity = '1';
    if (flashT) clearTimeout(flashT);
    flashT = setTimeout(function () { f.style.opacity = '0'; }, 2600);
  }

  // ---- resize -------------------------------------------------------------
  window.addEventListener('resize', function () {
    var w = window.innerWidth, h = window.innerHeight;
    camera.aspect = w / h; camera.updateProjectionMatrix();
    renderer.setSize(w, h);
    if (composer) composer.setSize(w, h);
    if (bloom) bloom.setSize(w, h);
    // (GPU picking renders a 1x1 region via camera.setViewOffset, so it needs no resize)
  });

  // ---- main loop ----------------------------------------------------------
  var prev = now(), frame = 0;
  function animate() {
    requestAnimationFrame(animate);
    var t = now(), dt = Math.min((t - prev) / 1000, 0.05); prev = t;

    if (alpha > CFG.alphaMin + 1e-4) layoutTick();      // keep settling while warm
    else if (frame % 4 === 0) layoutTick();             // gentle idle breathing

    if (mode === 'auto') autoStep(dt);
    else { flyStep(dt); if (autopilot && t - lastInput > CFG.idleMs) enterAuto(); }

    // Refresh the camera's world matrix NOW (render() does it too, but later):
    // picking and label projection below must use THIS frame's pose, not the
    // previous frame's, or the aim lags the view whenever the camera moves.
    camera.updateMatrixWorld();

    updateRender();
    if ((frame & 31) === 0) updateGraphRadius();
    updateOverlays();

    if (composer) composer.render(); else renderer.render(scene, camera);
    frame++;
  }

  // expose hooks: the Load button, plus programmatic loading (?src= / console)
  window.__graph3d = {
    load: function () { elFile.click(); },
    loadUrl: loadUrl,
    loadText: loadText,
    state: function () {
      var cx = 0, cy = 0, cz = 0, i;
      for (i = 0; i < N; i++) { cx += px[i]; cy += py[i]; cz += pz[i]; }
      cx /= N; cy /= N; cz /= N;
      // fraction of nodes currently in front of the camera (dot with view dir > 0)
      var f = new THREE.Vector3(); camera.getWorldDirection(f);
      var ahead = 0, v = new THREE.Vector3();
      for (i = 0; i < N; i++) { v.set(px[i] - camera.position.x, py[i] - camera.position.y, pz[i] - camera.position.z); if (v.dot(f) > 0) ahead++; }
      return {
        mode: mode, cam: [camera.position.x | 0, camera.position.y | 0, camera.position.z | 0],
        dir: [+f.x.toFixed(2), +f.y.toFixed(2), +f.z.toFixed(2)],
        camDist: camera.position.length() | 0, centroid: [cx | 0, cy | 0, cz | 0],
        graphRadius: graphRadius | 0, coreRadius: coreRadius | 0, viewRadius: viewRadius | 0,
        aheadPct: Math.round(100 * ahead / N)
      };
    },
    dbg: function () {
      return { lookMode: lookMode, autopilot: autopilot, mode: mode, cursor: [cursorX | 0, cursorY | 0], focusIdx: focusIdx, focusId: focusIdx >= 0 ? nodes[focusIdx].id : null };
    },
    // Debug: live spatial extent, recomputed from the CURRENT node positions
    // (unlike state().graphRadius, which is the value cached each frame by
    // updateGraphRadius). Max/mean distance from the centroid - handy for
    // checking that X-spread pushes every node out by the same fixed step.
    extent: function () {
      var cx = 0, cy = 0, cz = 0, i;
      for (i = 0; i < N; i++) { cx += px[i]; cy += py[i]; cz += pz[i]; }
      cx /= N; cy /= N; cz /= N;
      var mx = 0, sum = 0;
      for (i = 0; i < N; i++) {
        var dx = px[i] - cx, dy = py[i] - cy, dz = pz[i] - cz;
        var d = Math.sqrt(dx * dx + dy * dy + dz * dz);
        if (d > mx) mx = d;
        sum += d;
      }
      return { max: +mx.toFixed(2), mean: +(sum / N).toFixed(2), spreadDist: +spreadDist.toFixed(2) };
    },
    // Debug: pull the camera back to frame the whole graph (to inspect layout).
    overview: function () {
      var halfFov = THREE.MathUtils.degToRad(camera.fov * 0.5);
      var d = viewRadius / Math.sin(halfFov) * 1.15;
      camera.position.set(0, viewRadius * 0.3, d);
      camera.lookAt(0, 0, 0);
      mode = 'fly'; lastInput = now();
      return 'dist=' + (camera.position.length() | 0) + ' viewRadius=' + (viewRadius | 0);
    },
    // Debug: park the camera looking at the highest-degree hub so its focus
    // highlight (green outgoing / orange incoming edges) is easy to inspect.
    lookAtHub: function () {
      var best = 0, i;
      for (i = 1; i < N; i++) if (nodes[i].deg > nodes[best].deg) best = i;
      var target = new THREE.Vector3(px[best], py[best], pz[best]);
      var dir = new THREE.Vector3(1, 0.35, 1).normalize();
      camera.position.copy(target).addScaledVector(dir, 80);
      camera.lookAt(target);          // free-look: the quaternion is the state, no Euler needed
      mode = 'fly'; lastInput = now();
      return nodes[best].id + ' deg=' + nodes[best].deg;
    },
    // Deterministically step the autocam (bypasses rAF throttling) and report
    // dist-from-center + %nodes-ahead along the trajectory. Healthy flythrough:
    // starts ~100% ahead (outside), settles near ~50% (immersed), never spikes
    // dist far past coreRadius with ahead near 0 (that = flew out into the void).
    simulate: function (steps, dt) {
      dt = dt || 1 / 60;
      var out = [], every = Math.max(1, Math.ceil(steps / 14));
      var f = new THREE.Vector3(), v = new THREE.Vector3();
      for (var k = 0; k < steps; k++) {
        autoStep(dt);
        if (k % every === 0 || k === steps - 1) {
          camera.getWorldDirection(f);
          var ahead = 0;
          for (var i = 0; i < N; i++) { v.set(px[i] - camera.position.x, py[i] - camera.position.y, pz[i] - camera.position.z); if (v.dot(f) > 0) ahead++; }
          out.push('t=' + (k * dt).toFixed(1) + 's dist=' + (camera.position.length() | 0) + ' ahead=' + Math.round(100 * ahead / N) + '%');
        }
      }
      return out.join('\n');
    }
  };

  // ---- go -----------------------------------------------------------------
  buildGraph(sampleGraph());
  animate();

  var src = new URLSearchParams(location.search).get('src');
  if (src) loadUrl(src);
  else flash('Showing a sample graph - drop a .dot file (or click Load) to view your own');
})();
