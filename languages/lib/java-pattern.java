// java-pattern.java -- java.util.regex's Pattern, Matcher and MatchResult,
// written in Java and compiled with the program when it imports one of them
// (jPatternWants in lib/java-decltypes.js). One implementation serves all three
// engines: the matching is the shared regex engine's J dialect, reached through
// String primitives of the pattern's text (__rxfind, __rxcount, __rxname,
// __rxexpand, __rxflags; lib/java-strings.js and abnf/jsrtjavastr.go), which
// answer a match's group bounds as an int[] and throw PatternSyntaxException.
// Not modelled: hitEnd/requireEnd, and transparent or non-anchoring bounds.
package java.util.regex;

interface MatchResult {
    int start();
    int start(int group);
    int end();
    int end(int group);
    String group();
    String group(int group);
    int groupCount();
}

final class Pattern {
    public static final int UNIX_LINES = 1;
    public static final int CASE_INSENSITIVE = 2;
    public static final int COMMENTS = 4;
    public static final int MULTILINE = 8;
    public static final int LITERAL = 16;
    public static final int DOTALL = 32;
    public static final int UNICODE_CASE = 64;
    public static final int CANON_EQ = 128;
    public static final int UNICODE_CHARACTER_CLASS = 256;

    final String source;
    final int given;
    // The text the engine compiles: LITERAL quotes it, UNICODE_CHARACTER_CLASS
    // is the inline (?U).
    final String text;
    final int groups;

    private Pattern(String regex, int flags) {
        source = regex;
        given = flags;
        String t = (flags & LITERAL) != 0 ? quote(regex) : regex;
        if ((flags & UNICODE_CHARACTER_CLASS) != 0) { t = "(?U)" + t; }
        text = t;
        groups = text.__rxcount(flags);
    }
    public static Pattern compile(String regex) { return new Pattern(regex, 0); }
    public static Pattern compile(String regex, int flags) { return new Pattern(regex, flags); }
    public static boolean matches(String regex, CharSequence input) { return compile(regex).matcher(input).matches(); }
    public static String quote(String s) {
        int e = s.indexOf("\\E");
        if (e < 0) { return "\\Q" + s + "\\E"; }
        StringBuilder sb = new StringBuilder("\\Q");
        int at = 0;
        while (e >= 0) {
            sb.append(s, at, e).append("\\E\\\\E\\Q");
            at = e + 2;
            e = s.indexOf("\\E", at);
        }
        return sb.append(s.substring(at)).append("\\E").toString();
    }
    public String pattern() { return source; }
    public String toString() { return source; }
    public int flags() { return text.__rxflags(given); }
    public Matcher matcher(CharSequence input) { return new Matcher(this, input.toString()); }
    public String[] split(CharSequence input) { return split(input, 0); }
    // Pattern.split: a positive limit caps the pieces, zero drops the trailing
    // empty ones, and an empty match at 0 makes no leading empty piece.
    public String[] split(CharSequence input, int limit) {
        String s = input.toString();
        java.util.ArrayList<String> out = new java.util.ArrayList<>();
        Matcher m = matcher(s);
        int index = 0;
        boolean any = false;
        while (m.find()) {
            if (limit > 0 && out.size() >= limit - 1) { break; }
            if (index == 0 && m.start() == 0 && m.end() == 0) { continue; }
            out.add(s.substring(index, m.start()));
            index = m.end();
            any = true;
        }
        if (!any && index == 0) { return new String[]{s}; }
        out.add(s.substring(index));
        int n = out.size();
        if (limit == 0) {
            while (n > 0 && out.get(n - 1).isEmpty()) { n--; }
        }
        String[] r = new String[n];
        for (int i = 0; i < n; i++) { r[i] = out.get(i); }
        return r;
    }
    public java.util.function.Predicate<String> asPredicate() {
        Pattern self = this;
        java.util.function.Predicate<String> p = s -> self.matcher(s).find();
        return p;
    }
    public java.util.function.Predicate<String> asMatchPredicate() {
        Pattern self = this;
        java.util.function.Predicate<String> p = s -> self.matcher(s).matches();
        return p;
    }
}

final class Matcher implements MatchResult {
    Pattern pat;
    String text;
    // The last match's group bounds (2 per group, -1 for one that took no
    // part), or null when there is none.
    int[] caps;
    // Where the next find() searches from, and where the previous match ended
    // (\G and appendReplacement's position).
    int next;
    int last;
    int appendPos;
    // The region [from, to) the matching sees, with java's default opaque
    // and anchoring bounds: the engine is handed that part of the text, so a
    // lookaround cannot see past it and ^ and $ match at its ends.
    int from;
    int to;

