# Python subset self test - BIG 4: string and text processing.
#
# Theme: text algorithms built from the primitive string operations the subset gives
# (indexing, slicing, len, membership, concatenation, comparison) - there are no
# string methods and no ord/chr/str/int builtins, so case conversion, tokenizing,
# number parsing and formatting are all done by hand. Covers case folding, reversal
# and palindromes, run-length encode/decode, a Caesar cipher, word frequency
# counting, manual split/join, atoi/itoa, substring counting and anagrams. The file
# runs top to bottom and ends with exit(fails[0]); the interpreter and the LLVM-IR
# compiler (both engines) must agree byte for byte.

fails = [0]


def check(name, got, want):
    if got != want:
        print("FAIL", name, "got", got, "want", want)
        fails[0] += 1


def check_true(name, got):
    if not got:
        print("FAIL", name, "expected a true value")
        fails[0] += 1


def render(a):
    return f"{a}"


# ----- character tables and primitives -----

LOWER = "abcdefghijklmnopqrstuvwxyz"
UPPER = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
DIGITS = "0123456789"
VOWELS = "aeiou"


def idx_in(s, ch):
    for i in range(len(s)):
        if s[i] == ch:
            return i
    return -1


def is_digit(ch):
    return idx_in(DIGITS, ch) >= 0


def to_upper(s):
    out = ""
    for ch in s:
        i = idx_in(LOWER, ch)
        if i >= 0:
            out += UPPER[i]
        else:
            out += ch
    return out


def to_lower(s):
    out = ""
    for ch in s:
        i = idx_in(UPPER, ch)
        if i >= 0:
            out += LOWER[i]
        else:
            out += ch
    return out


check("upper hello", to_upper("hello"), "HELLO")
check("upper mixed", to_upper("aB3z"), "AB3Z")
check("lower shout", to_lower("HELLO World"), "hello world")
check("case round-trip", to_lower(to_upper("MixedCase123")), "mixedcase123")


# ----- reversal and palindromes -----

def reverse_str(s):
    out = ""
    for i in range(len(s)):
        out = s[i] + out
    return out


def is_palindrome(s):
    return s == reverse_str(s)


check("reverse abc", reverse_str("abc"), "cba")
check("reverse empty", reverse_str(""), "")
check_true("racecar palindrome", is_palindrome("racecar"))
check_true("single palindrome", is_palindrome("x"))
check("hello not palindrome", is_palindrome("hello"), False)
pals = [w for w in ["level", "world", "noon", "test", "kayak"] if is_palindrome(w)]
check("palindrome filter", render(pals), "['level', 'noon', 'kayak']")


# ----- run-length encoding and decoding -----

def rle_encode(s):
    if len(s) == 0:
        return ""
    out = ""
    prev = s[0]
    count = 1
    for i in range(1, len(s)):
        if s[i] == prev:
            count += 1
        else:
            out += prev + f"{count}"
            prev = s[i]
            count = 1
    out += prev + f"{count}"
    return out


def rle_decode(s):
    out = ""
    i = 0
    while i < len(s):
        ch = s[i]
        i += 1
        num = 0
        while i < len(s) and is_digit(s[i]):
            num = num * 10 + idx_in(DIGITS, s[i])
            i += 1
        for k in range(num):
            out += ch
    return out


check("rle encode aaabbc", rle_encode("aaabbc"), "a3b2c1")
check("rle encode single", rle_encode("x"), "x1")
check("rle encode long run", rle_encode("wwwwwwwwwwww"), "w12")
check("rle decode", rle_decode("a3b2c1"), "aaabbc")
check("rle decode long", rle_decode("w12"), "wwwwwwwwwwww")
rle_ok = 0
for s in ["aaabbc", "abcabc", "zzzzz", "q", "mississippi"]:
    if rle_decode(rle_encode(s)) == s:
        rle_ok += 1
check("rle round-trips", rle_ok, 5)


# ----- Caesar cipher (letters shift, everything else passes through) -----

def caesar(s, shift):
    out = ""
    for ch in s:
        i = idx_in(LOWER, ch)
        if i >= 0:
            out += LOWER[(i + shift) % 26]
        else:
            j = idx_in(UPPER, ch)
            if j >= 0:
                out += UPPER[(j + shift) % 26]
            else:
                out += ch
    return out


check("caesar shift 3", caesar("abc", 3), "def")
check("caesar wrap", caesar("xyz", 3), "abc")
check("caesar upper", caesar("Hello, World!", 5), "Mjqqt, Btwqi!")
check("caesar decode", caesar(caesar("Attack at dawn", 7), -7), "Attack at dawn")
rot13_twice = caesar(caesar("The quick brown fox", 13), 13)
check("rot13 twice is identity", rot13_twice, "The quick brown fox")


