#!/usr/bin/env bash
# A syntax error on the second line of a continued command line, after a here-document
# whose body the parser reads out of order: the report names the line as written and quotes
# it alone, as real bash 5.3 does ("line 10: `  c ) d'"), after "a" ran.
f() { :; }
f <<EOF; echo a
body
EOF
echo b \
  c ) d
echo not reached
