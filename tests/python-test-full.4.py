# Full-syntax test: Python (3.12 core grammar) - part 4, the library-module
# sections 148, 149 and 150 (datetime, decimal, struct, array, pprint).
#
# Like part 3, each of these compiles whole Python-source modules
# (lib/python-*.py), which a -frozen interpreter leg pays for in IR steps, so
# they have a file of their own to keep every file clear of the step brake
# (-max-steps, 100M per call); see tests/python-test-full.py for the
# conventions, which are the same.

fails = [0]
checks = [0]

def check(name, cond):
    checks[0] += 1
    if not cond:
        println("FAIL " + name)
        fails[0] += 1

# ===== SECTION 148: datetime =====
# The calendar, timedelta's normal form and text, ISO 8601, strftime in the C
# locale and fixed-offset time zones; every value here is CPython 3.14's.
import datetime as S148dt

def s148():
    td = S148dt.timedelta
    check("dt1", [str(x) for x in [td(days=-1, seconds=5), td(hours=25), td(weeks=1, milliseconds=1), td(seconds=1.5)]]
          == ["-1 day, 0:00:05", "1 day, 1:00:00", "7 days, 0:00:00.001000", "0:00:01.500000"]
          and repr(td(minutes=-1)) == "datetime.timedelta(days=-1, seconds=86340)" and repr(td(0)) == "datetime.timedelta(0)"
          and td(days=1) / td(hours=5) == 4.8 and td(days=1) // td(hours=5) == 4 and str(td(days=1) % td(hours=5)) == "4:00:00"
          and str(td(hours=1) * 1.5) == "1:30:00" and td(1).total_seconds() == 86400.0 and hash(td(1)) == hash(td(hours=24)))
    d = S148dt.date(2024, 1, 5)
    check("dt2", repr(d) == "datetime.date(2024, 1, 5)" and d.toordinal() == 738890 and d.weekday() == 4
          and tuple(d.isocalendar()) == (2024, 1, 5) and str(S148dt.date(2021, 1, 3).isocalendar()) == "datetime.IsoCalendarDate(year=2020, week=53, weekday=7)"
          and d.strftime("%Y-%m-%d %a %A %b %B %j %U %W %y %x %%") == "2024-01-05 Fri Friday Jan January 005 00 01 24 01/05/24 %"
          and d.ctime() == "Fri Jan  5 00:00:00 2024" and str(d + td(400)) == "2025-02-08" and str(S148dt.date(2024, 3, 1) - S148dt.date(2024, 1, 1)) == "60 days, 0:00:00"
          and S148dt.date.fromisoformat("20240105") == d and S148dt.date.fromordinal(738000) == S148dt.date(2021, 7, 29)
          and f"{d:%d.%m.%Y}" == "05.01.2024")
    errs = []
    for f in [lambda: S148dt.date(2024, 2, 30), lambda: S148dt.date(2024, 13, 1), lambda: S148dt.time(24),
              lambda: S148dt.date.max + td(1), lambda: td(days=1000000000), lambda: S148dt.date.fromisoformat("2024-1-1")]:
        try:
            f()
        except (ValueError, OverflowError) as e:
            errs.append(type(e).__name__ + ": " + str(e))
    check("dt3", errs == ["ValueError: day 30 must be in range 1..29 for month 2 in year 2024",
                          "ValueError: month must be in 1..12, not 13", "ValueError: hour must be in 0..23, not 24",
                          "OverflowError: date value out of range",
                          "OverflowError: days=1000000000; must have magnitude <= 999999999",
                          "ValueError: Invalid isoformat string: '2024-1-1'"])
    utc = S148dt.timezone.utc
    ist = S148dt.timezone(td(hours=5, minutes=30))
    x = S148dt.datetime(2024, 12, 31, 23, 59, 59, 999999, ist)
    a = S148dt.datetime(2024, 1, 5, 12, tzinfo=utc)
    check("dt4", repr(x) == "datetime.datetime(2024, 12, 31, 23, 59, 59, 999999, tzinfo=datetime.timezone(datetime.timedelta(seconds=19800)))"
          and str(x) == "2024-12-31 23:59:59.999999+05:30" and x.isoformat(timespec="minutes") == "2024-12-31T23:59+05:30"
          and x.strftime("%H:%M:%S.%f %I %p %z %Z|%:z") == "23:59:59.999999 11 PM +0530 UTC+05:30|+05:30"
          and str(x + td(hours=12)) == "2025-01-01 11:59:59.999999+05:30" and str(a.astimezone(ist)) == "2024-01-05 17:30:00+05:30"
          and a == a.astimezone(ist) and a.timestamp() == 1704456000.0
          and str(S148dt.datetime.fromtimestamp(1700000000.5, ist)) == "2023-11-15 03:43:20.500000+05:30"
          and S148dt.datetime.fromisoformat("2024-01-05T10:20:30.123+05:00").isoformat() == "2024-01-05T10:20:30.123000+05:00"
          and str(S148dt.datetime.combine(d, S148dt.time(10, 30))) == "2024-01-05 10:30:00"
          and S148dt.datetime(2024, 1, 5) != d and repr(S148dt.time(1, 2, 3, 4)) == "datetime.time(1, 2, 3, 4)")
    try:
        a < S148dt.datetime(2024, 1, 5)
        check("dt5", False)
    except TypeError as e:
        check("dt5", str(e) == "can't compare offset-naive and offset-aware datetimes")

