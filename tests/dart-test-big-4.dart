// Dart self-checking test 4/4 for the metacompiler: STRING + NUMBER PROCESSING.
//
// A recursive-descent arithmetic evaluator held in a class with mutable parse state, Roman
// numeral conversion both ways, arbitrary base conversion, a Caesar cipher and case folding
// built purely from charAt / indexOf over alphabet strings, plus classic string utilities
// (reverse, palindrome, vowel count, hand-written word split, run-length encode/decode). Many
// checks are round trips (decode(encode(x)) == x) so they verify themselves. int main()
// returns the number of failures, so the exit code is 0 on success and goja and frozen must
// agree byte-for-byte.

// Top-level alphabet tables shared by the letter routines.
String lowerAlpha = 'abcdefghijklmnopqrstuvwxyz';
String upperAlpha = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
String digitChars = '0123456789';
String baseChars = '0123456789ABCDEF';

int digitOf(String ch) => digitChars.indexOf(ch);

// ---------- recursive-descent expression evaluator ----------
// grammar:  expr := term {('+'|'-') term} ; term := factor {('*'|'/') factor} ;
//           factor := number | '(' expr ')'
class Evaluator {
  String src;
  int pos;
  Evaluator(this.src) {
    pos = 0;
  }
  bool atEnd() => pos >= src.length;
  String peek() => this.atEnd() ? '' : src.charAt(pos);
  void advance() {
    pos = pos + 1;
  }

  int number() {
    int val = 0;
    while (!this.atEnd()) {
      int d = digitOf(this.peek());
      if (d < 0) break;
      val = val * 10 + d;
      this.advance();
    }
    return val;
  }

  int factor() {
    if (this.peek() == '(') {
      this.advance();
      int v = this.expr();
      this.advance(); // consume ')'
      return v;
    }
    return this.number();
  }

  int term() {
    int v = this.factor();
    while (!this.atEnd()) {
      String op = this.peek();
      if (op == '*') {
        this.advance();
        v = v * this.factor();
      } else if (op == '/') {
        this.advance();
        v = v ~/ this.factor();
      } else {
        break;
      }
    }
    return v;
  }

  int expr() {
    int v = this.term();
    while (!this.atEnd()) {
      String op = this.peek();
      if (op == '+') {
        this.advance();
        v = v + this.term();
      } else if (op == '-') {
        this.advance();
        v = v - this.term();
      } else {
        break;
      }
    }
    return v;
  }
}

int evalExpr(String s) {
  Evaluator e = Evaluator(s);
  return e.expr();
}

// ---------- Roman numerals ----------

String intToRoman(int n) {
  List<int> vals = [1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1];
  List<String> syms = ['M', 'CM', 'D', 'CD', 'C', 'XC', 'L', 'XL', 'X', 'IX', 'V', 'IV', 'I'];
  String out = '';
  for (int i = 0; i < vals.length; i = i + 1) {
    while (n >= vals[i]) {
      out = out + syms[i];
      n = n - vals[i];
    }
  }
  return out;
}

int romanCharVal(String c) {
  if (c == 'I') return 1;
  if (c == 'V') return 5;
  if (c == 'X') return 10;
  if (c == 'L') return 50;
  if (c == 'C') return 100;
  if (c == 'D') return 500;
  if (c == 'M') return 1000;
  return 0;
}

int romanToInt(String s) {
  int total = 0;
  for (int i = 0; i < s.length; i = i + 1) {
    int cur = romanCharVal(s.charAt(i));
    int nxt = (i + 1 < s.length) ? romanCharVal(s.charAt(i + 1)) : 0;
    if (cur < nxt) {
      total = total - cur;
    } else {
      total = total + cur;
    }
  }
  return total;
}

// ---------- base conversion ----------

String toBase(int n, int base) {
  if (n == 0) return '0';
  String s = '';
  while (n > 0) {
    s = baseChars.charAt(n % base) + s;
    n = n ~/ base;
  }
  return s;
}

int fromBase(String s, int base) {
  int v = 0;
  for (int i = 0; i < s.length; i = i + 1) {
    v = v * base + baseChars.indexOf(s.charAt(i));
  }
  return v;
}

