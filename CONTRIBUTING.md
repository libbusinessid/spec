# Contributing

Thank you for helping LibBusinessID. This repository is the source of truth of
the project: the rules, the IR, the conformance suite and the compiler all live
here, and every engine consumes what is published from here.

Read `DATA_POLICY.md` before touching `conformance/**` or `testdata/**`.

## Getting started

```bash
make tools        # build the version locked generation tools
make generate     # regenerate the Protobuf code and the decoder fixtures
make test         # unit, integration and conformance tests
make verify       # run the whole conformance suite against the reference interpreter
make compile      # rebuild every artifact under dist/
make ci           # everything the pull request pipeline runs
```

The repository works offline once the dependencies are downloaded.

## Non negotiable principles

1. **A false negative is the most serious defect of the project.** A value is
   declared `invalid` only when a documented and applicable rule proves it. An
   unknown rule, an unpublished algorithm, an ambiguous variant or an incomplete
   coverage always produce `unsupported`.
2. **One source, several idiomatic engines.** The rules and the conformance
   cases are shared; the engines stay independent.
3. **Closed execution on incompatibility.** An unknown version, capability or
   opcode closes the execution. Nothing is ever partially executed.
4. **Observable determinism.** For the same bundle, input and context, every
   engine produces the same canonical value, the same executed steps, the same
   statuses and the same reason codes.
5. **No network dependency.** Compiling the rules and running the conformance
   suite work offline.

## Test driven development

The parser, the linker, the typechecker, the lowering, the bundle validator and
the reference interpreter are developed test first. A new IR operation starts
with, in this order: the nominal case, the bounds, the error, the security case,
then the cross language conformance case.

Coverage thresholds are enforced by `make cover`: at least 95 % of lines and
90 % of blocks, with 100 % of the IR operations and of the bundle rejection
paths covered.

## Adding or changing a rule

A rule pull request is only admissible when it contains:

- **an entry in [`docs/identifier-atlas.md`](docs/identifier-atlas.md)**, moving
  the identifier from whatever status it held to `supported` with its maturity
  and case count. `businessidc lint` refuses a definition the atlas does not
  name (LINT006), because a rule nobody can find is a rule that does not help
  anyone;

- **the official source**, or a documented justification when no official source
  exists, recorded in a `source` block with its URL, authority, title, access
  date, jurisdiction, language, notes and applicable terms;
- **valid and invalid conformance cases**, following the minimum matrix: at
  least two independent valid examples when available, a too short and a too
  long value, invalid characters at each position class, a valid checksum, a
  mutation of each check digit, the separators and the letter case accepted by
  canonicalization, the empty value, a missing and a contradictory country when
  relevant, both profiles when they differ, the `unsupported` branch when no
  checksum is published, the exact bounds of every range and each branch of a
  multi variant algorithm;
- **an analysis of the false negative risk**;
- **the output of `businessidc diff`** between the previous and the new bundle;
- **the provenance update**;
- **a fully green conformance run**;
- **no unintended restriction** of an existing format.

A restriction is a high risk change and requires a reinforced review. A
documented widening that fixes a false negative can be published as a rule
patch.

### Which identifiers belong here

An identifier is admissible when an authority issues it and publishes something
about its form. That is what makes a rule checkable and a source citable, and
without it there is nothing to be right or wrong about.

A validator with no issuer behind it is not a weak rule, it is a different kind
of thing: it reports `valid` on the strength of nobody. The caller reads that as
a verdict from a register, and it is a verdict from a character class. The
project would rather answer nothing than answer that.

This is why `nationalregistration` was not carried over from the implementation
this repository replaces. It accepted any value of at most 64 characters drawn
from `[A-Z0-9./ -+]`, for no named register, and it required a country code
whose absence it reported through a reason code the specification does not
define. Adding that reason code would have meant a new frozen capability - a
permanent commitment - in support of a rule with no source. A national
registration number is welcome here as a rule per jurisdiction, each with the
register that issues it.

