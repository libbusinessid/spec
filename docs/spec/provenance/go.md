## Language-specific traps

- A package-level `var t = map[string]X{...}` or `[]byte{...}` is **initialized
  at program start-up** and allocates. A `const t = "\x01\x02..."` indexed as
  `t[i]` lives in `.rodata` and costs nothing. Prefer a generated `switch` over
  a map for dispatch.
- `len(s)` on a `string` already counts UTF-8 bytes, which is what the 1 024-byte
  input bound means. This is the one language where the natural expression is the
  correct one — do not "fix" it into a rune count.
- Returning `string(buf[:n])` allocates. Return the original input unchanged when
  canonicalization is a no-op.
