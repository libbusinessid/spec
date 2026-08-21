## Language-specific traps

- **`String.length` counts UTF-16 code units**, so any character outside the
  Basic Multilingual Plane counts twice. The specification reasons in code
  points and bounds input in UTF-8 bytes. Use `codePointCount` where code points
  are meant, and measure the input bound over UTF-8.
- `s.toByteArray(Charsets.UTF_8).size` allocates just to measure length. Compute
  the UTF-8 length without materializing the array.
- An `object` holding a `val Map<...>` runs class initialization on first touch.
  Prefer a generated `when` for dispatch, and `ByteArray` constants for tables.
