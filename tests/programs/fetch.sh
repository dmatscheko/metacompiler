#!/usr/bin/env bash
#
# fetch.sh - download the pinned third-party programs that activity E of
# docs/master-prompt-find-tickets.md runs through tests/probe.sh.
#
#   tests/programs/fetch.sh            fetch and verify everything
#   tests/programs/fetch.sh --list     print the table, fetch nothing
#
# Everything lands in tests/programs/.cache/ (git-ignored). Downloaded programs
# are NEVER committed. Every download and every extracted program is checked
# against the SHA-256 recorded here; a mismatch deletes the file and fails.
#
# THIS SCRIPT ONLY DOWNLOADS AND EXTRACTS. It runs nothing it fetched, and it
# must stay that way. Downloaded code is untrusted: read a program before you run
# it, run only a scratch COPY (mktemp -d), with a timeout and no network, and
# never run a downloaded build script.
#
# The programs are not in our dialects as fetched. A probe needs a minimal,
# recorded adaptation (the oracle runs the SAME adapted file): a small N in place
# of argv, text output in place of binary PBM bytes, println in place of
# console.log for js, an entry class named Main for java, and so on. The
# per-program notes below say what the first run needed.
#
# SOURCES AND LICENSES
#
# 1. The Computer Language Benchmarks Game, git commit
#    40296663ed350d5fe4a6ab5e367bab61cb77c219 (2025-03-07) of
#    https://salsa.debian.org/benchmarksgame-team/benchmarksgame
#    - the source zip public/download/benchmarksgame-sourcecode.zip at that
#    commit, entries extracted by name. License: 3-Clause BSD ("Revised BSD"),
#    LICENSE.md at that commit, (c) Brent Fulgham, Isaac Gouy and contributors.
#    The "-8" programs are the naive single-threaded transliterations.
#
# 2. Rosetta Code task pages, pinned by PAGE REVISION ID (oldid), fetched as raw
#    wikitext. One program = code block N (0-based, <syntaxhighlight> or <lang>)
#    of the page's =={{header|LANG}}== section, which the extractor below
#    selects deterministically. License: GNU Free Documentation License 1.3
#    (https://www.gnu.org/licenses/fdl-1.3.html), per rosettacode.org.
#
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
cache="$here/.cache"
list_only=0
[ "${1:-}" = "--list" ] && list_only=1

sha() { shasum -a 256 "$1" | cut -d' ' -f1; }

# fetch URL SHA256 DEST
fetch() {
    local url=$1 want=$2 dest=$3
    if [ ! -f "$dest" ]; then
        mkdir -p "$(dirname "$dest")"
        curl -sfL --max-time 120 -A "metacompiler-fetch/1.0" -o "$dest.part" "$url"
        mv "$dest.part" "$dest"
    fi
    local got; got=$(sha "$dest")
    if [ "$got" != "$want" ]; then
        rm -f "$dest"
        echo "fetch.sh: SHA-256 MISMATCH for $url" >&2
        echo "  want $want" >&2
        echo "  got  $got (file removed)" >&2
        exit 1
    fi
}

