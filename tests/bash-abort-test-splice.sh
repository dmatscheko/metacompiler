#!/usr/bin/env bash
# A syntax error on the second line of a continued command line, after a here-document
# whose body the parser reads out of order: the reported position is the line and column
# as written, line 10 column 5 (real bash 5.3 names line 10).
f() { :; }
f <<EOF; echo a
body
EOF
echo b \
  c ) d
echo not reached