// ---------- letters: Caesar cipher and case folding (charAt / indexOf only) ----------

String caesar(String s, int shift) {
  int k = shift % 26;
  String out = '';
  for (int i = 0; i < s.length; i = i + 1) {
    String ch = s.charAt(i);
    int p = lowerAlpha.indexOf(ch);
    if (p < 0) {
      out = out + ch;
    } else {
      out = out + lowerAlpha.charAt((p + k) % 26);
    }
  }
  return out;
}

String caesarDecode(String s, int shift) => caesar(s, (26 - shift % 26) % 26);

String toUpper(String s) {
  String out = '';
  for (int i = 0; i < s.length; i = i + 1) {
    String ch = s.charAt(i);
    int p = lowerAlpha.indexOf(ch);
    out = out + (p >= 0 ? upperAlpha.charAt(p) : ch);
  }
  return out;
}

String toLower(String s) {
  String out = '';
  for (int i = 0; i < s.length; i = i + 1) {
    String ch = s.charAt(i);
    int p = upperAlpha.indexOf(ch);
    out = out + (p >= 0 ? lowerAlpha.charAt(p) : ch);
  }
  return out;
}

// ---------- string utilities ----------

String reverseStr(String s) {
  String out = '';
  for (int i = s.length - 1; i >= 0; i = i - 1) {
    out = out + s.charAt(i);
  }
  return out;
}

bool isPalindrome(String s) => s == reverseStr(s);

int countVowels(String s) {
  int c = 0;
  for (int i = 0; i < s.length; i = i + 1) {
    if ('aeiou'.indexOf(s.charAt(i)) >= 0) c = c + 1;
  }
  return c;
}

List<String> splitWords(String s, String sep) {
  List<String> out = [];
  String cur = '';
  for (int i = 0; i < s.length; i = i + 1) {
    String ch = s.charAt(i);
    if (ch == sep) {
      out.add(cur);
      cur = '';
    } else {
      cur = cur + ch;
    }
  }
  out.add(cur);
  return out;
}

String rleEncode(String s) {
  String out = '';
  int i = 0;
  while (i < s.length) {
    String ch = s.charAt(i);
    int count = 1;
    while (i + count < s.length && s.charAt(i + count) == ch) {
      count = count + 1;
    }
    out = out + ch + '$count';
    i = i + count;
  }
  return out;
}

String rleDecode(String s) {
  String out = '';
  int i = 0;
  while (i < s.length) {
    String ch = s.charAt(i);
    i = i + 1;
    int count = 0;
    while (i < s.length && digitOf(s.charAt(i)) >= 0) {
      count = count * 10 + digitOf(s.charAt(i));
      i = i + 1;
    }
    for (int k = 0; k < count; k = k + 1) {
      out = out + ch;
    }
  }
  return out;
}

