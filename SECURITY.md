# Security policy

## Reporting a vulnerability

Report privately through the GitHub security advisories of
`entid-org/spec`. Never open a public issue for:

- a supply chain compromise;
- a forged bundle accepted by an engine;
- a denial of service on the compiler or an engine;
- a conformance value identifying a real natural person (see `DATA_POLICY.md`).

A first answer is sent within five working days.

## Threat model

The repository considers the following threats:

| Threat | Mitigation |
|---|---|
| Forged or corrupted bundle | Full defensive validation before any execution, in the order of `docs/ir.md` section 9. |
| Size or depth bomb | Normative limits on the binary size, node counts, call depth and evaluation budget. |
| Allocation amplification | A produced string and every canonicalization step are billed against the evaluation budget, one unit per started slice of 64 code points, so the memory one operation can materialize is bounded by the budget. |
| Unknown node silently ignored by Protobuf | Every unknown field is refused at any depth; the loader never relies on `required_feature_ids` alone. |
| Integer overflow | Every addition, multiplication, negation and conversion is checked; unprovable arithmetic makes the bundle `invalid_ruleset`. |
| Unicode divergence between runtimes | The `whitespace_v1` table and the ASCII classes are frozen in `docs/ir.md`; a runtime never delegates them. |
| ReDoS | The IR holds no generic regular expression. |
| Rule cycle | The call graph is proven acyclic at compile time and at load time. |
| Checksum executed on a non conforming input | The format always acts as a guard; a failed format makes the checksum `not_run`. |
| Wrong or obsolete business source | Every rejecting rule carries a source; `entidc diff` classifies restrictions as high risk changes. |
| Supply chain attack | Locked tool versions, SBOM, dependency audit, OIDC attestation of the release, downstream verification of the SHA-256 and of the attestation. |

## Untrusted inputs

HCL, JSONL, Protobuf and any custom bundle are treated as **untrusted**. No
input may cause a panic, an infinite loop or an unbounded allocation. The fuzz
targets of the repository cover the HCL parser, the JSONL reader, the Protobuf
loader, reference graphs, unusual Unicode and the arithmetic bounds.

## Release integrity

A release is published by `.github/workflows/release.yml` with a GitHub OIDC
workflow identity. The build job holds only `contents: read`; a distinct job,
protected by a GitHub environment, receives `id-token: write` and
`attestations: write`. Release tags are immutable and protected. No long lived
secret signs an artifact.

Each artifact is published with:

- its SHA-256 in `SHA256SUMS`;
- a Sigstore / GitHub artifact attestation bound to the repository, the commit,
  the protected tag and the expected release workflow;
- an SPDX SBOM.

Engine repositories verify **both** the SHA-256 **and** the attestation
(owner, repository, workflow, commit and tag) before accepting an update.

## Revocation

When a published release must be revoked:

1. a security advisory is published;
2. the release is withdrawn when the defect allows a false negative or a false
   positive on a rejecting rule;
3. the digest of the revoked artifacts is added to the signed revocation list
   published with the next release;
4. a rollback pull request is opened automatically in the four engine
   repositories;
5. the automation can never approve or merge its own pull request.

## Out of scope in V1

Engines never download rules at runtime. If dynamic distribution is ever
introduced, a runtime trust policy and a dedicated application level signature
must be specified before it is enabled.
