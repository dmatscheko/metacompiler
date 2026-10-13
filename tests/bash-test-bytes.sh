#!/usr/bin/env bash
# printf and echo -e WRITE the NUL bytes their escapes make (\0, \0NNN, \x00, \u0000,
# and %c of an empty operand), and a width counts one as a character. A shell string
# cannot hold one, so nothing inside a program can observe it: tests/native-full.sh
# compares the native binary's stdout with tests/bash-bytes.expected (real bash 5.3's
# output), and the matrix and --cross hold the two halves to the same bytes.
echo -e '\0'X
printf '\0'
printf 'a\0b\n'
printf '%s\0' x y
echo -e 'a\0b' 'a\00101b'
printf '%b|%b\n' 'p\0q' 'r\0101s'
printf 'c\x00d|\u0000u\n'
printf '[%c][%3c][%-3c]\n' '' '' ''
printf '[%5b][%-5b][%.2b]\n' 'a\0b' 'a\0b' 'a\0b'
printf '\0%s\0\n' mid
printf '\x01|\001|%b\n' '\x01'
echo -e 'e\0\c' 'f'
echo