# check FILE SHA256 - an extracted program
check() {
    local got; got=$(sha "$1")
    if [ "$got" != "$2" ]; then
        rm -f "$1"
        echo "fetch.sh: SHA-256 MISMATCH for extracted $1 (want $2, got $got; removed)" >&2
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# 1. Benchmarks Game
# ---------------------------------------------------------------------------
BG_COMMIT=40296663ed350d5fe4a6ab5e367bab61cb77c219
BG_ZIP_URL="https://salsa.debian.org/benchmarksgame-team/benchmarksgame/-/raw/$BG_COMMIT/public/download/benchmarksgame-sourcecode.zip"
BG_ZIP_SHA=aabcf6726cdc14f0f45b99e5daba48584f94bbb48883fd3711a1d040474d1cb4

# zip entry | SHA-256 | adaptation the first run needed
BG_PROGRAMS='
nbody/nbody.python3-8.python3                b0adb5671e270b37d1f398bfb8e77ad15bdc2d5898315a0d7e844ae5297088ab  argv->1000; main->bg_main
spectralnorm/spectralnorm.python3-8.python3  008630d2ec8f650d813004a250970305b51ac6d2773cbdd84f059e02f398894b  argv->50; main->bg_main
fannkuchredux/fannkuchredux.python3-8.python3 fad3eec38b9b516c4c0e8b49b2466d8f689d912f08b3d72bc558b051721fb2af argv->7
fasta/fasta.python3-8.python3                d8412fe8f6fa285c014158a2ffd2f3718578177edd4a7f462c9c51925133d2d1  argv->250; main->bg_main
mandelbrot/mandelbrot.python3-8.python3      4fd93dceb9dab51b11ed25821ba93c346b7296b93c2bf73f69a6726d06645601  CRLF file; argv->20; byte write->print(int); main->bg_main
binarytrees/binarytrees.python3              4eba41e47c256a732446527a2889eaf7e88bb959ce250905eb5d98fa48fb6b20  multiprocessing removed (map); argv->6; main->bg_main
nbody/nbody.go-8.go                          a684e611ada6c41073ddf7f4e0d01e35632f4f34b3b67064a811049fbbd86d4b  n->1000
spectralnorm/spectralnorm.go-8.go            8f277cc7c9f4b0cb287c9bd697d047fc8d40562d960030d9c718a948073dd6ab  N->50
fannkuchredux/fannkuchredux.go-8.go          b2f253a103bb797863b132e10b522034451f395bca02ec470a3e98c2031f742b  none
fasta/fasta.go-8.go                          2dace805868f891c8ff406bcc4f2a0433752b2ef1ce946490ce063180941a5d3  n->250
mandelbrot/mandelbrot.go-8.go                5c738174a4d392c995978c4d5121785c738b36347fb75608f70e52a8ca83c157  n->20; WriteByte->Fprintln
binarytrees/binarytrees.go                   72d31464e6ef9df69c740c364e18099be00dbdca22a12bd04ed919f7e664e6c3  n->6
nbody/nbody.node-8.node                      5b532337e1eff729e239b78b63c62c53cb8c4e7144fa494515be961ce0f37ea3  console.log->println; entry in main()
spectralnorm/spectralnorm.node-8.node        099d40f182f0867290ffb40a51f9b2cce739cf2a76995189d5b6eea4c3b10f6e  console.log->println; argv->50; entry in main()
fannkuchredux/fannkuchredux.node-8.node      c5d480a9624506b92482105f2681d8f3749f9cafd06b3d561d471be8e32c1894  console.log->println; argv->7; entry in main()
fasta/fasta.node-8.node                      475875e8235e3be7dea5a05e930ff929cae5b327966f7b5c0635c78c094c19b5  process.stdout.write->print; argv->250; entry in main()
mandelbrot/mandelbrot.node-8.node            017d67c23f93540b0352d736da91edaba06948ac27cb60c8647dda0408164e35  needs Uint8Array
nbody/nbody.java-8.java                      dd4b1141d10a86a1b87e827ff894e0c73e07552630ffa2320e111dcef3d0a36f  CRLF file; class->Main; argv->1000
spectralnorm/spectralnorm.java-8.java        3b9fb7ad8840638e4193e91a6c46209756faa72a14fa0fb5d9aba4dd33f06a7f  class->Main; argv->50
fannkuchredux/fannkuchredux.java-8.java      546f1d8bdb8d19eb739bc18d23bb2f6d8d1c302ec596445f2e363ab190de198b  class->Main
fasta/fasta.java-8.java                      fa62024eeabe389d3eb809fded18cba2eda06bafb336b406548113f1fc5e22a4  class->Main; argv->250
mandelbrot/mandelbrot.java-8.java            8a8d0cb3f7fd0fcaa202ed6bddaf71e08b1e79ea28e12ad6394e7191a9ee5afb  class->Main; argv->20; write->println
nbody/nbody.lua                              aaaecfb8847e9609a6d0a6b95a0d19506b208c579abba62857a47a00ffef7575  tonumber(arg..)->1000
spectralnorm/spectralnorm.lua                1acdfef437c9cae18f1dfb8394acc9144028d3e38ca4c581f7d699294fe81fed  tonumber(arg..)->50
fannkuchredux/fannkuchredux.lua              e6db90f101bafdfc2f213ce700d247e9b719c23a3782fc2c843c39ea5f1b157a  tonumber(arg..)->7
binarytrees/binarytrees.lua-2.lua            58afb23db343d5c59e0c23b9d8b6188dab41fc378b0e588f965c0d24000173ed  tonumber(arg..)->6
nbody/nbody.gcc-8.gcc                        e6e067338140f86ec3aca2b43e2770f3dbe6c7c2d6108934671c58f00af13317  printf-based: compiler legs only
spectralnorm/spectralnorm.gcc-8.gcc          5ae8e0c4d2cb97b744d682e4e974a22b727b5209ecc3293d4d1581ccf86d7574  uses VLAs (outside the C subset)
fannkuchredux/fannkuchredux.gcc-8.gcc        65c8d4b7b5697e1ee1ccff75aec8fb66ae97811fbd96b02a0ef46941c49dea97  uses VLAs (outside the C subset)
fasta/fasta.gcc-8.gcc                        6bcc35ab71b391fe3c96263b0f8b4bf47a67aa965d2a85117ae80e9d342784ed  n->250
mandelbrot/mandelbrot.gcc-8.gcc              c785a5031762cb7818d4b3ca595dbce90c68279763aebd5625c0ef4956943246  argv->20; putc->printf
nbody/nbody.swift-8.swift                    13c6883527b990d977b9257397512a054d418883421e74fd8e82698f9337b1c6  Glibc->Foundation; argv->1000
spectralnorm/spectralnorm.swift-8.swift      d9fb6ddd98c6e32e9a9cc1b06cab7fa4484530bedd6c4e4e8c247e4655bc1125  argv->50
fannkuchredux/fannkuchredux.swift-8.swift    e0e3ffb41e50c953dfe60949e538523dd67b2060cdc40cfc074a0db98f036913  argv->7
fasta/fasta.swift-8.swift                    b3f2aa88e94a8d387bc42da98295d43913137319443c5d1c43593d99053f70db  argv->250
mandelbrot/mandelbrot.swift-8.swift          434d40f9b6c4923cb8cde206c2fd11ccdccb9e3a17f434826bc5c6e11287094e  Glibc->Foundation; argv->20; putc->print
nbody/nbody.ruby-8.ruby                      952b9bf363f49f97daad7562516a016487290e3d733a71ed9ab18e818a553630  ARGV->1000
spectralnorm/spectralnorm.ruby-8.ruby        fe548f0ad23d64b495928d3e78eec236e28bd4784ddeb4d635191d932af36706  ARGV->50; main->bg_main
fannkuchredux/fannkuchredux.ruby-8.ruby      45992fea487848a981b03fa1f65eb5dc13221043b937b9716b4019041052f782  ARGV->7
mandelbrot/mandelbrot.ruby-8.ruby            5264c2373c346c6aa1921d473e8c93a8c2c16493e220f46ff3f36990bb77d218  ARGV->20; .chr->puts; main->bg_main
'

# ---------------------------------------------------------------------------
# 2. Rosetta Code
# ---------------------------------------------------------------------------
# page title | revision id (oldid) | SHA-256 of the raw wikitext
RC_PAGES='
100_doors                        416441 e6fef1f21e836056df24cb8b1f78e7581e1c419d299eb6ce8ec4ead8a9636e44
Ackermann_function               415618 a62bf39b01613d8429b3f68c97d4bf166dff92d9147d86a8cb42cce1e5b559ec
Gray_code                        415507 2bb6a860b85bd6d66e4e40451c31893c1a05267de9e050717315b713b5c3e3db
Hailstone_sequence               415511 fca88f84b728462a467377c9ac639b240dbf33add826c7199b7bd7423fa7180a
Josephus_problem                 414279 9d6b8c8e3adb577355e3c6b9f4a63512e878667511ad689dda555c9b974f142f
Levenshtein_distance             416654 89e3bfa413fe93245a4957168c95721b4d0be0d37ec61ce1c1b502d4ca4f7560
Look-and-say_sequence            415918 788315cd548e369d82b25a4303675a672c0178863857c0db127988206ec3ba49
Luhn_test_of_credit_card_numbers 414317 5e9179ccf3803bcfba4f843285feb11826422dd3a450eef22b5a9b5aaaf9eeb8
Mandelbrot_set                   416985 22130dfd46682629bb0658d4939223681c3c93215e9a3f3be7b27726dccf2df0
N-queens_problem                 417025 ee6d12d0ee103b907d6b4f041c11cb279ad35b9f9402df2900b3a7241469f3c6
Roman_numerals/Encode            416271 94122938a40b7b24043b62a9349df9e927666bd0b0c4a749689eb6fb333dbefd
Sieve_of_Eratosthenes            416982 710993f39d206a830d7e1c55c110cf8ded610f2634505b023525823e6a9a7fce
Towers_of_Hanoi                  416983 02ce4a179150c1912a1a504efb31ac28cab28a7c8d223ca9989f945ee69f5024
'

# output file | page | header | block | SHA-256 of the extracted program | adaptation
RC_PROGRAMS='
python/doors.py           100_doors                        Python 0 29e4139177b4633c7c1b9ae17ff0e241bb95ba38f6d77e22465bda4f0e75b576 none
python/ackermann.py       Ackermann_function               Python 1 8d0de70cdcc72bd9b8036e8ac8e8e278f85bfca9b3bbe11cebae040b169be5f1 append a print loop
python/gray.py            Gray_code                        Python 0 b6677b341480cd18853b3a9a2f7b32d18e649966d25a15cde9ae535c415d42c9 none
python/hailstone.py       Hailstone_sequence               Python 2 6108eca2f5777f453233ef390b8ce32b335b583879d0d0ff7008710fa2fd5e16 main->rc_main; 99999->999
python/josephus.py        Josephus_problem                 Python 2 6ccda4a5a81142d77f30b1778eda8ea037c863649d96f2765423af5ad7702d23 print() the two calls
python/levenshtein.py     Levenshtein_distance             Python 4 95deb4f893c42c8713f90e39ce525141c98c3f1c63d8d4d9b6be90c2d429a2fc main->rc_main
python/lookandsay.py      Look-and-say_sequence            Python 0 990168db2dc6500f21f8719bce45ce6ca30a98de08cd48277ff656de3866a2e2 py2 print->print()
python/luhn.py            Luhn_test_of_credit_card_numbers Python 1 41ccbc4d1bc4ec3d4ac5cd29ecf08024e74590dbaa13bd06c5974b7de9b3fd2c main->rc_main
python/mandelbrot.py      Mandelbrot_set                   Python 0 8f1d7482d1eec9c6377db012e3ee2533b7a5fc5d2054edf94e5f254f146068cf none
python/nqueens.py         N-queens_problem                 Python 7 6ccab7bd44c5434a463b9abae572d5a50b641f1b2dc39b1b5b4fdd325f1eb30d main->rc_main (incl. main.__doc__)
python/roman.py           Roman_numerals/Encode            Python 7 7d1debdad335da088cf16580be10af4ac0f14fef361e8e3184e0f489ec67715d main->rc_main
python/sieve.py           Sieve_of_Eratosthenes            Python 6 7daeb2bd00e305a7101f303c37715ff7596b55edb0f4d080944a12c01ac61a62 append print(list(primes235(300)))
python/hanoi.py           Towers_of_Hanoi                  Python 1 67aca664bd90eb0042664053f68e17d96ce7d30610d9b8e6872ced092520c9af none
go/doors/main.go          100_doors                        Go 0 5be243897e8c6ae6ed13b454303c6994baa9042b95b8dd61aaf6a50638146b95 none
go/gray/main.go           Gray_code                        Go 0 d04d1d45340600e6681b5ee55a604cacdf98cbc453e91d1626b04674aebdf627 none
go/hailstone/main.go      Hailstone_sequence               Go 0 4d4d44795bfe8b9d8b013cd11a5ead477d78c7266dd47ac99b7cf6b1c4640896 100000->1000
go/hanoi/main.go          Towers_of_Hanoi                  Go 0 3ea0223a5a81bfbd3bf0c02dc0af285c8ebf289d8a18023cd44747cd9bcfbb64 none
go/josephus/main.go       Josephus_problem                 Go 0 5b6d7df67312ffeacb62cc751ce067315939c17f8cc91b5f16d1d4bcf798134c none
go/levenshtein/main.go    Levenshtein_distance             Go 0 d4a45809741bc4fbd3fd25a7143771d6d51f3e14e7805fcc1325c4d1207e019a none
go/lookandsay/main.go     Look-and-say_sequence            Go 0 4e15d2f87c23d250b5f611c82e63868ffcf8566690a1f151448850f0e5505c88 none
go/luhn/main.go           Luhn_test_of_credit_card_numbers Go 0 91bb02f00a244fd3f80ab82f6b76bc7f05c4f54b91a381d2d478db772422d4a9 none
go/mandelbrot/main.go     Mandelbrot_set                   Go 0 cb745fdf09424424fbcaa7d77fb404871a87e18815bdd40ba90dad2285bf1d42 needs math/cmplx
go/nqueens/main.go        N-queens_problem                 Go 0 cb5d7213937b875a60144e17dfccb6654aaaf8e1bfe86db5d539ffccad4ee983 none
go/roman/main.go          Roman_numerals/Encode            Go 0 4d72fc5eb6baaed2b57d9181917bccdb3b8b8e49c07e6231e46c91dbbf6fda5a none
go/sieve/main.go          Sieve_of_Eratosthenes            Go 0 78a0b8e6c626c80ca36c9418ea120aef99d39623c96460e43c8afd292243309f none
lua/ackermann.lua         Ackermann_function               Lua 0 97e4553ed9f37078c8d34ccc6db81208e0cb04b74e59e58c87bffebfcf00fd30 append a print loop
lua/hailstone.lua         Hailstone_sequence               Lua 0 beaa3e3145e123ff51014f48590209fd2c7ef0788d9f063acbf7e9956511b28b 100000->1000
lua/hanoi.lua             Towers_of_Hanoi                  Lua 0 0305985bfe74dc2e0dbde2db31dadfd59de89196de3e0d23f5c3ed4b997cc991 none
lua/josephus.lua          Josephus_problem                 Lua 0 e089900dde4f5bcc0b3eaa1812924c430152ef082d565d152f11ab329b5b0aef none
lua/levenshtein.lua       Levenshtein_distance             Lua 0 71f1a91ef4322e6f2d04bd4b6b7c8e0aa669eb37379ebdeb4ab7a964a49fb0e6 none (exponential: slow in interp)
lua/lookandsay.lua        Look-and-say_sequence            Lua 0 04a96a1cf36919c1fce81102ecf3b0aee2b37f6d86006b921daaee21db7ceeef none
lua/luhn.lua              Luhn_test_of_credit_card_numbers Lua 0 1bd47127cdd9a0ac1c4df941c6ab5ec8ebcb9d8c5087471ae5006135bef2c236 append block 2 (the test loop)
lua/mandelbrot.lua        Mandelbrot_set                   Lua 1 49fd9224e8f59401d9bc9c1c61b768faf5c05d4b76ecf5f4421d464e9280bf09 none
lua/nqueens.lua           N-queens_problem                 Lua 0 daacbdbffac05bdb7ca7230458d5bf307f637e805fd5d7cc1f8d84b058e13492 none
lua/roman.lua             Roman_numerals/Encode            Lua 0 1053475f12293dc65b8ae2aa7edc63950b25a69c03e8466ada9857be5264966c io.read->literal list; unpack->table.unpack
lua/sieve.lua             Sieve_of_Eratosthenes            Lua 0 63ca8ce564cc8b98e7e54a007b55cd6132c63a010855257f09fefb2392982298 append a print
'

# The block extractor: deterministic, reads the verified wikitext only.
extract() { # WIKI HEADER INDEX OUT
    python3 -I - "$@" <<'PY'
import re, sys
wiki, lang, idx, out = sys.argv[1], sys.argv[2], int(sys.argv[3]), sys.argv[4]
text = open(wiki, encoding="utf-8").read()
hdr = re.compile(r"^={2,3}\s*\{\{header\|([^}|]+)")
sec, on = [], False
for ln in text.split("\n"):
    m = hdr.match(ln)
    if m:
        if on:
            break
        on = m.group(1).strip().lower() == lang.lower()
        continue
    if on:
        sec.append(ln)
blocks = re.findall(r"<(?:syntaxhighlight|lang)\b[^>]*>(.*?)</(?:syntaxhighlight|lang)>",
                    "\n".join(sec), re.S)
open(out, "w", encoding="utf-8").write(blocks[idx].strip("\n") + "\n")
PY
}

if [ $list_only = 1 ]; then
    echo "Benchmarks Game @ $BG_COMMIT (3-Clause BSD):"
    echo "$BG_PROGRAMS" | awk 'NF { printf "  %-46s %s\n", $1, substr($0, index($0,$3)) }'
    echo "Rosetta Code (GFDL 1.3):"
    echo "$RC_PROGRAMS" | awk 'NF { printf "  %-22s %s rev-pinned, %s block %s\n", $1, $2, $3, $4 }'
    exit 0
fi

# --- Benchmarks Game
zip="$cache/benchmarksgame/sourcecode-$BG_COMMIT.zip"
fetch "$BG_ZIP_URL" "$BG_ZIP_SHA" "$zip"
n=0
while read -r entry want _; do
    [ -z "$entry" ] && continue
    dest="$cache/benchmarksgame/$entry"
    mkdir -p "$(dirname "$dest")"
    unzip -p "$zip" "$entry" > "$dest"
    check "$dest" "$want"
    n=$((n + 1))
done <<< "$BG_PROGRAMS"
echo "benchmarksgame: $n programs verified in $cache/benchmarksgame"

# --- Rosetta Code
rev_of() { echo "$RC_PAGES" | awk -v p="$1" '$1 == p { print $2 }'; }
while read -r page rev want; do
    [ -z "$page" ] && continue
    fetch "https://rosettacode.org/w/index.php?title=$page&action=raw&oldid=$rev" "$want" \
          "$cache/rosettacode/pages/${page//\//_}.$rev.wiki"
done <<< "$RC_PAGES"
n=0
while read -r out page lang idx want _; do
    [ -z "$out" ] && continue
    dest="$cache/rosettacode/$out"
    mkdir -p "$(dirname "$dest")"
    extract "$cache/rosettacode/pages/${page//\//_}.$(rev_of "$page").wiki" "$lang" "$idx" "$dest"
    if [ "$want" = "@" ]; then
        echo "  $out $(sha "$dest")   (no SHA recorded yet)"
    else
        check "$dest" "$want"
    fi
    n=$((n + 1))
done <<< "$RC_PROGRAMS"
echo "rosettacode: $n programs extracted in $cache/rosettacode"
