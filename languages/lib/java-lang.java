// java-lang.java -- the part of java.lang both java grammars provide written in
// Java: java-interpreter.abnf and java-to-llvm-ir.abnf compile it with a program
// that creates one of its types (jLangWants in lib/java-decltypes.js), the way
// lib/java-util.java serves java.util. One implementation serves all three
// engines. The exception messages are java 24.0.2's.
package java.lang;

// The text is kept as appended pieces that are joined - pairwise, so a loop of
// n appends costs O(n log n) rather than O(n^2) - only when it is read.
final class StringBuilder {
    String text;
    String[] parts;
    int nparts;
    int count;

    public StringBuilder() { text = ""; parts = new String[8]; }
    public StringBuilder(int capacity) {
        if (capacity < 0) { throw new NegativeArraySizeException("" + capacity); }
        text = "";
        parts = new String[8];
    }
    public StringBuilder(String str) {
        text = str;
        parts = new String[8];
        count = str.length();
    }
    public StringBuilder(CharSequence seq) {
        text = seq.toString();
        parts = new String[8];
        count = text.length();
    }

    String flat() {
        if (nparts == 0) { return text; }
        int k = nparts;
        while (k > 1) {
            int j = 0;
            for (int i = 0; i < k; i += 2) {
                parts[j] = i + 1 < k ? parts[i] + parts[i + 1] : parts[i];
                j++;
            }
            k = j;
        }
        text = text + parts[0];
        for (int i = 0; i < nparts; i++) { parts[i] = null; }
        nparts = 0;
        return text;
    }
    void set(String s) {
        flat();
        text = s;
        count = s.length();
    }
    void checkIndex(int index) {
        if (index < 0 || index >= count) {
            throw new StringIndexOutOfBoundsException("Index " + index + " out of bounds for length " + count);
        }
    }
    void checkRange(int start, int end) {
        if (start < 0 || start > end || end > count) {
            throw new StringIndexOutOfBoundsException("Range [" + start + ", " + end + ") out of bounds for length " + count);
        }
    }

    public StringBuilder append(Object o) {
        String s = "" + o;
        if (nparts == parts.length) {
            String[] more = new String[parts.length * 2];
            for (int i = 0; i < nparts; i++) { more[i] = parts[i]; }
            parts = more;
        }
        parts[nparts] = s;
        nparts++;
        count += s.length();
        return this;
    }
    public StringBuilder append(CharSequence s, int start, int end) {
        String t = "" + s;
        return append(t.substring(start, end));
    }
    public StringBuilder insert(int offset, Object o) {
        checkRange(offset, count);
        String s = flat();
        set(s.substring(0, offset) + o + s.substring(offset));
        return this;
    }
    public int length() { return count; }
    public boolean isEmpty() { return count == 0; }
    public char charAt(int index) {
        checkIndex(index);
        return flat().charAt(index);
    }
    public String toString() { return flat(); }
    public String substring(int start) { return substring(start, count); }
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
        int e = end > count ? count : end;
        checkRange(start, e);
        String s = flat();
        set(s.substring(0, start) + s.substring(e));
        return this;
    }
    public StringBuilder replace(int start, int end, String str) {
        int e = end > count ? count : end;
        checkRange(start, e);
        String s = flat();
        set(s.substring(0, start) + str + s.substring(e));
        return this;
    }
    public void setLength(int n) {
        if (n < 0) { throw new StringIndexOutOfBoundsException("String index out of range: " + n); }
        String s = flat();
        if (n <= count) {
            set(s.substring(0, n));
        } else {
            char[] zeros = new char[n - count];
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
