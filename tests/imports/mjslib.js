/* A project module imported by tests/metajs-test-multifile.js (via -i tests/imports).
   Uses only the restricted MetaJS subset (no class): plain functions + object literals.
   Its top-level functions register in the shared global scope when the file is loaded. */

function makeVec(x, y) { return { x: x, y: y }; }

function vecDot(a, b) { return a.x * b.x + a.y * b.y; }

function vecScaleX(v, f) { return v.x * f; }
