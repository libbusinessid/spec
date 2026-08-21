## Language-specific traps

- **`String.count` counts grapheme clusters, not code points.** The
  specification reasons in code points and bounds input in UTF-8 bytes. Use
  `s.unicodeScalars` or work over `s.utf8` — a naive `.count` diverges on
  combining sequences and will pass every ASCII test while being wrong.
- The 1 024-byte input bound is UTF-8 bytes: `s.utf8.count`.
- A `static let` dictionary is lazily initialized but still allocates on first
  use. Prefer a `switch` for dispatch, and `StaticString` or fixed-size arrays
  for tables.
- Mark generated hot paths `@inlinable` so they inline across module boundaries.
