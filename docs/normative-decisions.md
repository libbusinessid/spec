# Normative decisions

This document records every ambiguity of `docs/spec/spec.md` that the
implementation had to resolve, the exact wording that created it, the options
considered and the decision taken. It is a companion of the specification, not a
replacement: `docs/spec/spec.md` stays the normative source, and this file states
how it is applied when it does not fully determine the behaviour.

Each decision is referenced from the code that implements it.

---

## ND-001 - Binary fixtures in the canonical conformance stream

**Wording.** Section 7.2 requires each source file of the canonical stream to be
"UTF-8 sans BOM" with "CRLF et CR" normalized to LF, and states that the
conformance digest "couvre tous les JSONL ainsi que les fixtures incorporées,
avec le même encodage".

**Conflict.** Section 8.4 requires the `load_ruleset` fixtures to be binary
Protobuf payloads read from `testdata/` and embedded byte for byte. A binary
payload is not UTF-8 and applying the CR/CRLF rewriting to it would corrupt it,
which would in turn break the very decoder cases the fixtures exist for.

**Options.**

1. Refuse binary fixtures: impossible, section 8.4 mandates them.
2. Normalize the fixture bytes: corrupts the payload and contradicts "le
   compilateur lit et incorpore les octets".
3. Apply the *framing* of the canonical stream to every entry, and the *content
   normalization* only to text entries.

**Decision.** Option 3. "Avec le même encodage" is read as "with the same stream
framing": the domain separator, the 8 byte big-endian length prefixes, the UTF-8
paths and the byte ordered sort are identical for both kinds of entry. Content
normalization - UTF-8 validation, BOM rejection, CR/CRLF rewriting - applies to
text entries only. A binary fixture contributes its exact bytes.

**Implemented by** `internal/artifact/digest.go` (`SourceEntry.Binary`).

---

## ND-002 - Reported country when the token cannot be normalized

**Wording.** Section 8.4 and `engine.md` 5.8 state that after the dispatcher is
resolved but before a definition is selected, `countryCode` is "le pays
normalisé s'il existe".

**Conflict.** When the explicit country token does not match `[A-Z]{2}` after
trim and upper casing, no normalized country exists, and the specification does
not say whether the result reports the upper cased token, the raw token or
nothing.

**Decision.** The result keeps the **raw context string, unchanged**.
Normalization is only defined for a well formed token; upper casing an arbitrary
string would publish a value that is not a country code, and dropping it would
lose information the caller supplied. This is consistent with the rule already
stated for the states before the dispatcher is resolved, where the country is
"le contexte brut".

An empty or whitespace only token behaves like an absent context.

**Implemented by** `internal/reference/engine.go` (`dispatch`).

---

## ND-003 - Order of the country check and of the pre-canonicalizer

**Wording.** Section 6.11 lists the country normalization at step 3 and the
pre-canonicalizer at step 4, while section 8.4 requires the reported canonical
value to be the **pre-canonical** value as soon as the dispatcher is resolved,
including when the country is rejected at step 3.

**Decision.** The pre-canonicalizer runs as soon as the dispatcher is resolved,
before the country checks. This is observationally identical: a canonicalization
program never fails on user input and never observes the country context (the
`PREPEND_COUNTRY_IF_MISSING` step is forbidden in a pre-canonicalizer). It is
the only order that can report the pre-canonical value for
`unsupported_country`, as section 8.4 requires. Each phase still runs at most
once per public operation.

**Implemented by** `internal/reference/engine.go` (`dispatch`).

---

## ND-004 - Scope of a capture

**Wording.** Section 6.7 lists `capture(name)` among the V1 string constructors
and section 7.5 bounds the number of "captures par format". Nothing states
whether a checksum rule can read a capture declared by a format rule.

**Decision.** A capture is **scoped to the format rule that declares it**. A
checksum rule recomputes the view it needs from `value()` or `subject()`. A
capture is a named view "validée" by the assertions of its own rule, and sharing
mutable analysis state between two independent programs would make the pipeline
harder to reproduce identically in four engines.