int main() {
  int failures = 0;

  // ---------- expression evaluator ----------
  if (evalExpr('2+3*4') != 14) failures = failures + 1;
  if (evalExpr('(2+3)*4') != 20) failures = failures + 1;
  if (evalExpr('100-40-10') != 50) failures = failures + 1;
  if (evalExpr('2*3+4*5') != 26) failures = failures + 1;
  if (evalExpr('((1+2)*(3+4))') != 21) failures = failures + 1;
  if (evalExpr('20/3') != 6) failures = failures + 1;   // integer division
  if (evalExpr('1000') != 1000) failures = failures + 1;
  if (evalExpr('2*(3+4)*(5-1)') != 56) failures = failures + 1;
  if (evalExpr('7') != 7) failures = failures + 1;
  // the evaluator agrees with directly computed arithmetic
  if (evalExpr('12+34*2') != 12 + 34 * 2) failures = failures + 1;

  // ---------- Roman numerals: known values and a round trip ----------
  if (intToRoman(4) != 'IV') failures = failures + 1;
  if (intToRoman(9) != 'IX') failures = failures + 1;
  if (intToRoman(40) != 'XL') failures = failures + 1;
  if (intToRoman(1994) != 'MCMXCIV') failures = failures + 1;
  if (intToRoman(2023) != 'MMXXIII') failures = failures + 1;
  if (romanToInt('XIV') != 14) failures = failures + 1;
  if (romanToInt('MCMXCIV') != 1994) failures = failures + 1;
  for (int n = 1; n <= 100; n = n + 1) {
    if (romanToInt(intToRoman(n)) != n) failures = failures + 1;
  }
  if (romanToInt(intToRoman(3888)) != 3888) failures = failures + 1;

  // ---------- base conversion: known values and round trips ----------
  if (toBase(10, 2) != '1010') failures = failures + 1;
  if (toBase(255, 16) != 'FF') failures = failures + 1;
  if (toBase(0, 2) != '0') failures = failures + 1;
  if (toBase(8, 8) != '10') failures = failures + 1;
  if (fromBase('1010', 2) != 10) failures = failures + 1;
  if (fromBase('FF', 16) != 255) failures = failures + 1;
  if (fromBase('777', 8) != 511) failures = failures + 1;
  for (int n = 0; n <= 300; n = n + 7) {
    if (fromBase(toBase(n, 2), 2) != n) failures = failures + 1;
    if (fromBase(toBase(n, 16), 16) != n) failures = failures + 1;
    if (fromBase(toBase(n, 8), 8) != n) failures = failures + 1;
  }

  // ---------- Caesar cipher: shift and round trip ----------
  if (caesar('abc', 1) != 'bcd') failures = failures + 1;
  if (caesar('xyz', 3) != 'abc') failures = failures + 1; // wraps around
  if (caesar('hello world', 5) != 'mjqqt btwqi') failures = failures + 1; // space preserved
  String secret = caesar('themetacompiler', 13);
  if (caesarDecode(secret, 13) != 'themetacompiler') failures = failures + 1;
  for (int sh = 0; sh < 26; sh = sh + 1) {
    if (caesarDecode(caesar('dartlang', sh), sh) != 'dartlang') failures = failures + 1;
  }

  // ---------- case folding ----------
  if (toUpper('hello') != 'HELLO') failures = failures + 1;
  if (toLower('WORLD') != 'world') failures = failures + 1;
  if (toUpper('MixEd123') != 'MIXED123') failures = failures + 1; // digits untouched
  if (toLower(toUpper('roundtrip')) != 'roundtrip') failures = failures + 1;

  // ---------- string utilities ----------
  if (reverseStr('abcde') != 'edcba') failures = failures + 1;
  if (reverseStr('') != '') failures = failures + 1;
  if (!isPalindrome('racecar')) failures = failures + 1;
  if (!isPalindrome('level')) failures = failures + 1;
  if (isPalindrome('dart')) failures = failures + 1;
  if (countVowels('metacompiler') != 5) failures = failures + 1; // e a o i e
  if (countVowels('rhythm') != 0) failures = failures + 1;

  List<String> words = splitWords('the quick brown fox', ' ');
  if (words.length != 4) failures = failures + 1;
  if (words[0] != 'the') failures = failures + 1;
  if (words[3] != 'fox') failures = failures + 1;
  // total length of all words plus the 3 separators equals the source length
  int lenSum = 0;
  for (int i = 0; i < words.length; i = i + 1) {
    lenSum = lenSum + words[i].length;
  }
  if (lenSum + 3 != 19) failures = failures + 1;

  // ---------- run-length encoding round trip ----------
  if (rleEncode('aaabbc') != 'a3b2c1') failures = failures + 1;
  if (rleEncode('wwwwwwwwwwww') != 'w12') failures = failures + 1; // multi-digit count
  if (rleDecode('a3b2c1') != 'aaabbc') failures = failures + 1;
  if (rleDecode(rleEncode('aaabbc')) != 'aaabbc') failures = failures + 1;
  if (rleDecode(rleEncode('mississippi')) != 'mississippi') failures = failures + 1;
  if (rleDecode(rleEncode('abcdef')) != 'abcdef') failures = failures + 1;

  // ---------- combine several utilities ----------
  // encrypt, reverse, un-reverse, decrypt gets the original back
  String round = caesarDecode(reverseStr(reverseStr(caesar('portable', 7))), 7);
  if (round != 'portable') failures = failures + 1;

  print('dart-test-big-4 (string + number processing) finished with $failures failures');
  return failures;
}
