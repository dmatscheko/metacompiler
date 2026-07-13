// Multi-file Dart test: the class Vec and the top-level function addAll live in
// tests/imports/geomlib.dart, found via the -i include root (mec -i tests/imports ...).
// The imported file is parsed with the same grammar; its declarations register like the
// main file's own top level. 'dart:math' is a builtin no-op import, mixed in on purpose.
// int main() returns the failure count, so the exit code is 0 on success.
import 'dart:math';
import 'geomlib.dart';

int main() {
  int failures = 0;

  Vec a = Vec(3, 4);
  Vec b = Vec(2, -1);

  // imported instance method + imported-class field access
  if (a.dot(b) != 2) failures = failures + 1;
  if (a.x != 3) failures = failures + 1;
  if (b.y != -1) failures = failures + 1;

  // another imported method (locals, if, unary minus)
  if (a.manhattan(b) != 6) failures = failures + 1;

  // imported method using string interpolation
  if (a.describe() != 'Vec(3, 4)') failures = failures + 1;

  // imported top-level function
  if (addAll([1, 2, 3, 4, 5]) != 15) failures = failures + 1;
  if (addAll([10, 20, 30]) != 60) failures = failures + 1;

  if (failures == 0) {
    print('dart multifile test passed');
  }
  return failures;
}
