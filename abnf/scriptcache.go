package abnf

// The frozen-mode script cache: compiled annotation-script modules, kept on
// disk across runs.
//
// A -frozen run must compile every distinct annotation script of its grammars
// with the frozen MetaJS compiler before anything can execute - for a
// tag-heavy grammar that is 1.5-2s of identical work on every run. The
// compile is deterministic and its only inputs are the script source and the
// bootstrap snapshot (jsbootstrap.ll + jsagrammar.go, regenerated together by
// -freeze). So compileScript stores each emitted module as .ll text, keyed by
// a hash of snapshot+source, and later runs reload it with asm.ParseString -
// the same print/parse round trip the snapshot itself goes through at -freeze
// time. Cache entries never go stale silently: a refreeze changes the
// snapshot hash and with it every key.
//
// Location: $MEC_SCRIPT_CACHE if set (the value "off" disables caching),
// otherwise <user cache dir>/mec/scripts. Unreadable or corrupt entries are
// recompiled and overwritten. Writes are atomic (temp file + rename), so
// parallel mec processes can share the cache safely.

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/llir/llvm/asm"
	"github.com/llir/llvm/ir"
)

// scriptCacheFormat invalidates the whole cache when the Go side changes what
// compileScript emits for the same source (e.g. a compile-walk change in
// compiler.go). Bump it in that case.
const scriptCacheFormat = "1"

var scriptCacheDir string  // "" = disabled; set by scriptCacheInit.
var scriptCacheKey string  // Hash over format+snapshot, mixed into every entry name.
var scriptCacheReady bool

func scriptCacheInit() {
	if scriptCacheReady {
		return
	}
	scriptCacheReady = true
	dir := os.Getenv("MEC_SCRIPT_CACHE")
	switch dir {
	case "off":
		return
	case "":
		base, err := os.UserCacheDir()
		if err != nil {
			return
		}
		dir = filepath.Join(base, "mec", "scripts")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	sum := sha256.Sum256([]byte(scriptCacheFormat + "\x00" + jsBootstrapLL))
	scriptCacheDir = dir
	scriptCacheKey = hex.EncodeToString(sum[:])
}

// scriptCachePath maps a script source to its cache file ("" when disabled).
func scriptCachePath(code string) string {
	scriptCacheInit()
	if scriptCacheDir == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(scriptCacheKey + "\x00" + code))
	return filepath.Join(scriptCacheDir, hex.EncodeToString(sum[:])+".ll")
}

// loadCachedScript returns the cached module of a script source, or nil.
func loadCachedScript(code string) *ir.Module {
	path := scriptCachePath(code)
	if path == "" {
		return nil
	}
	dat, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	mod, err := asm.ParseString(path, string(dat))
	if err != nil {
		return nil // Corrupt entry: compileScript recompiles and overwrites it.
	}
	return mod
}

// storeCachedScript writes the compiled module of a script source, atomically.
// Caching is best effort: any failure (including a module whose printing
// panics) just skips the store - the run itself must not be affected.
func storeCachedScript(code string, mod *ir.Module) {
	defer func() { recover() }()
	path := scriptCachePath(code)
	if path == "" {
		return
	}
	text := mod.String()
	tmp, err := os.CreateTemp(filepath.Dir(path), ".mec-script-*")
	if err != nil {
		return
	}
	_, werr := tmp.WriteString(text)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		os.Remove(tmp.Name())
		return
	}
	if os.Rename(tmp.Name(), path) != nil {
		os.Remove(tmp.Name())
	}
}
