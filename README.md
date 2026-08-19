# LibBusinessID - `spec`

`spec` is the source of truth of LibBusinessID. It holds the authoring language,
the intermediate representation, the official rules, the shared conformance
suite and the compiler that turns all of it into the artifacts the engines
consume.

## What this project does, and what it does not

LibBusinessID answers two precise questions about a business identifier:

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
rules/          the official rules, written in the LibBusinessID HCL language
conformance/    the reviewed JSONL corpus shared by every engine
cmd/businessidc the compiler, linker, linter, inspector and publisher
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
businessid-rules.binpb  businessid-conformance.binpb
        |                  |
        +--------+---------+
                 v
       engines and test suites
```

HCL is an authoring language only. No production engine parses HCL: every
symbolic reference is resolved by `businessidc`.

## Getting started

```bash
make tools        # build the version locked generation tools
make generate     # regenerate the Protobuf code and the decoder fixtures
make verify       # run the whole conformance suite
make compile      # rebuild every artifact under dist/
make ci           # everything the pull request pipeline runs
```

## `businessidc`

```text
businessidc fmt [--check] [paths...]
businessidc lint [--rules rules] [--cases conformance]
businessidc compile --rules rules --cases conformance --out dist [--write-docs] [--release] [--optimize=false]
businessidc verify --rules rules --cases conformance
businessidc inspect [--json] dist/businessid-rules-<version>.binpb
businessidc diff [--json] old.binpb new.binpb
businessidc check-generated
businessidc version
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

## Published artifacts

```text
businessid-rules-YYYY.MM.PATCH.binpb
businessid-conformance-YYYY.MM.PATCH.binpb
businessid-conformance-YYYY.MM.PATCH.jsonl.gz
businessid-manifest-YYYY.MM.PATCH.json
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
identifies a bundle by that version and its SHA-256, and `businessidc diff`, the
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
