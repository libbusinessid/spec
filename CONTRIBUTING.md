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