# ----- manual split and join -----

def split_words(s):
    words = []
    cur = ""
    for ch in s:
        if ch == " ":
            if len(cur) > 0:
                words.append(cur)
                cur = ""
        else:
            cur += ch
    if len(cur) > 0:
        words.append(cur)
    return words


def join_words(words, sep):
    out = ""
    for i in range(len(words)):
        if i > 0:
            out += sep
        out += words[i]
    return out


check("split simple", render(split_words("a b c")), "['a', 'b', 'c']")
check("split extra spaces", render(split_words("  the   fox  ")), "['the', 'fox']")
check("split empty", len(split_words("   ")), 0)
check("join words", join_words(["a", "b", "c"], "-"), "a-b-c")
check("split then join", join_words(split_words("hello there world"), " "), "hello there world")


# ----- word frequency counting -----

def word_freq(text):
    counts = {}
    for w in split_words(to_lower(text)):
        counts[w] = counts.get(w, 0) + 1
    return counts


freq = word_freq("the cat the dog the bird cat")
check("freq the", freq["the"], 3)
check("freq cat", freq["cat"], 2)
check("freq dog", freq["dog"], 1)
check("freq distinct words", len(freq), 4)
check("freq render", render(freq), "{'the': 3, 'cat': 2, 'dog': 1, 'bird': 1}")

# the most frequent word (first-seen wins ties, matching insertion order)
def most_common(counts):
    best = ""
    best_n = -1
    for pair in counts.items():
        if pair[1] > best_n:
            best_n = pair[1]
            best = pair[0]
    return best


check("most common", most_common(freq), "the")


# ----- vowel counting and a character histogram -----

def count_vowels(s):
    n = 0
    for ch in to_lower(s):
        if ch in VOWELS:
            n += 1
    return n


def histogram(s):
    hist = {}
    for ch in s:
        hist[ch] = hist.get(ch, 0) + 1
    return hist


check("vowels hello", count_vowels("hello"), 2)
check("vowels sentence", count_vowels("The Quick Brown Fox"), 5)
h = histogram("banana")
check("hist b", h["b"], 1)
check("hist a", h["a"], 3)
check("hist n", h["n"], 2)
check("hist render", render(histogram("mississippi")), "{'m': 1, 'i': 4, 's': 4, 'p': 2}")


# ----- atoi and itoa built by hand (no int()/str()) -----

def atoi(s):
    if len(s) == 0:
        return 0
    neg = False
    start = 0
    if s[0] == "-":
        neg = True
        start = 1
    val = 0
    for i in range(start, len(s)):
        val = val * 10 + idx_in(DIGITS, s[i])
    if neg:
        return -val
    return val


def itoa(n):
    if n == 0:
        return "0"
    neg = n < 0
    x = n
    if neg:
        x = -x
    out = ""
    while x > 0:
        out = DIGITS[x % 10] + out
        x = x // 10
    if neg:
        return "-" + out
    return out


check("atoi 12345", atoi("12345"), 12345)
check("atoi negative", atoi("-42"), -42)
check("atoi zero", atoi("0"), 0)
check("itoa 6789", itoa(6789), "6789")
check("itoa negative", itoa(-500), "-500")
check("itoa matches f-string", itoa(31415), f"{31415}")
num_ok = 0
for n in [0, 7, 42, -13, 1000, -99999, 250]:
    if atoi(itoa(n)) == n:
        num_ok += 1
check("atoi/itoa round-trip", num_ok, 7)


# ----- substring counting and anagrams -----

def count_sub(hay, needle):
    if len(needle) == 0:
        return 0
    count = 0
    for i in range(len(hay) - len(needle) + 1):
        if hay[i:i + len(needle)] == needle:
            count += 1
    return count


def is_anagram(a, b):
    if len(a) != len(b):
        return False
    counts = {}
    for ch in a:
        counts[ch] = counts.get(ch, 0) + 1
    for ch in b:
        if ch not in counts:
            return False
        counts[ch] = counts[ch] - 1
        if counts[ch] < 0:
            return False
    return True


check("count aa in aaaa", count_sub("aaaa", "aa"), 3)
check("count ab in ababab", count_sub("ababab", "ab"), 3)
check("count missing", count_sub("hello", "z"), 0)
check("count too long", count_sub("hi", "hello"), 0)
check_true("listen/silent anagram", is_anagram("listen", "silent"))
check_true("empty anagram", is_anagram("", ""))
check("different length", is_anagram("abc", "ab"), False)
check("not anagram", is_anagram("abc", "abd"), False)


check("no failures", fails[0], 0)
if fails[0] == 0:
    print("Python big-4 (string processing) self test passed")
exit(fails[0])
