# Conformance cases

The conformance suite is the shared contract of the four engines. It is written
and reviewed in JSONL and compiled into `entid-conformance.binpb`.

- JSONL is the canonical human source;
- BINPB is the typed artifact consumed by the engines;
- the BINPB is never edited by hand;
- both carry the same `rules_version`.

Read [`../DATA_POLICY.md`](../DATA_POLICY.md) before adding a value.

## 1. No circular oracle

The compiler **never** computes an expected result from the rule under test.
Every canonical value, status, reason code and message key is written or
approved explicitly by a reviewer, from an independent source: the published
algorithm, an independent implementation such as `tools/vectors.py`, or an
authority document.

The reference interpreter checks the written expectations; it never produces
them.

## 2. Schema

Each line is a self contained JSON object. Unknown fields are refused.

```json
{"id":"vat-be-valid-001","description":"Synthetic Belgian VAT number with a correct modulo 97 check","kind":"vat","input":"BE 0123.456.749","profile":"compatible","operation":"validate","expected":{"canonicalValue":"BE0123456749","countryCode":"BE","format":{"status":"valid","reasonCode":"ok"},"checksum":{"status":"valid","reasonCode":"ok"}},"tags":["mod97","synthetic","valid"],"sourceIds":["be-fps-finance-vat"],"dataClassification":"synthetic","redistributionBasis":"..."}
```

| Field | Required | Meaning |
|---|---|---|
| `id` | always | unique identifier of the case |
| `description` | recommended | one sentence, at most 4096 bytes |
| `kind` | business operations | the requested kind, possibly an alias |
| `countryCode` | optional | the country context supplied by the caller |
| `input` | business operations | the raw value |
| `profile` | business operations | `compatible` or `strict_current` |
| `operation` | always | see below |
| `expected` | business operations | the reviewed expectation |
| `tags` | always | sorted, unique, non empty |
| `sourceIds` | official examples | sorted, unique source ids of the rules |
| `fixture` | `load_ruleset` | path under `testdata/` |
| `expectedEngineError` | `load_ruleset` | `invalid_ruleset` or `incompatible_ruleset` |
| `generated` | optional | reserved for a future generator; always `false` today |
| `dataClassification` | always | `official_public_example` or `synthetic` |
| `redistributionBasis` | always | non empty justification |

Supported operations: `canonicalize`, `validate_format`, `validate_checksum`,
`validate` and `load_ruleset`.

A `load_ruleset` case forbids `kind`, `countryCode`, `input`, `profile` and
`expected`, and requires `fixture` and `expectedEngineError`.

## 3. Expectations

`canonicalize` expects `canonicalValue`, an optional `countryCode`, a `status`
and a `reasonCode`, plus an optional `messageKey`.

The three validation operations expect `canonicalValue`, an optional
`countryCode`, and the two steps `format` and `checksum`, each holding a
`status`, a `reasonCode` and an optional `messageKey`. After a valid format,
`validate_format` requires `checksum` to be `not_run` / `not_requested`.

`expected.kind` is only written when the requested kind is an alias, since the
result reports the canonical kind.

The compiler copies mechanically only `kind`, `input`, `profile`, the country
declared inside `expected`, and the `rulesVersion` / `formatVersion` of the
bundle. Everything else is authored.

Status and reason code pairings are constrained: `valid` always uses `ok`;
`not_run` uses `not_requested`, `not_run_format_invalid` or
`not_run_format_unsupported`; `invalid` uses only a reason that proves an
invalidity; every other business reason is `unsupported`.

## 4. Minimum matrix per variant

Each variant must hold:

- at least two independent valid examples when available;
- a too short and a too long value;
- invalid characters at each position class;
- a valid checksum;
- a mutation of each check digit;
- the separators and the letter case accepted by canonicalization;
- the empty value;
- a missing country and a contradictory country when relevant;
- both profiles when they differ;
- the `unsupported` branch when a checksum is not published;
- the exact bounds of every range;
- each branch of a multi variant algorithm.

Any reported false negative or false positive must produce a regression case
**before** the fix.

## 5. Limits

```text
conformance bundle    <= 64 MiB
cases                 <= 1 000 000
description           <= 4 096 bytes
tags                  sorted and unique
ids                   unique
```

Engines may stream or index the cases, but must run all of them. A case is never
excluded to make a CI pass.

## 6. Local run

```bash
make verify                       # the whole suite against the reference interpreter
go test ./internal/conformance/   # the same run as a Go test
```