In the IR, a capture reference is lowered to a direct node reference inside the
same program; `Program.captures` keeps the name and the node for diagnostics and
for the per format limit.

**Implemented by** `internal/typecheck/expr.go` and `internal/lower/lower.go`.

---

## ND-005 - Indeterminate integers and unevaluable checksums

**Wording.** Section 9.1 of `engine.md` lists the internal value types as
"chaîne, entier borné, booléen, chaîne absente et résultat checksum": there is no
absent integer. Section 6.10 requires a checksum rule to return `valid`,
`invalid` or `unsupported`.

**Conflict.** An integer operation can be impossible to evaluate at runtime: a
non digit character reaching `mod_digits`, a code point outside a mapping, an
index outside a remainder table. Returning an arbitrary integer would risk a
false `invalid`, which section 2.1 forbids.

**Decision.** An integer expression can be **indeterminate**. Indeterminacy
propagates through the integer operations and makes the enclosing checksum node
evaluate to `unsupported` with `unsupported_checksum`. It is never observable as
a value type of the public contract, and it can never produce `invalid`.

**Implemented by** `internal/reference/checksum.go`, documented in
`docs/ir.md` section 1.2.

---

## ND-006 - `weighted_sum` when the operand and the weights differ in length

**Wording.** Section 6.10 requires `weighted_sum` to declare "la séquence des
poids", an alignment among `left`, `right` and `cycle`, a character mapping and
an input slice, without stating what happens when the two lengths differ.

**Decision.** `LEFT` pairs position `i` with `weights[i]`, `RIGHT` pairs the last
position with the last weight, and both pair only `min(len(operand),
len(weights))` positions; the remaining positions contribute nothing. `CYCLE`
pairs position `i` with `weights[i mod len(weights)]` and requires a statically
bounded operand so the sum can be proven not to overflow.

**Implemented by** `internal/reference/checksum.go`, documented in
`docs/ir.md` section 3.2.

---

## ND-007 - `unsupported` when no V1 capability can express a published algorithm

**Wording.** Section 7.4 requires a new capability ID, its implementation in the
four engines, conformance cases and the publication of the engines *before* an
official rule may use a new operation. Section 2.1 requires `unsupported` rather
than `invalid` whenever a conclusion is impossible.

**Situation.** The German VAT check digit follows the iterative MOD 11,10
procedure of DIN ISO 7064. It carries a data dependent state from one digit to
the next, which the V1 arithmetic - weighted sums, digit by digit moduli and
constant remainder tables - cannot express.

**Decision.** The rule declares an explicit `no_checksum` block with the reason
`unsupported_checksum` and the justification. The format is still fully
validated. Adding an iterative primitive would require the whole capability
process of section 7.4 and is therefore out of scope for V1.

**Implemented by** `rules/vat/de.hcl`.

---

## ND-008 - Canonical kinds usable as a symbolic reference

**Wording.** Section 6.11 allows a kind to match `[a-z][a-z0-9_-]{0,63}`, while
section 6.2 requires a dispatcher target to reference its definition as
`identifier.<kind>.<country>`.

**Conflict.** A kind holding a hyphen cannot be written as an HCL traversal.

**Decision.** A **canonical** kind declared in this repository matches
`[a-z][a-z0-9_]{0,63}` so that the symbolic reference stays writable. **Kind
aliases** keep the full `[a-z][a-z0-9_-]{0,63}` grammar, and the engines accept
the full grammar for both, since the restriction belongs to the authoring
language and never to the IR.

**Implemented by** `internal/linker/linker.go`, documented in
`docs/language.md`.

---

## ND-009 - No generated conformance case in the published corpus

**Wording.** Section 8.2 allows the compiler to generate extra robustness or
mutation cases, marked `generated`. Section 17 requires the JSONL corpus and the
binary suite to stay "strictement synchronisés".

**Decision.** The compiler generates **no** case. Generation is optional
("PEUT"), while the strict synchronisation of the two representations is part of
the definition of done. Every robustness and mutation case is authored in JSONL
and reviewed like any other. The `generated` field stays part of the schema and
is accepted by the reader, so a future generator can be introduced without a
format change.

