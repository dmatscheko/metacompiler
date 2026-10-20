# python-datetime.py -- the datetime module, written in Python and compiled
# with a program that imports it (core.importFile in both python grammars), so
# one implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_datetime);
# helpers carry the _mec_dt prefix. The behaviour is Lib/_pydatetime.py's
# (3.14): the proleptic Gregorian calendar, timedelta normalized to days,
# seconds and microseconds, fixed-offset timezones, ISO 8601 and strftime in
# the C locale, error texts included. What needs the machine's clock or local
# zone - today(), now(), a naive timestamp() or fromtimestamp() without tz - is
# left out. An error naming a type says 'date' where CPython says
# 'datetime.date'.

_mec_dt_MINYEAR = 1
_mec_dt_MAXYEAR = 9999
_mec_dt_MAXORDINAL = 3652059
_mec_dt_DIM = [-1, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31]
_mec_dt_DBM = [-1]
_mec_dt_n = 0
for _mec_dt_m in _mec_dt_DIM[1:]:
    _mec_dt_DBM.append(_mec_dt_n)
    _mec_dt_n += _mec_dt_m
_mec_dt_MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]
_mec_dt_MONTHNAMES = ["January", "February", "March", "April", "May", "June", "July", "August",
                      "September", "October", "November", "December"]
_mec_dt_DAYS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
_mec_dt_DAYNAMES = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"]


def _mec_dt_leap(y):
    return y % 4 == 0 and (y % 100 != 0 or y % 400 == 0)


def _mec_dt_dim(y, m):
    return 29 if m == 2 and _mec_dt_leap(y) else _mec_dt_DIM[m]


def _mec_dt_days_before_year(y):
    y -= 1
    return y * 365 + y // 4 - y // 100 + y // 400


def _mec_dt_ymd2ord(y, m, d):
    return _mec_dt_days_before_year(y) + _mec_dt_DBM[m] + (m > 2 and _mec_dt_leap(y)) + d


# The (year, month, day) of a proleptic Gregorian ordinal, 1 being 0001-01-01.
def _mec_dt_ord2ymd(n):
    n -= 1
    n400, n = divmod(n, 146097)
    year = n400 * 400 + 1
    n100, n = divmod(n, 36524)
    n4, n = divmod(n, 1461)
    n1, n = divmod(n, 365)
    year += n100 * 100 + n4 * 4 + n1
    if n1 == 4 or n100 == 4:
        return year - 1, 12, 31
    leap = n1 == 3 and (n4 != 24 or n100 == 3)
    month = (n + 50) >> 5
    preceding = _mec_dt_DBM[month] + (month > 2 and leap)
    if preceding > n:
        month -= 1
        preceding -= _mec_dt_DIM[month] + (month == 2 and leap)
    return year, month, n - preceding + 1


def _mec_dt_int(v):
    if isinstance(v, int):
        return int(v)
    if hasattr(type(v), "__index__"):
        return v.__index__()
    raise TypeError("'" + type(v).__name__ + "' object cannot be interpreted as an integer")


def _mec_dt_check_date(year, month, day):
    year, month, day = _mec_dt_int(year), _mec_dt_int(month), _mec_dt_int(day)
    if not _mec_dt_MINYEAR <= year <= _mec_dt_MAXYEAR:
        raise ValueError("year must be in 1..9999, not " + str(year))
    if not 1 <= month <= 12:
        raise ValueError("month must be in 1..12, not " + str(month))
    dim = _mec_dt_dim(year, month)
    if not 1 <= day <= dim:
        raise ValueError("day " + str(day) + " must be in range 1.." + str(dim) + " for month " + str(month)
                         + " in year " + str(year))
    return year, month, day


def _mec_dt_check_time(hour, minute, second, microsecond, fold):
    hour, minute = _mec_dt_int(hour), _mec_dt_int(minute)
    second, microsecond = _mec_dt_int(second), _mec_dt_int(microsecond)
    for name, v, hi in (("hour", hour, 23), ("minute", minute, 59), ("second", second, 59),
                        ("microsecond", microsecond, 999999)):
        if not 0 <= v <= hi:
            raise ValueError(name + " must be in 0.." + str(hi) + ", not " + str(v))
    if fold not in (0, 1):
        raise ValueError("fold must be either 0 or 1")
    return hour, minute, second, microsecond


def _mec_dt_check_tz(tz):
    if tz is not None and not isinstance(tz, _mec_dt_tzinfo):
        raise TypeError("tzinfo argument must be None or of a tzinfo subclass, not type '"
                        + type(tz).__name__ + "'")


def _mec_dt_2(n):
    return ("0" if n < 10 else "") + str(n)


def _mec_dt_pad(n, w):
    s = str(n)
    return "0" * (w - len(s)) + s