Two things that are *not* grounds for refusal:

- **The identifier also identifies a natural person.** A sole trader invoices
  under a personal number in several countries, so refusing that form refuses a
  legitimate business identifier. What a rule accepts and what the corpus stores
  are separate questions; the second is settled by `DATA_POLICY.md`.
- **The issuer publishes no check algorithm.** That is what
  `checksum_not_published` is for. A format rule with no checksum is still a
  rule, as long as the format itself is published.

### Identifiers that contain other identifiers

When one identifier contains another, build the **inner** validator first, prove
it, then compose the outer one from it. Never restate the inner rule inside the
outer one.

- **SIRET** = SIREN (9) + NIC (5). `checksum.fr.siret` applies
  `checksum.fr.siren` to the nine leading digits and adds the Luhn over
  fourteen.
- **EUID** = country code + register identifier + `.` + the national number.
  `format.euid.fr` reuses `format.fr.siren` through `use_format`, and
  `checksum.euid.fr` reuses `checksum.fr.siren` through `apply_checksum`.

Two reasons, and the second is the one that is easy to miss.

Two copies of a rule drift, and the drift is silent: the corpus of one
identifier does not exercise the other, so nothing fails when they diverge.

Composition also **strengthens** what is checked, because the inner check is
rarely implied by the outer one. Luhn over nine positions and Luhn over fourteen
double the opposite positions, so a valid SIRET does not mathematically imply a
valid SIREN: measured on 50 000 random Luhn-valid SIRET, **90.2%** carry nine
leading digits that fail their own check, and the uncomposed rule accepted every
one of them.

That is also why composition is a **restriction** and needs the strongest
evidence available before it is merged. Here the whole INSEE stock was checked
first — 43 896 818 establishments, none with a SIREN that fails — and then the
composed rule was swept over the same register. Do the same for any other
jurisdiction whose number nests one inside another: the inner rule first, with
its own sources, its own corpus and ideally its own sweep.

### Which historical variants to accept

Section 2.6 of the specification gives the criterion: usage, never the issuing
date. A variant stays supported as long as a conforming value can legitimately
appear in data processed today, including data whose subject no longer exists —
an identifier outlives the company in invoices and archives.

Removing a variant therefore needs a source attesting that it stopped
circulating, while adding one needs a source attesting that it exists. Without
proof, the variant is supported. A variant that is no longer issued but still
circulates belongs to `compatible` and may be refused by `strict_current`; that
profile split is the only normative way to mark a variant as historical.

### Never write an identifier from memory

Every real identifier in the corpus must be **fetched**, in the session that
adds it, from the issuer's own register or a reliable business directory. Not
recalled, not reconstructed, not adapted from one that looks similar.

In order of preference:

1. **The issuer's register**, queried live — Companies House, the NTA lookup
   site, the INSEE bulk files, a national trade register.
2. **The issuer's own documentation**, when it carries a worked example — the
   NTA check-digit note builds `8700110005901`, the Receita Federal announced
   `00.000.000/E08G-12` as the first alphanumeric CNPJ.
3. **A reliable business directory** — Pappers, OpenCorporates and equivalents,
   when the register itself is not reachable.

Keep the URL and cite it in `sourceIds` and `redistributionBasis`.

Synthetic values remain welcome and remain synthetic: generated per
`DATA_POLICY.md` section 4, marked `dataClassification: synthetic`, never
presented as real. If a value cannot be fetched, use a synthetic one and say so.

The reason is not pedantry. A remembered identifier looks right and does not
exist. Adding China, Japan and Brazil, seven such numbers were used before being
checked — five Japanese, two Chinese — and every one was wrong while the
algorithms they were meant to verify were correct. That is worse than a failing
test: a plausible identifier that happens to validate makes a broken rule look
proven. **When an algorithm disagrees with a number, suspect the number first.**

Never write an expected result by running the reference interpreter. Expectations
come from the published algorithm, from an independent implementation such as
`tools/vectors.py`, or from an authority document.