**Implemented by** `internal/conformance/compile.go`.

---

## ND-010 - Billing the produced code points against the evaluation budget

**Wording.** Section 7.5 sets a budget of 100 000 "étapes par validation" without
defining what an step is, and section 12.4 requires that no input cause an
"allocation non bornée".

**Problem found during the adversarial review.** Counting one step per node
evaluation bounds the *number* of operations but not the *size* of the values
they build. A hostile custom bundle can defeat it in two ways:

- a canonicalization `SEQUENCE` whose operand list repeats a `prepend` of a
  4 096 byte constant many times;
- a chain of `CONCAT` nodes where each node concatenates the previous one with
  itself, which doubles the string at every level.

Both stay inside every structural limit and inside the node budget, yet they
make an engine materialize hundreds of megabytes.

**Decision.** An operation that produces a string, and every canonicalization
step, is billed **one further budget unit per started slice of 64 produced code
points**, in addition to its node step. The total number of code points a single
public operation can materialize is therefore bounded by
`MaxStepsPerValidation * 64`, and so is the memory an engine allocates.

The charge is invisible to real rules: an identifier is at most a few dozen code
points, so every rule of this repository consumes exactly one unit per
operation. The rule is normative, published in `docs/ir.md` section 2, and
applies identically to the four engines.

**Implemented by** `internal/reference/exec.go` (`machine.charge`),
`internal/reference/format.go` and `internal/limits/limits.go`
(`CodePointsPerStep`).

---

## ND-011 - The EUID register identifier, checked in membership where a list exists

**Wording.** Commission Implementing Regulation (EU) 2021/1042, which replaced
(EU) 2015/884, builds the EUID from a country code, a **register identifier** and
a registration number. It specifies the *structure* and requires ISO 6523
compliance; it does not enumerate the register identifier values, and no EU wide
table of them is published. The values are national, and there are many per
country: an EUID of the form `DEK1101R.HRB116737` carries the XJustiz court code
of the Amtsgericht Hamburg, so Germany has one identifier per register court, and
France has one per greffe.

**Decision.** Where an authoritative list of a country's register identifiers is
published and obtainable, the rule checks **membership**, so that the country
code and the code that follows it must agree. Where no such list is obtainable,
the rule checks **shape only**, and says so.

The condition is on the list, not on the country. A list that is authoritative
but unobtainable is the same as no list: it cannot be reviewed, and a rule built
on an unverifiable transcription is a rule without provenance.

**Why membership is not the safer default.** An allowlist that is wrong or stale
**refuses a real company**, which is the worst failure this library can produce:
a shape check never rejects a valid EUID, while a missing code does. Membership
is therefore only worth its risk when the list is complete, sourced and
maintained, and each country that adopts it takes on the obligation to re-check
the source.

**France, done.** The registrars publish `Liste des greffes` on data.gouv.fr
under the Open Licence v2.0: 148 codes in the `code_greffe` column, every one
exactly four digits, all distinct, `3102` being Toulouse. The dataset content is
behind an account on the publisher portal, so the codes are transcribed into
`rules/euid/fr.hcl` with their source, authority, licence and access date rather
than fetched by the build. Membership is expressed as an exact length followed by
`PREFIX_IN`: the capture is four digits and every code is four digits, so
starting with one of them is equalling it, and no new capability is needed.

Measured before and after, on inputs the rule used to accept:

```
FRZZZZ.012345674   valid -> invalid, invalid_characters   letters, not digits
FRQ.012345674      valid -> invalid, invalid_length       one digit, not four
FR9999.012345674   valid -> invalid, invalid_format       four digits, no such greffe
```

**Germany, done, and with one limit stated.** The XJustiz code list
`GDS.Gerichte`, published on XRepository by the BLK-AG IT-Standards in der
Justiz, carries 2566 nationwide court codes in version 3.4: 1748 of five
characters and 818 of six. `K1101R` is the centralised Handelsregister of the
Amtsgericht Hamburg, `F1103` the Amtsgericht Charlottenburg.

