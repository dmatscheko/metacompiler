<?php
// A helper library imported by tests/php-test-multifile.php with `require 'geomlib.php';`
// (found via the -i tests/imports include root). The file is parsed with the same PHP
// grammar; its top-level function and class register in the global scope, exactly as if
// they had been declared in the main file. Imported files start with the '<?php' open
// tag, like every file this grammar parses.

function geom_area($w, $h) {
    return $w * $h;
}

class Rect {
    public $w;
    public $h;
    public function __construct($w, $h) {
        $this->w = $w;
        $this->h = $h;
    }
    public function area() {
        return $this->w * $this->h;
    }
    public function scale($f) {
        return new Rect($this->w * $f, $this->h * $f);
    }
}