# ===== SECTION 149: decimal =====
# The default context (28 digits, half even), exact construction from a float,
# the specification's arithmetic and exponents, quantize, sqrt, format() and the
# trapped signals' text; every value here is CPython 3.14's.
import decimal as S149dec

def s149():
    D = S149dec.Decimal
    check("dec1", [str(x) for x in [D(1) / D(3), D(2) / D(3), D(10) / D(4), D("1.00") / D("0.1"), D("1.1") + D("2.20"),
                                    D("1.5") * D("1.5"), D(2) ** 100, D(3) ** -1, D(2).sqrt(), D("0.25").sqrt()]]
          == ["0.3333333333333333333333333333", "0.6666666666666666666666666667", "2.5", "10.0", "3.30", "2.25",
              "1.267650600228229401496703205E+30", "0.3333333333333333333333333333", "1.414213562373095048801688724", "0.5"]
          and repr(D(1.1)) == "Decimal('1.100000000000000088817841970012523233890533447265625')"
          and str(D("-7") // D(2)) == "-3" and str(D("-7") % D(2)) == "-1" and str(D("1e5")) == "1E+5"
          and str(D("123.456").quantize(D("0.01"))) == "123.46" and str(D("2.675").quantize(D("0.01"), rounding=S149dec.ROUND_HALF_UP)) == "2.68"
          and str(D("1200").normalize()) == "1.2E+3" and D("1.5").as_tuple() == (0, (1, 5), -1)
          and round(D("2.5")) == 2 and round(D("3.5")) == 4 and int(D("-7.9")) == -7 and float(D("1.1")) == 1.1
          and D("1.0") == 1 and D("0.5") == 0.5 and hash(D("0.5")) == hash(0.5) and sum([D("0.1")] * 3) == D("0.3"))
    check("dec2", [format(D("1234567.891"), s) for s in ["", ",.2f", ".3e", "^12.3f", "015,.2f", ".0%"]]
          == ["1234567.891", "1,234,567.89", "1.235e+6", "1234567.891 ", "0,001,234,567.89", "123456789%"])
    errs = []
    for f in [lambda: D(1) / 0, lambda: D(0) / 0, lambda: D("abc"), lambda: D("9e999999") * 10, lambda: divmod(D(1), 0)]:
        try:
            f()
        except S149dec.DecimalException as e:
            errs.append(type(e).__name__ + " " + str(e))
    check("dec3", errs == ["DivisionByZero [<class 'decimal.DivisionByZero'>]",
                           "InvalidOperation [<class 'decimal.DivisionUndefined'>]",
                           "InvalidOperation [<class 'decimal.ConversionSyntax'>]", "Overflow [<class 'decimal.Overflow'>]",
                           "InvalidOperation [<class 'decimal.InvalidOperation'>, <class 'decimal.DivisionByZero'>]"]
          and issubclass(S149dec.DivisionByZero, ZeroDivisionError))
    S149dec.getcontext().clear_flags()
    with S149dec.localcontext(prec=5, traps=[]) as c:
        r = [str(D(1) / D(7)), str(D(1) / 0), str(D(0) / 0)]
        rc = repr(c)
    check("dec4", r == ["0.14286", "Infinity", "NaN"] and S149dec.getcontext().prec == 28
          and rc == "Context(prec=5, rounding=ROUND_HALF_EVEN, Emin=-999999, Emax=999999, capitals=1, clamp=0, "
                    "flags=[DivisionByZero, InvalidOperation, Inexact, Rounded], traps=[])")

# ===== SECTION 150: struct, array, pprint =====
# struct's sizes, alignment and exact float encodings, array's item checks and
# bytes, pprint's line breaking.
import struct as S150st
import array as S150ar
import pprint as S150pp

def s150():
    check("struct1", S150st.calcsize("@bhiq") == 16 and S150st.calcsize("<bhiq") == 15 and S150st.calcsize("bxl") == 16
          and S150st.pack(">hId", -2, 7, 1.5) == b"\xff\xfe\x00\x00\x00\x07?\xf8\x00\x00\x00\x00\x00\x00"
          and S150st.pack("<f", 0.1) == b"\xcd\xcc\xcc=" and S150st.pack("e", 65520.0 / 2) == b"\x00x"
          and S150st.unpack("<f", b"\xcd\xcc\xcc=") == (0.10000000149011612,) and S150st.unpack("<3sxh", b"abc\x00\x05\x00") == (b"abc", 5)
          and S150st.pack("<d", 5e-324) == b"\x01\x00\x00\x00\x00\x00\x00\x00" and S150st.Struct(">2h").size == 4)
    errs = []
    for f in [lambda: S150st.pack("b", 200), lambda: S150st.pack("2h", 1), lambda: S150st.unpack("i", b"ab")]:
        try:
            f()
        except S150st.error as e:
            errs.append(str(e))
    try:
        S150st.pack("f", 1e300)
    except OverflowError as e:
        errs.append(str(e))
    check("struct2", errs == ["'b' format requires -128 <= number <= 127", "pack expected 2 items for packing (got 1)",
                              "unpack requires a buffer of 4 bytes", "float too large to pack with f format"])
    A = S150ar.array
    a = A("i", [1, 2, 3])
    a.append(4)
    a[1:3] = A("i", [7, 7, 7])
    b = A("h")
    b.frombytes(b"\x01\x00\x02\x00")
    b.byteswap()
    msgs = []
    for f in [lambda: A("b", [200]), lambda: A("I", [-1]), lambda: A("z")]:
        try:
            f()
        except (OverflowError, ValueError) as e:
            msgs.append(str(e))
    check("array1", repr(a) == "array('i', [1, 7, 7, 7, 4])" and a.tobytes()[:8] == b"\x01\x00\x00\x00\x07\x00\x00\x00"
          and repr(b) == "array('h', [256, 512])" and A("f", [0.1])[0] == 0.10000000149011612 and repr(A("w", "hi")) == "array('w', 'hi')"
          and a.itemsize == 4 and A("q").itemsize == 8 and S150ar.typecodes == "bBuwhHiIlLqQfd"
          and msgs == ["signed char is greater than maximum", "can't convert negative value to unsigned int",
                       "bad typecode (must be b, B, u, w, h, H, i, I, l, L, q, Q, f or d)"])
    data = {"name": "metacompiler", "langs": ["python", "ruby", "kotlin", "javascript", "typescript", "go", "lua"],
            "z": (1,), "set": {3, 1, 2}}
    check("pprint1", S150pp.pformat(data) == "{'langs': ['python', 'ruby', 'kotlin', 'javascript', 'typescript', 'go', 'lua'],\n"
                                             " 'name': 'metacompiler',\n 'set': {1, 2, 3},\n 'z': (1,)}"
          and S150pp.pformat(list(range(30)), width=40, compact=True) == "[0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,\n 12, 13, 14, 15, 16, 17, 18, 19, 20, 21,\n 22, 23, 24, 25, 26, 27, 28, 29]"
          and S150pp.pformat({"b": 1, "a": [1, 2]}, sort_dicts=False, width=10) == "{'b': 1,\n 'a': [1,\n       2]}"
          and S150pp.saferepr({"b": [1, 2], "a": (3,)}) == "{'a': (3,), 'b': [1, 2]}" and S150pp.isreadable([1, 2]))

def main():
    s148() # SECTION-CALL 148
    s149() # SECTION-CALL 149
    s150() # SECTION-CALL 150
    println(f"full: {checks[0]} checks, {fails[0]} failures")
    return fails[0]