Two independent implementations guard the artifacts:

- `tools/vectors.py` re-implements every published check algorithm from its
  description, and the JSONL corpus is authored from its output by
  `tools/gen_*.py`;
- `tools/canonical_stream.py` re-implements the canonical source stream of
  `spec.md` section 7.2, and the CI compares its digests with the ones the Go
  compiler writes into the manifest.

## Operations no rule exercises

The generated coverage document lists every operation the IR defines and no
rule emits, and `TestUnusedOperationsAreTheOnesWeKnowAbout` holds that list to
what `knownUnusedOperations` says.

An engine compiles the bundle into native code, so it never generates an
operation no rule carries — but it must still implement one, in case a later
bundle does, and no conformance case proves that implementation. The list is
that untested surface, and the capability freeze makes it permanent once a
release exists.

The test fails in both directions, on purpose:

- **An operation appears that nothing uses.** Either write a rule that needs it,
  or add it to the list with a reason. Widening what every engine owes with no
  evidence behind it should be a decision, not a side effect.
- **A rule starts using one.** Remove the entry. The list should say what is
  true.

`prefix_in` is why this exists: it sat there fully implemented across proto,
catalogue, type checker and interpreter while the UK company number rule spelled
out forty one `starts_with` by hand, and nothing said so.

## Changing the core

A pull request touching the language, the IR or the pipeline must answer:

- which shared semantics does it affect?
- can the four engines implement it without divergence?
- which false negative risk does it introduce?
- which bounds and hostile inputs are tested?
- must the conformance suite evolve?
- is the IR capability already reserved and supported by the engines?
- does the public contract stay compatible?

A new IR operation requires, in this order: a documented capability ID, its
implementation in the four engines, conformance cases, the publication of the
compatible engines, and only then its use by an official rule.

## Bumping a pinned generator

The generated files record which plugin produced them, so a bump of a
generator pinned in `tools/pinned/go.mod` makes the committed generated code
stale and the CI refuses it. That refusal is the point: it is what ties the
committed code to the locked tool versions. Never filter the version comment
out of the comparison.

Instead, in the same pull request:

- run `make generate` and commit the result;
- rebuild the artifacts and state in the pull request whether the published
  digests moved, as section 12.3 of the specification requires. A
  `protoc-gen-go` bump normally leaves the wire bytes untouched, but that is
  something to verify, not to assume.

## Conformance of an engine

Never reimplement the conformance suite inside an engine. An engine that reads
the corpus and interprets the expectations itself can pass by comparing too
weakly, and that failure is invisible.

Run `make conformance`, which drives the reference testee through the protocol of
specification section 8.7. To qualify another engine, point the runner at its
testee:

```sh
go run ./cmd/conformance-runner --corpus dist/businessid-conformance-<version>.binpb -- ./testee
```

A run restricted with `--operation` is a diagnosis aid and never a verdict; the
runner says so and exits non-zero even when every selected case matches.

### Sweeping an issuer's whole register

The corpus is chosen by hand: one case per prefix, one per shape, one per
boundary. It proves a rule handles the forms someone thought of. A **register
sweep** proves what the corpus cannot — that the rule refuses nothing the
issuer has actually handed out.

```sh
# The dump is never committed: half a gigabyte, renewed monthly, published
# under the issuer's terms. conformance/registers.json says where to get it.
conformance-runner --register gb-companies-house=BasicCompanyData.csv -- ./testee
```

The UK register takes about 100 seconds for its 5 695 465 companies and peaks
under 20 MB, because cases are generated, sent and discarded one at a time.
Gzipped dumps are read directly.

A sweep asserts **one** thing, and only one: no identifier in an issuer's own
register is refused. It says nothing about the canonical value, and nothing
about which of `valid` or `unsupported` the checksum should be — the register
establishes neither, and filling them in would mean computing them with the
interpreter under test, the one source a conformance expectation may never come
from.

