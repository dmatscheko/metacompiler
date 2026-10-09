// See mod.kt.
void main() {
  int s = 0;
  int i = 0;
  while (i < 40000) {
    s = s + i % 7;
    i = i + 1;
  }
  print(s);
}
