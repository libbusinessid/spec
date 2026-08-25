# LibEntID - `spec`

`spec` is the source of truth of LibEntID. It holds the authoring language,
the intermediate representation, the official rules, the shared conformance
suite and the compiler that turns all of it into the artifacts the engines
consume.

**Which identifiers does it support?**
[`docs/identifier-atlas.md`](docs/identifier-atlas.md) answers that, and answers
it next to what every major economy issues — so the gap between the two is
visible rather than implied.

## What this project does, and what it does not

LibEntID answers two precise questions about a business identifier:

- **format valid**: the shape is compatible with a documented variant;
- **checksum valid**: the documented internal check is satisfied.

It never claims that a company exists, is active, or belongs to anyone. That
requires a register, and V1 ships no register provider and performs no network
call.

A value is declared `invalid` only when a documented and applicable rule proves
it. An unknown rule, an unpublished algorithm, an ambiguous variant or an
incomplete coverage always produce `unsupported`. Refusing a valid identifier is
considered the most serious defect of the project.

## Contents

```text
proto/          the IR and conformance Protobuf schemas
rules/          the official rules, written in the LibEntID HCL language
conformance/    the reviewed JSONL corpus shared by every engine
cmd/entidc the compiler, linker, linter, inspector and publisher
internal/       the compiler stages and the internal reference interpreter
docs/           the language guide and the generated normative references
testdata/       the decoder fixtures and the minimal reference bundle
tools/          the version locked tooling and the independent oracles
```

The repository contains **no** engine. The Go, Swift, Kotlin and TypeScript
engines live in their own repositories and consume the published artifacts. The
Go interpreter of `internal/reference` exists only to verify the specification,
the bundles and the reviewed expectations; it is never published as a library.

## Pipeline

```text
rules/*.hcl + conformance/*.jsonl
                 |
                 v
       parser / linker / typechecker
                 |
                 v
 typed IR: dispatchers + definitions + programs
                 |
        +--------+---------+
        v                  v
entid-rules.binpb  entid-conformance.binpb
        |                  |
        +--------+---------+
                 v
       engines and test suites
```

HCL is an authoring language only. No production engine parses HCL: every
symbolic reference is resolved by `entidc`.

## Getting started

```bash
make tools        # build the version locked generation tools
make generate     # regenerate the Protobuf code and the decoder fixtures
make verify       # run the whole conformance suite
make compile      # rebuild every artifact under dist/
make ci           # everything the pull request pipeline runs
```

## `entidc`

```text
entidc fmt [--check] [paths...]
entidc lint [--rules rules] [--cases conformance]
entidc compile --rules rules --cases conformance --out dist [--write-docs] [--release] [--optimize=false]
entidc verify --rules rules --cases conformance
entidc inspect [--json] dist/entid-rules-<version>.binpb
entidc diff [--json] old.binpb new.binpb
entidc check-generated
entidc version
```

Every command accepts `--json` to emit machine readable output on stdout; human
readable diagnostics go to stderr.

| Exit code | Meaning |
|---:|---|
| 0 | success |
| 1 | usage error |
| 2 | the inputs are rejected, diagnostics were reported |
| 3 | internal error |

`compile` writes atomically and never leaves a partial artifact behind. In
release mode `SOURCE_DATE_EPOCH` is mandatory and the manifest is marked
`reproducible: true`; a local build without it is marked `reproducible: false`.
`--optimize=false` disables the structural deduplication of identical
sub-graphs, which is useful when debugging an IR graph; an equivalence test
proves that both modes execute identically.

## Proving an engine conformant

An engine is not expected to interpret the bundle. It compiles it to native code
with a generator of its own, written in whatever language suits it — this
repository hosts no generator and knows no target language.

Conformance is therefore checked through a protocol rather than a shared test
suite. `conformance-runner` is the only program that reads expected results. The
engine supplies a **testee**: a small executable reading requests on stdin,
calling its public API, and writing responses on stdout, each message preceded by
its length as a 32-bit little-endian integer.

