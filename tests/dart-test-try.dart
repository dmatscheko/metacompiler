// Dart try/catch/finally/throw, genuinely executed (interpreter and compiler).
//
// throw raises a value that unwinds through calls to the nearest catch; catch binds it
// (the leftmost catch clause wins - exception types, including `on Type`, are parsed but
// not discriminated); finally always runs. A return/break/continue that leaves a try or
// catch body works in both engines. An uncaught throw is a clean runtime error.
//
// main() counts failed checks and returns that count, so the run exits 0 exactly when
// everything works; the interpreter and compiler must agree.

class Boom {
  int code;
  Boom(this.code);
}

int fails = 0;

void check(int got, int want) {
  if (got != want) { fails = fails + 1; }
}

// A throw raised from a nested call unwinds to the caller's catch.
int risky(int n) {
  if (n > 3) { throw Boom(n); }
  return n * 2;
}

// return out of a try, and out of a catch.
int classify(int n) {
  try {
    if (n > 0) { return n * 10; }
    throw Boom(0);
  } catch (e) {
    return -1;
  } finally { }
}

// A return out of an INNER try propagates through the OUTER try.
int nestedReturn() {
  try {
    try { return 9; } finally { }
  } finally { }
  return 0;
}

// break / continue leaving a try body inside a loop.
int loopBreak() {
  int sum = 0;
  for (int i = 0; i < 10; i = i + 1) {
    try { if (i == 3) { break; } sum = sum + i; } finally { }
  }
  return sum;              // 0+1+2 = 3
}

int loopContinue() {
  int sum = 0;
  for (int i = 0; i < 5; i = i + 1) {
    try { if (i == 2) { continue; } sum = sum + i; } catch (e) { }
  }
  return sum;              // 0+1+3+4 = 8
}

int main() {
  // basic try/catch/finally: the log records the order try -> catch -> finally.
  String log = "";
  try {
    log = log + "a";
    throw Boom(1);
  } catch (e) {
    log = log + "b";
  } finally {
    log = log + "c";
  }
  if (log != "abc") { fails = fails + 1; }

  // throw from a nested call, caught with a field read on the bound value.
  int caught = -1;
  try {
    risky(5);
    fails = fails + 1;      // not reached
  } catch (e) {
    caught = e.code;
  }
  check(caught, 5);

  // no-throw path: risky(2) returns normally, no catch runs.
  check(risky(2), 4);

  // `on Type catch (e)` form: the typed catch still binds and reads the value.
  int onCaught = -1;
  try {
    throw Boom(7);
  } on Boom catch (e) {
    onCaught = e.code;
  }
  check(onCaught, 7);

  check(classify(4), 40);    // return out of a try body
  check(classify(-1), -1);   // return out of a catch body
  check(nestedReturn(), 9);  // return through nested tries
  check(loopBreak(), 3);     // break out of a try body in a loop
  check(loopContinue(), 8);  // continue out of a try body in a loop

  if (fails == 0) { print("Dart try/catch OK"); }
  return fails;
}
