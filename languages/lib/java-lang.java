// java-lang.java -- the part of java.lang both java grammars provide written in
// Java: java-interpreter.abnf and java-to-llvm-ir.abnf compile it with a program
// that creates one of its types (jLangWants in lib/java-decltypes.js), the way
// lib/java-util.java serves java.util. One implementation serves all three
// engines. The exception messages are java 24.0.2's.
package java.lang;

// The text is `text`, then the levels of a binary counter, then `tail`. An
// append concatenates onto tail, which stays short: once it holds 64 chars it
// is carried into the counter, where parts[k] holds 2^k carried tails joined,
// present exactly when bit k of ncarried is set, a higher level holding older
// text. A carry ripples through the levels the way an increment does through
// the bits, so over n appends a character is copied at most 64 times in the
// tail and O(log n) times in the counter. Few statements per append matter:
// the interpreter runs this file statement by statement. A level
// whose bit is clear may hold a stale piece that is never read. before is the
// length of everything ahead of tail.
final class StringBuilder {
    String text;
    String tail;
    String[] parts;
    int ncarried;
    int before;

    public StringBuilder() {
        parts = new String[32];
        set("");
    }
    public StringBuilder(int capacity) {
        if (capacity < 0) { throw new NegativeArraySizeException("" + capacity); }
        parts = new String[32];
        set("");
    }
    public StringBuilder(String str) {
        parts = new String[32];
        set(str);
    }
    public StringBuilder(CharSequence seq) {
        parts = new String[32];
        set(seq.toString());
    }

    void carry() {
        String s = tail;
        tail = "";
        before = before + s.length();
        int n = ncarried;
        ncarried = n + 1;
        int k = 0;
        while ((n & 1) != 0) {
            s = parts[k] + s;
            k++;
            n = n >> 1;
        }
        parts[k] = s;
    }
    String flat() {
        if (ncarried == 0 && tail.length() == 0) { return text; }
        String r = tail;
        int k = 0;
        for (int n = ncarried; n != 0; n = n >> 1) {
            if ((n & 1) != 0) { r = parts[k] + r; }
            k++;
        }
        set(text + r);
        return text;
    }
    void set(String s) {
        text = s;
        tail = "";
        ncarried = 0;
        before = s.length();
    }
    void checkIndex(int index) {
        int count = length();
        if (index < 0 || index >= count) {
            throw new StringIndexOutOfBoundsException("Index " + index + " out of bounds for length " + count);
        }
    }
    void checkRange(int start, int end) {
        int count = length();
        if (start < 0 || start > end || end > count) {
            throw new StringIndexOutOfBoundsException("Range [" + start + ", " + end + ") out of bounds for length " + count);
        }
    }

    public StringBuilder append(Object o) {
        tail = tail + o;
        if (tail.length() >= 64) { carry(); }
        return this;
    }
    public StringBuilder append(CharSequence s, int start, int end) {
        String t = "" + s;
        return append(t.substring(start, end));
    }
    public StringBuilder insert(int offset, Object o) {
        checkRange(offset, length());
        String s = flat();
        set(s.substring(0, offset) + o + s.substring(offset));
        return this;
    }
    public int length() { return before + tail.length(); }
    public boolean isEmpty() { return length() == 0; }
    public char charAt(int index) {
        checkIndex(index);
        return flat().charAt(index);
    }
    public String toString() { return flat(); }
    public String substring(int start) { return substring(start, length()); }
    public String substring(int start, int end) {
        checkRange(start, end);
        return flat().substring(start, end);
    }
    public int indexOf(String str) { return flat().indexOf(str); }
    public int indexOf(String str, int from) { return flat().indexOf(str, from); }
    public int lastIndexOf(String str) { return flat().lastIndexOf(str); }
    public int lastIndexOf(String str, int from) { return flat().lastIndexOf(str, from); }
    public int compareTo(StringBuilder other) { return flat().compareTo(other.toString()); }
    public void setCharAt(int index, char c) {
        checkIndex(index);
        String s = flat();
        set(s.substring(0, index) + c + s.substring(index + 1));
    }
    public StringBuilder deleteCharAt(int index) {
        checkIndex(index);
        String s = flat();
        set(s.substring(0, index) + s.substring(index + 1));
        return this;
    }
    public StringBuilder delete(int start, int end) {
        int count = length();
        int e = end > count ? count : end;
        checkRange(start, e);
        String s = flat();
        set(s.substring(0, start) + s.substring(e));
        return this;
    }
    public StringBuilder replace(int start, int end, String str) {
        int count = length();
        int e = end > count ? count : end;
        checkRange(start, e);
        String s = flat();
        set(s.substring(0, start) + str + s.substring(e));
        return this;
    }
    public void setLength(int n) {
        if (n < 0) { throw new StringIndexOutOfBoundsException("String index out of range: " + n); }
        String s = flat();
        if (n <= s.length()) {
            set(s.substring(0, n));
        } else {
            char[] zeros = new char[n - s.length()];
            set(s + new String(zeros));
        }
    }
    // A surrogate pair keeps its order, as java's reverse() does.
    public StringBuilder reverse() {
        char[] cs = flat().toCharArray();
        int n = cs.length;
        for (int i = 0; i < n / 2; i++) {
            char t = cs[i];
            cs[i] = cs[n - 1 - i];
            cs[n - 1 - i] = t;
        }
        for (int i = 0; i + 1 < n; i++) {
            int lo = cs[i];
            int hi = cs[i + 1];
            if (lo >= 0xDC00 && lo <= 0xDFFF && hi >= 0xD800 && hi <= 0xDBFF) {
                char t = cs[i];
                cs[i] = cs[i + 1];
                cs[i + 1] = t;
                i++;
            }
        }
        set(new String(cs));
        return this;
    }
}