    Matcher(Pattern pat, String text) {
        this.pat = pat;
        this.text = text;
        to = text.length();
    }
    public Pattern pattern() { return pat; }
    public Matcher reset() {
        caps = null;
        next = 0;
        last = 0;
        appendPos = 0;
        from = 0;
        to = text.length();
        return this;
    }
    public Matcher reset(CharSequence input) {
        text = input.toString();
        return reset();
    }
    public Matcher region(int start, int end) {
        if (start < 0 || start > text.length()) { throw new IndexOutOfBoundsException("start"); }
        if (end < 0 || end > text.length()) { throw new IndexOutOfBoundsException("end"); }
        if (start > end) { throw new IndexOutOfBoundsException("start > end"); }
        reset();
        from = start;
        to = end;
        return this;
    }
    public int regionStart() { return from; }
    public int regionEnd() { return to; }
    public boolean hasTransparentBounds() { return false; }
    public boolean hasAnchoringBounds() { return true; }
    // A new pattern from the same position on; the last match's groups are lost.
    public Matcher usePattern(Pattern newPattern) {
        if (newPattern == null) { throw new IllegalArgumentException("Pattern cannot be null"); }
        pat = newPattern;
        caps = null;
        return this;
    }
    // The engine over the region: positions in and out are the text's.
    int[] run(int start, int lastEnd, int mode) {
        if (from == 0 && to == text.length()) { return pat.text.__rxfind(pat.given, text, start, lastEnd, mode); }
        int[] c = pat.text.__rxfind(pat.given, text.substring(from, to), start - from, lastEnd - from, mode);
        if (c == null) { return null; }
        for (int i = 0; i < c.length; i++) {
            if (c[i] >= 0) { c[i] = c[i] + from; }
        }
        return c;
    }
    public boolean matches() {
        caps = run(from, from, 1);
        if (caps != null) { next = caps[1] == caps[0] ? caps[1] + 1 : caps[1]; last = caps[1]; }
        return caps != null;
    }
    public boolean lookingAt() {
        caps = run(from, from, 2);
        if (caps != null) { next = caps[1] == caps[0] ? caps[1] + 1 : caps[1]; last = caps[1]; }
        return caps != null;
    }
    public boolean find() {
        int start = next < from ? from : next;
        if (start > to) {
            caps = null;
            return false;
        }
        caps = run(start, last < from ? from : last, 0);
        if (caps == null) {
            next = to + 1;
            return false;
        }
        next = caps[1] == caps[0] ? caps[1] + 1 : caps[1];
        last = caps[1];
        return true;
    }
    // Every further match, as find() would answer them.
    public java.util.stream.Stream<MatchResult> results() {
        java.util.ArrayList<MatchResult> out = new java.util.ArrayList<>();
        while (find()) { out.add(toMatchResult()); }
        return Stream.ofCollection(out);
    }
    public boolean find(int start) {
        if (start < 0 || start > text.length()) { throw new IndexOutOfBoundsException("Illegal start index"); }
        reset();
        next = start;
        last = start;
        return find();
    }
    void check(int group) {
        if (caps == null) { throw new IllegalStateException("No match found"); }
        if (group < 0 || group > pat.groups) { throw new IndexOutOfBoundsException("No group " + group); }
    }
    int named(String name) {
        int g = pat.text.__rxname(pat.given, name);
        if (g < 0) { throw new IllegalArgumentException("No group with name <" + name + ">"); }
        return g;
    }
    public int start() { return start(0); }
    public int start(int group) {
        check(group);
        return caps[2 * group];
    }
    public int start(String name) { return start(named(name)); }
    public int end() { return end(0); }
    public int end(int group) {
        check(group);
        return caps[2 * group + 1];
    }
    public int end(String name) { return end(named(name)); }
    public String group() { return group(0); }
    public String group(int group) {
        check(group);
        int b = caps[2 * group];
        return b < 0 ? null : text.substring(b, caps[2 * group + 1]);
    }
    public String group(String name) {
        if (caps == null) { throw new IllegalStateException("No match found"); }
        return group(named(name));
    }
    public int groupCount() { return pat.groups; }
    public MatchResult toMatchResult() {
        Matcher r = new Matcher(pat, text);
        r.caps = caps;
        return r;
    }
    public Matcher appendReplacement(StringBuilder sb, String replacement) {
        if (caps == null) { throw new IllegalStateException("No match available"); }
        sb.append(text, appendPos, caps[0]).append(pat.text.__rxexpand(pat.given, text, caps, replacement));
        appendPos = caps[1];
        return this;
    }
    public StringBuilder appendTail(StringBuilder sb) { return sb.append(text.substring(appendPos)); }
    public String replaceAll(String replacement) {
        reset();
        StringBuilder sb = new StringBuilder();
        while (find()) { appendReplacement(sb, replacement); }
        return appendTail(sb).toString();
    }
    public String replaceAll(java.util.function.Function<MatchResult, String> f) {
        reset();
        StringBuilder sb = new StringBuilder();
        while (find()) {
            sb.append(text, appendPos, caps[0]).append(f.apply(toMatchResult()));
            appendPos = caps[1];
        }
        return appendTail(sb).toString();
    }
    public String replaceFirst(String replacement) {
        reset();
        StringBuilder sb = new StringBuilder();
        if (find()) { appendReplacement(sb, replacement); }
        return appendTail(sb).toString();
    }
    public static String quoteReplacement(String s) {
        if (s.indexOf('\\') < 0 && s.indexOf('$') < 0) { return s; }
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            if (c == '\\' || c == '$') { sb.append('\\'); }
            sb.append(c);
        }
        return sb.toString();
    }
    public String toString() {
        return "java.util.regex.Matcher[pattern=" + pat.source + " region=" + from + "," + to + " lastmatch="
            + (caps == null ? "" : text.substring(caps[0], caps[1])) + "]";
    }
}