```sh
make conformance   # runs the whole corpus against the reference testee
```

To qualify your own engine:

```sh
conformance-runner --corpus entid-conformance-<version>.binpb -- ./your-testee
```

Because the testee never sees an expected result, it cannot declare itself
conformant by comparing too weakly — the failure mode of a suite reimplemented in
each engine. `cmd/conformance-testee` is a worked example of the protocol, built
on the reference interpreter.

This is also how a third-party engine, in a language this project does not
publish, qualifies without any change here. Section 8.7 of the specification is
normative; `proto/entid/testee/v1/testee.proto` carries the schema.

## Published artifacts

```text
entid-rules-YYYY.MM.PATCH.binpb
entid-conformance-YYYY.MM.PATCH.binpb
entid-conformance-YYYY.MM.PATCH.jsonl.gz
entid-manifest-YYYY.MM.PATCH.json
reference-bundle.binpb            (minimal bundle for a new engine)
reference-conformance.binpb       (its conformance suite)
rules.proto
conformance.proto
ir.md
features.md
SHA256SUMS
SBOM.spdx.json
provenance.intoto.jsonl   (produced by the release attestation)
```

The bundles carry no timestamp: `generatedAt` exists only in the manifest and is
derived from `SOURCE_DATE_EPOCH`.

## Versions

Four things move at different rates and are versioned independently.

| Axis | Form | Changes when | Effect on the engines |
|---|---|---|---|
| `formatVersion` | integer | the structure of the IR changes | breaking: the engines must be published first |
| `requiredFeatures` | frozen capability IDs | a rule uses a new primitive | an engine refuses a bundle declaring an ID it does not implement |
| `rulesVersion` | `YYYY.MM.PATCH` | at every batch of rules | none, as long as the two above do not move |
| `compilerVersion` | SemVer | the tooling evolves | none |

A release publishes artifacts, not the specification. One release is one
`rulesVersion` and one immutable Git tag `v<rulesVersion>`; the patch number
carries every rule change, so the specification can stay untouched across dozens
of releases. Two artifacts must never share a `rulesVersion`: `rules.lock`
identifies a bundle by that version and its SHA-256, and `entidc diff`, the
revocation list and the downstream verification all rely on that being unique.

`RULES_STABILITY` declares the maturity of the **contract**, never the extent of
the rule coverage — rules keep evolving forever, so coverage can never be a
release criterion. A release stays `alpha` while only the reference interpreter
has validated the contract, and becomes `stable` once an independent engine
passes the whole conformance suite on a published bundle. Anything other than
`stable` is published as a GitHub pre-release, which keeps it out of the
`releases/latest` endpoint so that no consumer picks it up by accident.

## Documentation

| Document | Contents |
|---|---|
| [`conformance/registers.json`](conformance/registers.json) | where each issuer publishes its complete register, for `conformance-runner --register` |
| [`docs/identifier-atlas.md`](docs/identifier-atlas.md) | which identifiers are supported, at what maturity, and what the rest of the world issues |
| [`docs/language.md`](docs/language.md) | the authoring language, its declarations, expressions and restrictions |
| [`docs/ir.md`](docs/ir.md) | generated: the exhaustive normative semantics of every opcode, limit and error |
| [`docs/features.md`](docs/features.md) | generated: the frozen content of every capability ID |
| [`docs/conformance.md`](docs/conformance.md) | the conformance case schema and the review rules |
| [`docs/generated/coverage.md`](docs/generated/coverage.md) | generated: coverage, dispatch tables, algorithms, sources and statistics |
| [`docs/normative-decisions.md`](docs/normative-decisions.md) | the resolved ambiguities of the specification |
| [`docs/spec/`](docs/spec/) | the normative specification of this repository and the engine contracts |

## Governance

`CONTRIBUTING.md` describes the pull request requirements, `DATA_POLICY.md` is
normative for every conformance value, and `SECURITY.md` describes the threat
model, the release integrity chain and the revocation procedure.

## Licence

Apache-2.0. See `LICENSE`.