The list marks no register courts as such, and `K1101R` is not among the forty
five entries whose name says `Registergericht` -- only its name says
`Handelsregister`. Narrowing by name would be our inference rather than the
publisher's statement, and it would refuse a real company the day a register
court is named without the word. So the membership establishes that the code is
**a court**, not that the court keeps a register. That is what the source
supports, and it still refuses everything that is not a court code.

The France trick does not transfer: **782 of the 818 six character codes begin
with a five character code**, so over one list, starting with a code would not
mean being one. The check splits by length, and at a fixed length starting with
a code is equalling it, which keeps an exact membership expressible without a
new operation.

```
DEK1101R.HRB116737   valid     six characters, a real code
DEF1103.HRB12345     valid     five characters, a real code
DEZZZZZ.HRB12345     invalid   no court carries it
DEB1000X.HRB12345    invalid   a real five character code and one more character
```

**The other twenty five stay on shape**, until a list is found for each.

**Implemented by** `rules/euid/fr.hcl`, `format.euid.fr`, the `capture "register"`
checks.

## ND-012 - The source digest domain tag follows the project name

The canonical source stream is prefixed by a domain separation tag, frozen so
that a digest computed today and one computed next year cover the same bytes
under the same label. The tag read `LIBBUSINESSID-SOURCE-V1` and
`LIBBUSINESSID-CONFORMANCE-SOURCE-V1`.

The project was renamed to **entid** before any consumer existed. Keeping the
old tag would have carried a retired brand inside the hash domain forever, in a
constant no reader can interpret and no tool can migrate. Both tags therefore
became `ENTID-SOURCE-V1` and `ENTID-CONFORMANCE-SOURCE-V1`.

**The `-V1` suffix is deliberately unchanged.** It names the version of the
stream construction -- path length, path bytes, content length, content bytes --
and that construction did not move. Only the label did. The old strings are
retired, not coexisting: no released artifact carries a digest under both.

The consequence is measurable and was measured. Every source digest moves, and
with it the `source_digest` field the bundle embeds:

```
entid-rules-2026.08.36.binpb   120872 bytes before and after
                               32 bytes differ, at offset 37..68
                               exactly the source_digest field, nothing else
```

The golden value pinned by `TestSourceDigestIsStable` was recomputed by
`tools/canonical_stream.py`, the second implementation, and the two agree. The
value was never taken from the implementation under test.

**Implemented by** `internal/artifact/digest.go`, `tools/canonical_stream.py`,
spec.md section 7.2.

## ND-013 - The release tag names the rules version, and is checked

The README has always said it: "One release is one `rulesVersion` and one
immutable Git tag `v<rulesVersion>`", two sentences after "Two artifacts must
never share a `rulesVersion`". Nothing enforced the first sentence, so practice
drifted at the first opportunity:

```
v0.1.0   rules 2026.08.32
v0.1.1   rules 2026.08.33
```

Tag and rules version were two independent axes. The release workflow read the
tag from `GITHUB_REF_NAME` and the version from `RULES_VERSION`, and never
compared them. Nothing stopped two tags from delivering one `rulesVersion` --
which is precisely the collision `rules.lock` cannot express, since it
identifies a bundle by that version together with its SHA-256.

**The README wins**, for a reason beyond seniority: the tag ruleset already
makes every `refs/tags/v*` immutable, so binding the tag to the version makes
that immutability enforce the uniqueness invariant instead of merely coexisting
with it. A semantic version would need a second, separate mechanism to say the
same thing.

The consequence is that this repository is versioned by calendar and not by
semver, which fits what it publishes: rule artifacts, not a library. The four
engines are the libraries, and they keep their own versions.

The release now refuses a mismatched tag before it builds anything, and
`TestTheReleaseRefusesATagThatDoesNotNameTheRulesVersion` refuses a workflow
that has lost the check. Both were watched failing.

**Implemented by** `.github/workflows/release.yml`, the "The tag must name the
rules version" step.
