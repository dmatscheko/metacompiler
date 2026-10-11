-- print writes the BYTES a Lua string holds, valid UTF-8 or not (lua 5.5's
-- lua_writestring is fwrite). Nothing inside a program can observe its own
-- output, so this file is checked from outside: tests/native-full.sh compares the
-- native binary's stdout with tests/lua-bytes.expected (real lua 5.5's output),
-- and the matrix and --cross hold the two halves to the same bytes.
local ss = {"\xff\xfe", "caf\xc3\xa9", "\xe2\x82", "\u{1F600}", "a\0b",
            string.char(200, 65, 128), "\xc0\x80", "\xed\xa0\x80", "plain"}
for i = 1, #ss do
  print(i, ss[i], #ss[i])
end
print(table.concat(ss, "|"))
print(string.char(255) .. "x", 1.5, nil, true)
