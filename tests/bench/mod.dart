// See mod.kt. Same loop, so the languages are comparable to each other.
//
// dart had NO bench row until docs/todo.md 7.7, so nothing in this project ever
// loaded languages/lib/dart-rt.ll under a timer. dart imports
// lib/runtime.metajs and lib/runtime-dartswift.metajs, so a change to either
// shows up here as well as in the swift row.
void main() {
  int s = 0;
  int i = 0;
  while (i < 40000) {
    s = s + i % 7;
    i = i + 1;
  }
  print(s);
}