# HH:MM:SS[.ffffff] under timespec ('auto' leaves out zero microseconds).
def _mec_dt_fmt_time(hh, mm, ss, us, timespec):
    if timespec == "auto":
        timespec = "microseconds" if us else "seconds"
    if timespec == "hours":
        return _mec_dt_2(hh)
    if timespec == "minutes":
        return _mec_dt_2(hh) + ":" + _mec_dt_2(mm)
    base = _mec_dt_2(hh) + ":" + _mec_dt_2(mm) + ":" + _mec_dt_2(ss)
    if timespec == "seconds":
        return base
    if timespec == "milliseconds":
        return base + "." + _mec_dt_pad(us // 1000, 3)
    if timespec == "microseconds":
        return base + "." + _mec_dt_pad(us, 6)
    raise ValueError("Unknown timespec value")


# A utcoffset as +HH:MM[:SS[.ffffff]], sep between the fields.
def _mec_dt_fmt_off(off, sep=":"):
    if off is None:
        return ""
    sign = "+"
    if off.days < 0:
        sign = "-"
        off = -off
    mm, ss = divmod(off.days * 86400 + off.seconds, 60)
    hh, mm = divmod(mm, 60)
    s = sign + _mec_dt_2(hh) + sep + _mec_dt_2(mm)
    if ss or off.microseconds:
        s += sep + _mec_dt_2(ss)
        if off.microseconds:
            s += "." + _mec_dt_pad(off.microseconds, 6)
    return s


def _mec_dt_check_off(name, off):
    if off is None:
        return
    if not isinstance(off, _mec_dt_timedelta):
        raise TypeError("tzinfo." + name + "() must return None or timedelta, not '" + type(off).__name__ + "'")
    if not -_mec_dt_DAY < off < _mec_dt_DAY:
        raise ValueError(name + "()=" + str(off) + ", must be strictly between -timedelta(hours=24) and "
                         "timedelta(hours=24)")


# ----------------------------------------------------------------- timedelta

class _mec_dt_timedelta:
    def __init__(self, days=0, seconds=0, microseconds=0, milliseconds=0, minutes=0, hours=0, weeks=0):
        exact = 0
        frac = 0.0
        for name, v, scale in (("days", days, 86400000000), ("seconds", seconds, 1000000),
                               ("microseconds", microseconds, 1), ("milliseconds", milliseconds, 1000),
                               ("minutes", minutes, 60000000), ("hours", hours, 3600000000),
                               ("weeks", weeks, 604800000000)):
            if isinstance(v, int):
                exact += int(v) * scale
            elif isinstance(v, float):
                whole = int(v)
                exact += whole * scale
                frac += (v - whole) * scale
            else:
                raise TypeError("unsupported type for timedelta " + name + " component: " + type(v).__name__)
        _mec_dt_td_set(self, exact + round(frac))

    @property
    def days(self):
        return self._d

    @property
    def seconds(self):
        return self._s

    @property
    def microseconds(self):
        return self._us

    def _mec_us(self):
        return (self._d * 86400 + self._s) * 1000000 + self._us

    def total_seconds(self):
        return self._mec_us() / 10 ** 6

    def __repr__(self):
        parts = []
        if self._d:
            parts.append("days=" + str(self._d))
        if self._s:
            parts.append("seconds=" + str(self._s))
        if self._us:
            parts.append("microseconds=" + str(self._us))
        return "datetime.timedelta(" + (", ".join(parts) if parts else "0") + ")"

    def __str__(self):
        mm, ss = divmod(self._s, 60)
        hh, mm = divmod(mm, 60)
        s = str(hh) + ":" + _mec_dt_2(mm) + ":" + _mec_dt_2(ss)
        if self._d:
            s = str(self._d) + " day" + ("s" if abs(self._d) != 1 else "") + ", " + s
        if self._us:
            s += "." + _mec_dt_pad(self._us, 6)
        return s

    def __add__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            return _mec_dt_td(self._mec_us() + other._mec_us())
        return NotImplemented

    __radd__ = __add__

    def __sub__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            return _mec_dt_td(self._mec_us() - other._mec_us())
        return NotImplemented

    def __rsub__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            return _mec_dt_td(other._mec_us() - self._mec_us())
        return NotImplemented

    def __neg__(self):
        return _mec_dt_td(-self._mec_us())

    def __pos__(self):
        return self

    def __abs__(self):
        return -self if self._d < 0 else self

    def __mul__(self, other):
        if isinstance(other, int):
            return _mec_dt_td(self._mec_us() * other)
        if isinstance(other, float):
            n, d = other.as_integer_ratio()
            return _mec_dt_td(_mec_dt_div_even(self._mec_us() * n, d))
        return NotImplemented

    __rmul__ = __mul__

    def __truediv__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            return self._mec_us() / other._mec_us()
        if isinstance(other, int):
            if other == 0:
                raise ZeroDivisionError("division by zero")
            return _mec_dt_td(_mec_dt_div_even(self._mec_us(), other))
        if isinstance(other, float):
            n, d = other.as_integer_ratio()
            if n == 0:
                raise ZeroDivisionError("division by zero")
            return _mec_dt_td(_mec_dt_div_even(d * self._mec_us(), n))
        return NotImplemented

    def __floordiv__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            if other._mec_us() == 0:
                raise ZeroDivisionError("division by zero")
            return self._mec_us() // other._mec_us()
        if isinstance(other, int):
            if other == 0:
                raise ZeroDivisionError("division by zero")
            return _mec_dt_td(self._mec_us() // other)
        return NotImplemented

    def __mod__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            if other._mec_us() == 0:
                raise ZeroDivisionError("division by zero")
            return _mec_dt_td(self._mec_us() % other._mec_us())
        return NotImplemented

    def __divmod__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            if other._mec_us() == 0:
                raise ZeroDivisionError("division by zero")
            q, r = divmod(self._mec_us(), other._mec_us())
            return q, _mec_dt_td(r)
        return NotImplemented

    def __eq__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            return self._mec_us() == other._mec_us()
        return NotImplemented

    def __lt__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            return self._mec_us() < other._mec_us()
        return NotImplemented

    def __le__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            return self._mec_us() <= other._mec_us()
        return NotImplemented

    def __gt__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            return self._mec_us() > other._mec_us()
        return NotImplemented

    def __ge__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            return self._mec_us() >= other._mec_us()
        return NotImplemented

    def __hash__(self):
        return hash((self._d, self._s, self._us))

    def __bool__(self):
        return self._d != 0 or self._s != 0 or self._us != 0


# n / d rounded half to even, for integers.
def _mec_dt_div_even(n, d):
    q, r = divmod(n, d)
    r2 = r * 2
    if d < 0:
        if r2 < d or (r2 == d and q % 2 == 1):
            q += 1
    elif r2 > d or (r2 == d and q % 2 == 1):
        q += 1
    return q


def _mec_dt_td_set(td, us):
    s, u = divmod(us, 1000000)
    d, s = divmod(s, 86400)
    if abs(d) > 999999999:
        raise OverflowError("days=" + str(d) + "; must have magnitude <= 999999999")
    td._d = d
    td._s = s
    td._us = u


def _mec_dt_td(us):
    td = object.__new__(_mec_dt_timedelta)
    _mec_dt_td_set(td, us)
    return td


_mec_dt_DAY = _mec_dt_td(86400000000)


# ---------------------------------------------------------------------- tzinfo

class _mec_dt_tzinfo:
    def tzname(self, dt):
        raise NotImplementedError("tzinfo subclass must override tzname()")

    def utcoffset(self, dt):
        raise NotImplementedError("tzinfo subclass must override utcoffset()")

    def dst(self, dt):
        raise NotImplementedError("tzinfo subclass must override dst()")

    def fromutc(self, dt):
        off = dt.utcoffset()
        dst = dt.dst()
        if dst is None:
            raise ValueError("fromutc() requires a non-None dst() result")
        dt += off - dst
        dst = dt.dst()
        return dt + dst if dst else dt


class _mec_dt_timezone(_mec_dt_tzinfo):
    def __init__(self, offset, name=None):
        if not isinstance(offset, _mec_dt_timedelta):
            raise TypeError("offset must be a timedelta")
        if name is not None and not isinstance(name, str):
            raise TypeError("name must be a string")
        if not -_mec_dt_DAY < offset < _mec_dt_DAY:
            raise ValueError("offset must be a timedelta strictly between -timedelta(hours=24) and "
                             "timedelta(hours=24), not " + repr(offset))
        self._offset = offset
        self._name = name

    def utcoffset(self, dt):
        return self._offset

    def dst(self, dt):
        return None

    def tzname(self, dt):
        if self._name is not None:
            return self._name
        if not self._offset:
            return "UTC"
        return "UTC" + _mec_dt_fmt_off(self._offset)

    def fromutc(self, dt):
        if dt.tzinfo is not self:
            raise ValueError("fromutc: dt.tzinfo is not self")
        return dt + self._offset

    def __repr__(self):
        if self is _mec_dt_UTC:
            return "datetime.timezone.utc"
        if self._name is None:
            return "datetime.timezone(" + repr(self._offset) + ")"
        return "datetime.timezone(" + repr(self._offset) + ", " + repr(self._name) + ")"

    def __str__(self):
        return self.tzname(None)

    def __eq__(self, other):
        if isinstance(other, _mec_dt_timezone):
            return self._offset == other._offset
        return NotImplemented

    def __hash__(self):
        return hash(self._offset)


_mec_dt_UTC = _mec_dt_timezone(_mec_dt_td(0))


# ------------------------------------------------------------------- strftime

def _mec_dt_strftime(fmt, y, mo, d, hh, mm, ss, us, tz, owner):
    out = []
    i = 0
    n = len(fmt)
    wd = (_mec_dt_ymd2ord(y, mo, d) + 6) % 7
    yday = _mec_dt_DBM[mo] + (mo > 2 and _mec_dt_leap(y)) + d
    while i < n:
        c = fmt[i]
        if c != "%" or i + 1 >= n:
            out.append(c)
            i += 1
            continue
        k = fmt[i + 1]
        i += 2
        if k == ":" and i < n and fmt[i] == "z":
            i += 1
            out.append(_mec_dt_fmt_off(owner.utcoffset() if tz is not None else None))
        elif k == "Y":
            out.append(_mec_dt_pad(y, 4))
        elif k == "y":
            out.append(_mec_dt_2(y % 100))
        elif k == "m":
            out.append(_mec_dt_2(mo))
        elif k == "d":
            out.append(_mec_dt_2(d))
        elif k == "e":
            out.append(("" if d >= 10 else " ") + str(d))
        elif k == "H":
            out.append(_mec_dt_2(hh))
        elif k == "I":
            out.append(_mec_dt_2(hh % 12 or 12))
        elif k == "M":
            out.append(_mec_dt_2(mm))
        elif k == "S":
            out.append(_mec_dt_2(ss))
        elif k == "f":
            out.append(_mec_dt_pad(us, 6))
        elif k == "p":
            out.append("AM" if hh < 12 else "PM")
        elif k == "b" or k == "h":
            out.append(_mec_dt_MONTHS[mo - 1])
        elif k == "B":
            out.append(_mec_dt_MONTHNAMES[mo - 1])
        elif k == "a":
            out.append(_mec_dt_DAYS[wd])
        elif k == "A":
            out.append(_mec_dt_DAYNAMES[wd])
        elif k == "w":
            out.append(str((wd + 1) % 7))
        elif k == "u":
            out.append(str(wd + 1))
        elif k == "j":
            out.append(_mec_dt_pad(yday, 3))
        elif k == "U":
            out.append(_mec_dt_2((yday + 6 - (wd + 1) % 7) // 7))
        elif k == "W":
            out.append(_mec_dt_2((yday + 6 - wd) // 7))
        elif k == "G" or k == "V":
            iy, iw, _ = _mec_dt_isocal(y, mo, d)
            out.append(_mec_dt_pad(iy, 4) if k == "G" else _mec_dt_2(iw))
        elif k == "z":
            out.append(_mec_dt_fmt_off(owner.utcoffset() if tz is not None else None, ""))
        elif k == "Z":
            name = owner.tzname() if tz is not None else None
            out.append("" if name is None else name)
        elif k == "c":
            out.append(_mec_dt_DAYS[wd] + " " + _mec_dt_MONTHS[mo - 1] + " " + ("" if d >= 10 else " ") + str(d)
                       + " " + _mec_dt_2(hh) + ":" + _mec_dt_2(mm) + ":" + _mec_dt_2(ss) + " " + _mec_dt_pad(y, 4))
        elif k == "x":
            out.append(_mec_dt_2(mo) + "/" + _mec_dt_2(d) + "/" + _mec_dt_2(y % 100))
        elif k == "X":
            out.append(_mec_dt_2(hh) + ":" + _mec_dt_2(mm) + ":" + _mec_dt_2(ss))
        elif k == "F":
            out.append(str(y) + "-" + _mec_dt_2(mo) + "-" + _mec_dt_2(d))
        elif k == "T":
            out.append(_mec_dt_2(hh) + ":" + _mec_dt_2(mm) + ":" + _mec_dt_2(ss))
        elif k == "%":
            out.append("%")
        else:
            out.append("%" + k)
    return "".join(out)


# The ISO calendar (year, week, weekday) of a date: weeks start on Monday and
# week 1 holds the year's first Thursday.
def _mec_dt_isocal(y, m, d):
    def week1(year):
        first = _mec_dt_ymd2ord(year, 1, 1)
        wd = (first + 6) % 7
        start = first - wd
        if wd > 3:
            start += 7
        return start
    o = _mec_dt_ymd2ord(y, m, d)
    year = y
    start = week1(year)
    if o < start:
        year -= 1
        start = week1(year)
    elif o >= week1(year + 1):
        year += 1
        start = week1(year)
    week, day = divmod(o - start, 7)
    return year, week + 1, day + 1


class _mec_dt_IsoCalendarDate(tuple):
    @property
    def year(self):
        return self[0]

    @property
    def week(self):
        return self[1]

    @property
    def weekday(self):
        return self[2]

    def __repr__(self):
        return ("datetime.IsoCalendarDate(year=" + str(self[0]) + ", week=" + str(self[1]) + ", weekday="
                + str(self[2]) + ")")


# ------------------------------------------------------------------------ date

def _mec_dt_isodate(s):
    if len(s) == 10 and s[4] == "-" and s[7] == "-":
        parts = (s[0:4], s[5:7], s[8:10])
    elif len(s) == 8:
        parts = (s[0:4], s[4:6], s[6:8])
    else:
        return None
    for p in parts:
        if not (p.isdigit() and p.isascii()):
            return None
    return int(parts[0]), int(parts[1]), int(parts[2])


class _mec_dt_date:
    def __init__(self, year, month=None, day=None):
        self._y, self._m, self._d = _mec_dt_check_date(year, month, day)

    @property
    def year(self):
        return self._y

    @property
    def month(self):
        return self._m

    @property
    def day(self):
        return self._d

    @classmethod
    def fromordinal(cls, n):
        n = _mec_dt_int(n)
        if not 1 <= n <= _mec_dt_MAXORDINAL:
            raise ValueError("ordinal must be >= 1")
        y, m, d = _mec_dt_ord2ymd(n)
        return cls(y, m, d)

    @classmethod
    def fromisoformat(cls, s):
        if not isinstance(s, str):
            raise TypeError("fromisoformat: argument must be str")
        ymd = _mec_dt_isodate(s)
        if ymd is None:
            raise ValueError("Invalid isoformat string: " + repr(s))
        return cls(*ymd)

    @classmethod
    def fromisocalendar(cls, year, week, day):
        if not 1 <= day <= 7:
            raise ValueError("Invalid weekday: " + str(day) + " (range is [1, 7])")
        first = _mec_dt_ymd2ord(year, 1, 1)
        wd = (first + 6) % 7
        start = first - wd + (7 if wd > 3 else 0)
        y, m, d = _mec_dt_ord2ymd(start + (week - 1) * 7 + day - 1)
        return cls(y, m, d)

    def toordinal(self):
        return _mec_dt_ymd2ord(self._y, self._m, self._d)

    def weekday(self):
        return (self.toordinal() + 6) % 7

    def isoweekday(self):
        return self.weekday() + 1

    def isocalendar(self):
        return _mec_dt_IsoCalendarDate(_mec_dt_isocal(self._y, self._m, self._d))

    def isoformat(self):
        return _mec_dt_pad(self._y, 4) + "-" + _mec_dt_2(self._m) + "-" + _mec_dt_2(self._d)

    def __str__(self):
        return self.isoformat()

    def __repr__(self):
        return "datetime.date(" + str(self._y) + ", " + str(self._m) + ", " + str(self._d) + ")"

    def ctime(self):
        return self.strftime("%c")

    def strftime(self, fmt):
        return _mec_dt_strftime(fmt, self._y, self._m, self._d, 0, 0, 0, 0, None, self)

    def __format__(self, spec):
        if not isinstance(spec, str):
            raise TypeError("must be str, not " + type(spec).__name__)
        return self.strftime(spec) if spec else str(self)

    def replace(self, year=None, month=None, day=None):
        return type(self)(self._y if year is None else year, self._m if month is None else month,
                          self._d if day is None else day)

    def _mec_key(self):
        return (self._y, self._m, self._d)

    def _mec_same(self, other):
        return isinstance(other, _mec_dt_date) and not isinstance(other, _mec_dt_datetime)

    def __eq__(self, other):
        if isinstance(other, _mec_dt_datetime):
            return False
        if isinstance(other, _mec_dt_date):
            return self._mec_key() == other._mec_key()
        return NotImplemented

    def __lt__(self, other):
        if self._mec_same(other):
            return self._mec_key() < other._mec_key()
        return NotImplemented

    def __le__(self, other):
        if self._mec_same(other):
            return self._mec_key() <= other._mec_key()
        return NotImplemented

    def __gt__(self, other):
        if self._mec_same(other):
            return self._mec_key() > other._mec_key()
        return NotImplemented

    def __ge__(self, other):
        if self._mec_same(other):
            return self._mec_key() >= other._mec_key()
        return NotImplemented

    def __hash__(self):
        return hash(self._mec_key())

    def __add__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            o = self.toordinal() + other.days
            if not 1 <= o <= _mec_dt_MAXORDINAL:
                raise OverflowError("date value out of range")
            return type(self).fromordinal(o)
        return NotImplemented

    __radd__ = __add__

    def __sub__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            return self + -other
        if self._mec_same(other):
            return _mec_dt_td((self.toordinal() - other.toordinal()) * 86400000000)
        return NotImplemented


# ------------------------------------------------------------------------ time

class _mec_dt_time:
    def __init__(self, hour=0, minute=0, second=0, microsecond=0, tzinfo=None, *, fold=0):
        self._h, self._mi, self._s, self._us = _mec_dt_check_time(hour, minute, second, microsecond, fold)
        _mec_dt_check_tz(tzinfo)
        self._tz = tzinfo
        self._fold = fold

    @property
    def hour(self):
        return self._h

    @property
    def minute(self):
        return self._mi

    @property
    def second(self):
        return self._s

    @property
    def microsecond(self):
        return self._us

    @property
    def tzinfo(self):
        return self._tz

    @property
    def fold(self):
        return self._fold

    @classmethod
    def fromisoformat(cls, s):
        t = _mec_dt_isotime(s)
        if t is None:
            raise ValueError("Invalid isoformat string: " + repr(s))
        return cls(t[0], t[1], t[2], t[3], t[4])

    def utcoffset(self):
        if self._tz is None:
            return None
        off = self._tz.utcoffset(None)
        _mec_dt_check_off("utcoffset", off)
        return off

    def dst(self):
        return None if self._tz is None else self._tz.dst(None)

    def tzname(self):
        return None if self._tz is None else self._tz.tzname(None)

    def isoformat(self, timespec="auto"):
        return _mec_dt_fmt_time(self._h, self._mi, self._s, self._us, timespec) + _mec_dt_fmt_off(self.utcoffset())

    def __str__(self):
        return self.isoformat()

    def __repr__(self):
        s = "datetime.time(" + str(self._h) + ", " + str(self._mi)
        if self._s or self._us:
            s += ", " + str(self._s)
        if self._us:
            s += ", " + str(self._us)
        if self._tz is not None:
            s += ", tzinfo=" + repr(self._tz)
        if self._fold:
            s += ", fold=1"
        return s + ")"

    def strftime(self, fmt):
        return _mec_dt_strftime(fmt, 1900, 1, 1, self._h, self._mi, self._s, self._us, self._tz, self)

    def __format__(self, spec):
        return self.strftime(spec) if spec else str(self)

    def replace(self, hour=None, minute=None, second=None, microsecond=None, tzinfo=True, *, fold=None):
        return type(self)(self._h if hour is None else hour, self._mi if minute is None else minute,
                          self._s if second is None else second,
                          self._us if microsecond is None else microsecond,
                          self._tz if tzinfo is True else tzinfo, fold=self._fold if fold is None else fold)

    def _mec_key(self):
        off = self.utcoffset()
        us = ((self._h * 60 + self._mi) * 60 + self._s) * 1000000 + self._us
        if off is not None:
            us -= off._mec_us()
        return us

    def _mec_cmp(self, other, op):
        if not isinstance(other, _mec_dt_time):
            return NotImplemented
        if (self.utcoffset() is None) != (other.utcoffset() is None):
            if op == "==":
                return False
            raise TypeError("can't compare offset-naive and offset-aware times")
        a, b = self._mec_key(), other._mec_key()
        if op == "==":
            return a == b
        if op == "<":
            return a < b
        if op == "<=":
            return a <= b
        if op == ">":
            return a > b
        return a >= b

    def __eq__(self, other):
        return self._mec_cmp(other, "==")

    def __lt__(self, other):
        return self._mec_cmp(other, "<")

    def __le__(self, other):
        return self._mec_cmp(other, "<=")

    def __gt__(self, other):
        return self._mec_cmp(other, ">")

    def __ge__(self, other):
        return self._mec_cmp(other, ">=")

    def __hash__(self):
        return hash(self._mec_key())

    def __bool__(self):
        return True


# HH[:MM[:SS[.fff[fff]]]][+HH:MM[:SS[.ffffff]] | Z] as (h, m, s, us, tz), or None.
def _mec_dt_isotime(s):
    tz = None
    cut = -1
    for i, c in enumerate(s):
        if c in "+-Z":
            cut = i
            break
    if cut >= 0:
        tail = s[cut:]
        s = s[:cut]
        if tail == "Z":
            tz = _mec_dt_UTC
        else:
            t = _mec_dt_isotime(tail[1:])
            if t is None or t[4] is not None:
                return None
            off = _mec_dt_td(((t[0] * 60 + t[1]) * 60 + t[2]) * 1000000 + t[3])
            if tail[0] == "-":
                off = -off
            tz = _mec_dt_UTC if not off else _mec_dt_timezone(off)
    us = 0
    dot = s.find(".")
    if dot < 0:
        dot = s.find(",")
    if dot >= 0:
        fr = s[dot + 1:]
        s = s[:dot]
        if not (fr.isdigit() and fr.isascii()) or len(fr) not in (3, 6):
            return None
        us = int(fr) * (1000 if len(fr) == 3 else 1)
    if ":" in s:
        parts = s.split(":")
    else:
        parts = [s[i:i + 2] for i in range(0, len(s), 2)]
    if not 1 <= len(parts) <= 3:
        return None
    for p in parts:
        if len(p) != 2 or not (p.isdigit() and p.isascii()):
            return None
    while len(parts) < 3:
        parts.append("00")
    return int(parts[0]), int(parts[1]), int(parts[2]), us, tz


# -------------------------------------------------------------------- datetime

class _mec_dt_datetime(_mec_dt_date):
    def __init__(self, year, month=None, day=None, hour=0, minute=0, second=0, microsecond=0, tzinfo=None, *,
                 fold=0):
        self._y, self._m, self._d = _mec_dt_check_date(year, month, day)
        self._h, self._mi, self._s, self._us = _mec_dt_check_time(hour, minute, second, microsecond, fold)
        _mec_dt_check_tz(tzinfo)
        self._tz = tzinfo
        self._fold = fold

    @property
    def hour(self):
        return self._h

    @property
    def minute(self):
        return self._mi

    @property
    def second(self):
        return self._s

    @property
    def microsecond(self):
        return self._us

    @property
    def tzinfo(self):
        return self._tz

    @property
    def fold(self):
        return self._fold

    @classmethod
    def combine(cls, date, time, tzinfo=True):
        if not isinstance(date, _mec_dt_date):
            raise TypeError("combine() argument 1 must be datetime.date, not " + type(date).__name__)
        if not isinstance(time, _mec_dt_time):
            raise TypeError("combine() argument 2 must be datetime.time, not " + type(time).__name__)
        return cls(date.year, date.month, date.day, time.hour, time.minute, time.second, time.microsecond,
                   time.tzinfo if tzinfo is True else tzinfo, fold=time.fold)

    @classmethod
    def fromordinal(cls, n):
        n = _mec_dt_int(n)
        if not 1 <= n <= _mec_dt_MAXORDINAL:
            raise ValueError("ordinal must be >= 1")
        y, m, d = _mec_dt_ord2ymd(n)
        return cls(y, m, d)

    @classmethod
    def fromisoformat(cls, s):
        if not isinstance(s, str):
            raise TypeError("fromisoformat: argument must be str")
        if len(s) <= 10:
            ymd = _mec_dt_isodate(s)
            t = (0, 0, 0, 0, None)
        else:
            sep = 10 if s[4:5] == "-" else 8
            ymd = _mec_dt_isodate(s[:sep])
            t = _mec_dt_isotime(s[sep + 1:])
        if ymd is None or t is None:
            raise ValueError("Invalid isoformat string: " + repr(s))
        return cls(ymd[0], ymd[1], ymd[2], t[0], t[1], t[2], t[3], t[4])

    @classmethod
    def fromtimestamp(cls, t, tz=None):
        if tz is None:
            raise NotImplementedError("fromtimestamp() needs the local time zone; pass tz")
        us = round(t * 1000000) if isinstance(t, float) else t * 1000000
        base = _mec_dt_datetime(1970, 1, 1, tzinfo=_mec_dt_UTC) + _mec_dt_td(us)
        return tz.fromutc(base.replace(tzinfo=tz))

    def date(self):
        return _mec_dt_date(self._y, self._m, self._d)

    def time(self):
        return _mec_dt_time(self._h, self._mi, self._s, self._us, fold=self._fold)

    def timetz(self):
        return _mec_dt_time(self._h, self._mi, self._s, self._us, self._tz, fold=self._fold)

    def utcoffset(self):
        if self._tz is None:
            return None
        off = self._tz.utcoffset(self)
        _mec_dt_check_off("utcoffset", off)
        return off

    def dst(self):
        return None if self._tz is None else self._tz.dst(self)

    def tzname(self):
        return None if self._tz is None else self._tz.tzname(self)

    def timestamp(self):
        off = self.utcoffset()
        if off is None:
            raise NotImplementedError("timestamp() of a naive datetime needs the local time zone")
        return (self - _mec_dt_datetime(1970, 1, 1, tzinfo=_mec_dt_UTC)).total_seconds()

    def astimezone(self, tz=None):
        if tz is None:
            raise NotImplementedError("astimezone() needs the local time zone; pass tz")
        _mec_dt_check_tz(tz)
        off = self.utcoffset()
        if off is None:
            raise NotImplementedError("astimezone() of a naive datetime needs the local time zone")
        if tz is self._tz:
            return self
        utc = (self - off).replace(tzinfo=tz)
        return tz.fromutc(utc)

    def isoformat(self, sep="T", timespec="auto"):
        return (_mec_dt_date.isoformat(self) + sep + _mec_dt_fmt_time(self._h, self._mi, self._s, self._us, timespec)
                + _mec_dt_fmt_off(self.utcoffset()))

    def __str__(self):
        return self.isoformat(" ")

    def __repr__(self):
        s = ("datetime.datetime(" + str(self._y) + ", " + str(self._m) + ", " + str(self._d) + ", " + str(self._h)
             + ", " + str(self._mi))
        if self._s or self._us:
            s += ", " + str(self._s)
        if self._us:
            s += ", " + str(self._us)
        if self._tz is not None:
            s += ", tzinfo=" + repr(self._tz)
        if self._fold:
            s += ", fold=1"
        return s + ")"

    def strftime(self, fmt):
        return _mec_dt_strftime(fmt, self._y, self._m, self._d, self._h, self._mi, self._s, self._us, self._tz, self)

    def replace(self, year=None, month=None, day=None, hour=None, minute=None, second=None, microsecond=None,
                tzinfo=True, *, fold=None):
        return type(self)(self._y if year is None else year, self._m if month is None else month,
                          self._d if day is None else day, self._h if hour is None else hour,
                          self._mi if minute is None else minute, self._s if second is None else second,
                          self._us if microsecond is None else microsecond,
                          self._tz if tzinfo is True else tzinfo, fold=self._fold if fold is None else fold)

    # Microseconds since 0001-01-01T00:00, less the UTC offset.
    def _mec_us(self):
        us = ((self.toordinal() * 24 + self._h) * 60 + self._mi) * 60 + self._s
        us = us * 1000000 + self._us
        off = self.utcoffset()
        if off is not None:
            us -= off._mec_us()
        return us

    def _mec_cmp(self, other, op):
        if not isinstance(other, _mec_dt_datetime):
            if op == "==":
                return NotImplemented if not isinstance(other, _mec_dt_date) else False
            return NotImplemented
        if (self.utcoffset() is None) != (other.utcoffset() is None):
            if op == "==":
                return False
            raise TypeError("can't compare offset-naive and offset-aware datetimes")
        a, b = self._mec_us(), other._mec_us()
        if op == "==":
            return a == b
        if op == "<":
            return a < b
        if op == "<=":
            return a <= b
        if op == ">":
            return a > b
        return a >= b

    def __eq__(self, other):
        return self._mec_cmp(other, "==")

    def __lt__(self, other):
        return self._mec_cmp(other, "<")

    def __le__(self, other):
        return self._mec_cmp(other, "<=")

    def __gt__(self, other):
        return self._mec_cmp(other, ">")

    def __ge__(self, other):
        return self._mec_cmp(other, ">=")

    def __hash__(self):
        return hash(self._mec_us())

    def __add__(self, other):
        if not isinstance(other, _mec_dt_timedelta):
            return NotImplemented
        us = ((self._h * 60 + self._mi) * 60 + self._s) * 1000000 + self._us + other._mec_us()
        days, us = divmod(us, 86400000000)
        o = self.toordinal() + days
        if not 1 <= o <= _mec_dt_MAXORDINAL:
            raise OverflowError("date value out of range")
        y, m, d = _mec_dt_ord2ymd(o)
        s, us = divmod(us, 1000000)
        mi, s = divmod(s, 60)
        h, mi = divmod(mi, 60)
        return type(self)(y, m, d, h, mi, s, us, self._tz)

    __radd__ = __add__

    def __sub__(self, other):
        if isinstance(other, _mec_dt_timedelta):
            return self + -other
        if not isinstance(other, _mec_dt_datetime):
            return NotImplemented
        if (self.utcoffset() is None) != (other.utcoffset() is None):
            raise TypeError("can't subtract offset-naive and offset-aware datetimes")
        return _mec_dt_td(self._mec_us() - other._mec_us())


for _mec_dt_cls, _mec_dt_name in ((_mec_dt_timedelta, "timedelta"), (_mec_dt_tzinfo, "tzinfo"),
                                  (_mec_dt_timezone, "timezone"), (_mec_dt_date, "date"), (_mec_dt_time, "time"),
                                  (_mec_dt_datetime, "datetime"), (_mec_dt_IsoCalendarDate, "IsoCalendarDate")):
    _mec_dt_cls.__name__ = _mec_dt_name
    _mec_dt_cls.__qualname__ = _mec_dt_name
    _mec_dt_cls.__module__ = "datetime"

_mec_dt_timedelta.min = _mec_dt_td(-999999999 * 86400000000)
_mec_dt_timedelta.max = _mec_dt_td(1000000000 * 86400000000 - 1)
_mec_dt_timedelta.resolution = _mec_dt_td(1)
_mec_dt_timezone.utc = _mec_dt_UTC
_mec_dt_timezone.min = _mec_dt_timezone(_mec_dt_td(-86340000000))
_mec_dt_timezone.max = _mec_dt_timezone(_mec_dt_td(86340000000))
_mec_dt_date.min = _mec_dt_date(1, 1, 1)
_mec_dt_date.max = _mec_dt_date(9999, 12, 31)
_mec_dt_date.resolution = _mec_dt_td(86400000000)
_mec_dt_time.min = _mec_dt_time(0, 0, 0)
_mec_dt_time.max = _mec_dt_time(23, 59, 59, 999999)
_mec_dt_time.resolution = _mec_dt_td(1)
_mec_dt_datetime.min = _mec_dt_datetime(1, 1, 1)
_mec_dt_datetime.max = _mec_dt_datetime(9999, 12, 31, 23, 59, 59, 999999)
_mec_dt_datetime.resolution = _mec_dt_td(1)


class _mec_dt_module:
    __all__ = ["date", "datetime", "time", "timedelta", "timezone", "tzinfo", "MINYEAR", "MAXYEAR", "UTC"]
    date = _mec_dt_date
    datetime = _mec_dt_datetime
    time = _mec_dt_time
    timedelta = _mec_dt_timedelta
    timezone = _mec_dt_timezone
    tzinfo = _mec_dt_tzinfo
    MINYEAR = 1
    MAXYEAR = 9999
    UTC = _mec_dt_UTC

    def __repr__(self):
        return "<module 'datetime'>"


_mec_mod_datetime = type("module", (_mec_dt_module,), {})()
_mec_mod_datetime.__name__ = "datetime"
