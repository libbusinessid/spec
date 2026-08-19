# Data policy

This document is **normative** for every conformance case, fixture and example
of the repository. A pull request that violates it is refused, whatever its
technical quality.

## 1. Principle

A business identifier can also identify a natural person: a sole trader, a
self-employed professional or an individual entrepreneur is registered with the
same kind of number as a company. Every identifier value is therefore treated as
**potentially personal data by default**.

The project never claims that an entity exists. It verifies a documented format
and a documented internal checksum, nothing else.

## 2. Forbidden data

The corpus, the fixtures, the tests, the issues and the pull requests must never
contain:

- data extracted from a production system, a log, a database dump or a backup;
- an identifier transmitted by a user, a customer or a partner;
- personal data collected from a register, an incident report or telemetry;
- a bulk extract of a public register, even when the register is freely
  accessible;
- a credential, a token or any secret.

The expression "production case" is forbidden in the corpus and rejected
automatically by `businessidc lint`.

The project **never queries a register to check** whether a synthetic value
happens to be assigned: that verification would itself create an unjustified
processing of personal data.

## 3. Accepted data

Only four categories are accepted:

1. **`official_public_example`** - an example intentionally published by the
   authority owning the format, with a verified right of redistribution. The
   case keeps its source id, its URL, its access date and the applicable terms.
2. **`synthetic`** - a value entirely produced by the documented generator of
   section 4, derived from no real data set.
3. **`public_business_identifier`** - a real identifier of a legal person,
   already published by the entity itself or by a public register, used to
   verify that a rule matches what a register actually issues.
4. Synthetic mutations of the categories above, used as negative cases.

Every case declares `dataClassification` and a non empty `redistributionBasis`.

### Why the third category exists

A synthetic value proves an algorithm. A Luhn check does not know where its
digits came from, so `123456782` exercises it exactly as a real SIREN would.

What a synthetic value cannot prove is that the **rule describes the format a
register actually issues**. When twenty six national registers are ported at
once, that is the failing mode that matters: a rule can compute a perfect
checksum over a shape no register ever emits, and every synthetic case will
pass. A handful of real identifiers is the only thing that catches it.

The category is therefore for **verification of shape**, not for coverage. It
carries three conditions:

- the identifier must designate a **legal person**, never a natural person. Sole
  traders and self-employed identifiers are excluded, because such an identifier
  designates an individual;
- the value must already be published by the entity itself - legal notices,
  terms of service, filings - or by a public register that permits
  redistribution;
- `redistributionBasis` states where it was published and under which terms.

The project holds no name, address, officer, activity or any other attribute.
A case carries the identifier, the expected outcome, and nothing that would make
the corpus a directory. That distinction is what keeps this a test of an
algorithm rather than a republication of a register.

## 4. The synthetic generator

A positive synthetic value is produced as follows, and only as follows:

1. take an **ascending or constant digit run** as the body of the identifier,
   for example `01234567`, `00000000`, `12345678`, `09999999`;
2. complete it with the **published check algorithm** of the format, computed by
   the independent Python implementation of `tools/vectors.py`, which never
   consults the Go compiler or the Go reference interpreter;
3. add the canonical prefix declared by the rule when the format requires one.

For an identifier holding an issuer prefix, the generator uses a prefix that is
**not assigned by the issuer**. The LEI cases use the LOU prefix `0000`, which
GLEIF does not allocate; no LEI can therefore exist with that prefix.

This construction is the non attribution argument required by section 8.3 of
`docs/spec/spec.md`: the values are not drawn from any register, extract, user
submission or telemetry, and their shape - long ascending or constant digit runs
and unassigned issuer prefixes - is deliberately implausible as an assigned
identifier.

Negative cases are obtained by mutating a positive case: one digit, one
character class, one length or one check digit at a time. A negative case is by
construction not a valid identifier.

## 5. Residual risk and takedown

The project cannot prove, without querying a register, that a synthetic value is
unassigned. That verification is deliberately not performed.

If anyone reports that a published value identifies a real natural person:

1. open a **private** security advisory on the repository, never a public issue,
   and never repeat the value in a public channel;
2. a maintainer removes the case in the next commit, without waiting for a
   release;
3. the removal derogates from the usual retention of golden files: the value
   disappears from the corpus, from the generated artifacts and from the
   documentation of the current release;
4. a new rule release is published with the replacement case;
5. the incident is recorded in `SECURITY.md` without repeating the value.

An urgent removal never requires the usual review delay.

## 6. Review of a data pull request

A pull request touching `conformance/**` or `testdata/**` requires:

- a confidentiality and licence review by a maintainer;
- the automatic checks of `businessidc lint`: classification, non empty
  redistribution basis, forbidden phrases, source references;
- the secret scanning of the CI;
- for an `official_public_example`, the URL, the access date and the applicable
  terms of the authority.

## 7. Licences

Every example published by an authority keeps the terms of that authority in the
`license_or_terms` field of its source. When those terms forbid redistribution,
the example is **not** added: a synthetic value is used instead.

The repository itself is published under Apache-2.0. The conformance corpus is
published under the same terms, without prejudice to the terms of a
redistributed official example.
