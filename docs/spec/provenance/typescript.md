## Language-specific traps

- **`number` is a double: integers are exact only up to 2^53**, while the IR
  permits `int64` up to about 9.2 x 10^18. Section 7 of the specification is now
  explicit — a generator targeting such a language must emit arbitrary precision
  (`BigInt`) or refuse to generate when an expression can exceed 2^53. It must
  never emit a silently inexact computation. No current rule exceeds 2^53, so
  this binds future rules; make your generator check rather than assume.
- **`String.length` counts UTF-16 code units.** Iterating with `for...of` yields
  code points. The 1 024-byte input bound is UTF-8 bytes, not either of these.
- A module-level `Map` is built when the module loads. Prefer a generated
  `switch`, and `Uint8Array` for tables.
- Avoid classes in the generated code; plain functions optimize better and keep
  the output diffable.