Refusal is not only a format matter. A rule that accepts the shape and then
declares the checksum `invalid` has refused the identifier just as firmly, so a
sweep runs the whole validation and treats an invalid checksum as a refusal.
`unsupported` is never a refusal: section 1.2 is explicit that it must not turn
away a valid identifier.

So a sweep is **not** a conformance verdict and the runner never calls it one.
Conformance is settled by the corpus, which states what should happen down to
the reason code, for values that are valid and for values that are not. A sweep
catches the false refusal, which section 1.2 calls the worst defect this
project can commit and which a few hundred hand-picked cases can only sample.

A refusal is not automatically a regression. The issuer may have started
emitting a form the rule does not know, which is exactly as important to learn
and needs a person either way. When it turns out to be a real gap, the offending
value belongs in the corpus like any other defect.

## Keeping the engine checkouts in step

Before the first release the engines carry a local copy of the bundle instead of
a downloaded, attested one. `tools/sync_engines.sh` writes that copy:

```sh
go run ./cmd/businessidc compile --out dist --release --write-docs
./tools/sync_engines.sh
```

It refreshes the artifacts, rewrites `rules.lock`, and rewrites the header of
`spec/PROVENANCE.md` so the commit and version it names are the ones the lock
names. Doing it by hand drifted: three engines carried a lock at `2026.08.2`
under a header still claiming `2026.08.0`.

The script writes inside the sibling checkouts and stops there. It creates no
branch, no commit and no pull request, because publishing to another repository
stays a human decision.

An engine that then refuses the bundle with `incompatible_ruleset` is behaving
correctly: it is telling you it does not implement a capability the bundle
declares. That is a task for the engine, never a reason to remove the capability
here.

## Sources and their tier

Every rule carries at least one `source`, and every source declares a `tier`.

`primary` is a document published by the authority owning the format, or the law
establishing it. `secondary` is a third party description: a reference
implementation, an industry note, an encyclopedia.

Both are usable. They do not carry the same weight, and the point of recording
which is which is that the difference stays visible: a secondary source can be
accurate and still be someone's reading of an algorithm the authority never
published.

This is not hypothetical. Skatteverket publishes the shape of an
organisationsnummer — ten digits, the third at least two, the last a check
digit — and does not publish the algorithm computing that check digit. The
format is therefore `primary` and the Luhn algorithm `secondary`, on the same
definition.

When only a secondary source exists for a check algorithm, prefer it to no
checksum at all, and say so in `notes`: state the normative reference it cites
when there is one, and that the entry records where the algorithm was read
rather than an authority for it. A definition whose algorithm cannot be
established at all declares `absent_checksum_reason` instead of guessing.

## When a defect is found

Write the test that proves it first, watch it fail, then fix it. A fix landed
before its test has never been shown to fix anything.

**And put the offending value in the conformance corpus, not only in a Go
test.** A unit test protects this repository; a conformance case protects every
engine, in every language, including implementations this project does not
publish. The corpus is the contract between the specification and the engines,
so a defect proven only in Go leaves the others free to repeat it.

Where the value belongs depends on what it broke:

- a rule accepting or refusing the wrong identifier: a case under
  `conformance/`, carrying the exact input;
- a bundle that should have been refused: a fixture under `testdata/bundles/`
  with a `load_ruleset` case;
- a defect the corpus format genuinely cannot express - JSONL cannot hold
  invalid UTF-8, for instance - a Go test that says in its own comment why it
  stands in for a case.

The reasoning is worth keeping in mind: several defects found while porting the
historical rules were caught by a conformance case belonging to one country,
and would have shipped unnoticed in a country that happened to have no case
covering that shape.

## Commit and review

- Keep the change minimal and reviewable.
- Never disable a test, a linter or a coverage threshold to make the CI pass.
- Never introduce a mock or a `TODO` in place of a requested feature.
- Regenerate the artifacts with `make compile` and commit the generated
  documents when they change.
- `make check-generated` must stay green.
