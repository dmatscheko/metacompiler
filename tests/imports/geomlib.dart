// A small geometry library imported by tests/dart-test-multifile.dart (via -i tests/imports).
// It is parsed with the Dart grammar; its class Vec and top-level function addAll register
// like the main file's own top-level declarations. There is no main() here - only the main
// file's main() is the entry point, so importing this file just registers its declarations.

class Vec {
  int x;
  int y;
  Vec(this.x, this.y);
  int dot(Vec o) {
    return this.x * o.x + this.y * o.y;
  }
  int manhattan(Vec o) {
    int dx = this.x - o.x;
    int dy = this.y - o.y;
    if (dx < 0) dx = -dx;
    if (dy < 0) dy = -dy;
    return dx + dy;
  }
  String describe() => 'Vec(${this.x}, ${this.y})';
}

int addAll(List<int> xs) {
  int total = 0;
  for (int v in xs) {
    total = total + v;
  }
  return total;
}
