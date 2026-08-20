# Business identifier atlas

What this library validates today, and what exists in the world that it does
not validate yet.

Two questions are answered here, and they are different. **Coverage** is what
the rules cover, and it is a fact about this repository. **The atlas** is what
each country issues, and it is a fact about the world. Keeping them in one page
is what makes the gap visible, which is the whole point: the gap is the roadmap.

Priority follows economic weight, because the purpose is to check a company in
a SaaS product and the customers of a SaaS product are not uniformly
distributed. Universal coverage is the destination, not the starting order.

## How to read this

**Status** — what this library does with the identifier.

| Status | Meaning |
|---|---|
| **supported** | A rule validates it and the conformance corpus proves it. |
| **partial** | A rule covers some forms of it; the entry says which are missing. |
| **planned** | Named as a next target; no rule yet. |
| **none** | No rule. Not a judgement on the identifier, only a statement of fact. |
| **no single register** | The country issues no one national identifier; see the section on federal registers. |

**Maturity** — how strongly a supported rule is established. These are ordered.

| Maturity | Meaning |
|---|---|
| **register-swept** | The rule was run against the issuer's complete public register, and refused none of it. The strongest evidence available. |
| **checked** | The issuer publishes a check algorithm; it is implemented and cross-checked against `tools/vectors.py`, which re-implements it independently. |
| **format** | The issuer publishes the format but no check algorithm. `checksum` reports `checksum_not_published` or `unsupported`, never a refusal. |

A rule at **format** is not a lesser rule. It is a rule about an identifier
whose issuer publishes less. Section 1.2 makes refusing a valid identifier the
worst defect this project can commit, so a rule never invents a check the
issuer does not publish.

**Cases** is the number of conformance cases carrying that identifier. It is a
measure of how much is proven, not of how much code exists.

---

## What is covered today

`RULES_VERSION` 2026.08.6 — 65 definitions, 494 conformance cases.

### Transnational identifiers

These belong to no single country, which is why they sit apart.

| Identifier | Issuer | Scope | Status | Maturity | Cases |
|---|---|---|---|---|---|
| **EUID** | Business Registers Interconnection System (BRIS) | 27 EU member states | supported | see table below | 126 |
| **VAT** | Each member state, checked against the EU VIES format | 31 jurisdictions | supported | mostly checked | 170 |
| **LEI** | GLEIF | worldwide | supported | checked (ISO 17442, mod 97-10) | 28 |
| **EORI** | EU customs, national customs authorities | EU + associated | supported | format | 4 |
| **D-U-N-S** | Dun & Bradstreet | worldwide | supported | format | 3 |

The D-U-N-S sits at **format** deliberately. A ninth-digit check circulated in
older documentation and was withdrawn by the issuer, so applying it would
refuse numbers Dun & Bradstreet issues.

### EUID, by member state

| Country | Maturity | Cases | | Country | Maturity | Cases |
|---|---|---|---|---|---|---|
| Austria | format | 4 | | Latvia | checked | 3 |
| Belgium | checked | 5 | | Lithuania | checked | 3 |
| Bulgaria | checked | 4 | | Luxembourg | format | 4 |
| Croatia | format | 3 | | Malta | format | 4 |
| Cyprus | format | 3 | | Netherlands | format | 3 |
| Czechia | checked | 5 | | Poland | format | 3 |
| Denmark | checked | 4 | | Portugal | checked | 3 |
| Estonia | checked | 3 | | Romania | checked | 11 |
| Finland | checked | 3 | | Slovakia | checked | 5 |
| France | checked | 24 | | Slovenia | format | 3 |
| Germany | format | 3 | | Spain | checked | 4 |
| Greece | format | 3 | | Sweden | checked | 4 |
| Hungary | format | 2 | | | | |
| Ireland | format | 4 | | | | |
| Italy | checked | 4 | | | | |

### National identifiers

An EU member state appears twice below when its national register number is
validated in its own right as well as through the EUID. The EUID rule does not
restate the national algorithm, it applies the national rule to the part after
the dot, so the two cannot drift apart.

| Country | Identifier | Issuer | Status | Maturity | Cases |
|---|---|---|---|---|---|
| France | **SIREN** | INSEE | supported | **register-swept** (Luhn) | 40 |
| France | **SIRET** | INSEE | supported | **register-swept** (Luhn over the SIREN and over the fourteen, La Poste derogation) | 22 |
| United Kingdom | **Company number** | Companies House | supported | **register-swept** | 60 |
| United States | **EIN** | Internal Revenue Service | supported | format | 3 |
| China | **USCC** | State Administration for Market Regulation | supported | checked (GB 32100-2015, mod 31 over a 31 code point alphabet) | 19 |
| Japan | **Corporate Number** | National Tax Agency | supported | checked (mod 9 over alternating weights) | 14 |
| Brazil | **CNPJ** | Receita Federal | supported | checked (two rounds of mod 11, alphanumeric form of 2026 included) | 19 |
| Belgium | **Enterprise number** | Banque-Carrefour des Entreprises | supported | checked (mod 97) | 12 |
| Bulgaria | **EIK** | Registry Agency | supported | checked (mod 11) | 4 |
| Croatia | **MBS** | Sudski registar | supported | format | 3 |
| Czechia | **IČO** | Czech Statistical Office | supported | checked (mod 11) | 5 |
| Denmark | **CVR number** | Erhvervsstyrelsen | supported | checked (mod 11) | 4 |
| Estonia | **Registrikood** (`registrikood`) | Äriregister | supported | checked (mod 11, two rounds) | 3 |
| Finland | **Y-tunnus** | PRH | supported | checked (mod 11) | 3 |
| Greece | **GEMI number** | ΓΕΜΗ | supported | format | 3 |
| Hungary | **Cégjegyzékszám** (`cegjegyzekszam`) | Court of registration | supported | format | 2 |
| Latvia | **Reģistrācijas numurs** (`registracijas_numurs`) | Uzņēmumu reģistrs | supported | checked (mod 11) | 3 |
| Lithuania | **Juridinio asmens kodas** (`juridinio_asmens_kodas`) | Registrų centras | supported | checked (mod 11) | 3 |
| Netherlands | **KvK number** | Kamer van Koophandel | supported | format | 3 |
| Poland | **KRS number** | Krajowy Rejestr Sądowy | supported | format | 3 |
| Portugal | **NIPC** | IRN | supported | checked (mod 11) | 3 |
| Romania | **CUI** | ONRC | supported | checked (mod 11) | 11 |
| Slovakia | **IČO** | Obchodný register | supported | checked (mod 11) | 5 |
| Slovenia | **Matična številka** (`maticna_stevilka`) | AJPES | supported | format | 3 |
| Sweden | **Organisationsnummer** | Bolagsverket | supported | checked (Luhn) | 4 |

Three rules have been swept against their issuer's complete register, through
the same testee protocol an engine is judged by:

| Register | Identifiers | Refused |
|---|---|---|
| Companies House, 2026-08-01 | 5 695 465 | 0 |
| SIRENE legal units, 2026-08-01 | 29 922 486 | 0 |
| SIRENE establishments, 2026-08-01 | 43 896 818 | 1 |

That one is `58209045200015`, which satisfies neither the Luhn check nor the La
Poste derogation. No source documents a derogation for the company holding it,
an ordinary limited company registered in 1958, so it reads as a single bad
record in forty four million rather than a rule. Inventing an exception for one
number would be inventing a rule with no source.

`conformance/registers.json` records where each dump comes from; the dumps
themselves are never committed. That is the standard the other rules should
reach wherever the issuer publishes a bulk download.

---

## The atlas

Ranked by nominal GDP, in bands. The ranking is indicative and moves year to
year; the bands do not.

Every identifier named here needs a primary source before a rule is written —
this table is a map, not a specification. What it does record accurately is
which country issues what, and whether we cover it.

### Rank 1-20

| # | Country | Primary business identifier | Issuer | Also carries | Status |
|---|---|---|---|---|---|
| 1 | United States | *none nationally* | — | EIN (IRS), state entity numbers, CIK (SEC), UEI (SAM.gov) | **no single register** — EIN supported |
| 2 | China | **USCC** — Unified Social Credit Code, 18 chars, check character | State Administration for Market Regulation | — | **supported** |
| 3 | Germany | **Handelsregisternummer** (HRB/HRA + court) | Local registry courts | EUID, USt-IdNr. | supported via EUID and VAT |
| 4 | Japan | **Corporate Number** (法人番号), 13 digits, check digit | National Tax Agency | — | **supported** |
| 5 | India | **CIN**, 21 chars | Registrar of Companies (MCA) | GSTIN (15), PAN (10) | planned |
| 6 | United Kingdom | **Company number**, 8 chars | Companies House | VAT | **supported** |
| 7 | France | **SIREN** (9) / **SIRET** (14) | INSEE | EUID, TVA | **supported** |
| 8 | Italy | **Codice fiscale / P.IVA** (11) | Agenzia delle Entrate, Registro Imprese | EUID | supported via EUID and VAT |
| 9 | Canada | **Business Number** (9 + program) | Canada Revenue Agency | Provincial corporation numbers | planned |
| 10 | Brazil | **CNPJ**, 14 positions, alphanumeric from 2026, two check digits | Receita Federal | — | **supported** |
| 11 | Russia | **ИНН** (10) and **ОГРН** (13) | Federal Tax Service | — | none |
| 12 | South Korea | **BRN** (사업자등록번호), 10 digits, check digit | National Tax Service | CRN (13) | planned |
| 13 | Australia | **ABN** (11, check) / **ACN** (9, check) | ATO / ASIC | — | planned |
| 14 | Spain | **NIF/CIF** | Agencia Tributaria | EUID, IVA | supported via EUID and VAT |
| 15 | Mexico | **RFC**, 12 chars for legal persons | SAT | — | planned |
| 16 | Indonesia | **NIB**, 13 digits | OSS / BKPM | NPWP | none |
| 17 | Turkey | **MERSİS**, 16 digits | Ministry of Trade | Vergi No (10) | none |
| 18 | Netherlands | **KvK-nummer**, 8 digits | Kamer van Koophandel | EUID, BTW | supported via EUID and VAT |
| 19 | Saudi Arabia | **CR number**, 10 digits | Ministry of Commerce | VAT (15) | none |
| 20 | Switzerland | **UID** — CHE-xxx.xxx.xxx, check digit | Federal Statistical Office | MWST | planned |

### Rank 21-40

| # | Country | Primary business identifier | Issuer | Also carries | Status |
|---|---|---|---|---|---|
| 21 | Poland | **KRS** (10) / **REGON** (9 or 14) | National Court Register / GUS | EUID, NIP | supported via EUID and VAT |
| 22 | Taiwan | **UBN**, 8 digits, check digit | Ministry of Economic Affairs | — | none |
| 23 | Belgium | **Ondernemingsnummer**, 10 digits, mod 97 | Crossroads Bank for Enterprises | EUID, BTW/TVA | **supported**, nationally and through EUID |
| 24 | Argentina | **CUIT**, 11 digits, check digit | AFIP | — | planned |
| 25 | Sweden | **Organisationsnummer**, 10 digits, Luhn | Bolagsverket | EUID, moms | supported via EUID and VAT |
| 26 | Ireland | **CRO number** | Companies Registration Office | EUID, VAT | supported via EUID and VAT |
| 27 | Thailand | **Juristic person ID**, 13 digits | DBD, Ministry of Commerce | — | none |
| 28 | Israel | **ח.פ.**, 9 digits, starts with 5 | Registrar of Companies | — | none |
| 29 | Austria | **Firmenbuchnummer** (FN + check letter) | Firmenbuch | EUID, UID | supported via EUID and VAT |
| 30 | Singapore | **UEN** | ACRA | GST | planned |
| 31 | Norway | **Organisasjonsnummer**, 9 digits, mod 11 | Brønnøysundregistrene | MVA | supported via VAT |
| 32 | United Arab Emirates | *none nationally* — trade licence per emirate | Each emirate's DED | TRN (15) | **no single register** |
| 33 | Vietnam | **Enterprise code**, 10 or 13 digits | Ministry of Planning and Investment | — | none |
| 34 | Malaysia | **SSM registration number**, 12 digits | Companies Commission (SSM) | — | none |
| 35 | Philippines | **SEC registration number** | Securities and Exchange Commission | TIN | none |
| 36 | Bangladesh | **RJSC registration number** | RJSC | BIN (13) | none |
| 37 | Denmark | **CVR-nummer**, 8 digits, mod 11 | Erhvervsstyrelsen | EUID, moms | supported via EUID and VAT |
| 38 | South Africa | **CIPC number**, YYYY/NNNNNN/NN | CIPC | VAT (10) | planned |
| 39 | Egypt | **Commercial Register number** | GAFI / Commercial Registry | Tax card | none |
| 40 | Hong Kong | **BRN**, 8 digits | Inland Revenue Department | CR number | planned |

### Rank 41-70

| # | Country | Primary business identifier | Issuer | Also carries | Status |
|---|---|---|---|---|---|
| 41 | Romania | **CUI/CIF** | ONRC / ANAF | EUID, TVA | supported via EUID and VAT |
| 42 | Colombia | **NIT**, check digit | DIAN | — | planned |
| 43 | Chile | **RUT**, 8 digits + mod 11 check | SII | — | planned |
| 44 | Czechia | **IČO**, 8 digits, check digit | Czech Statistical Office | EUID, DIČ | supported via EUID and VAT |
| 45 | Finland | **Y-tunnus**, 7 digits + check | PRH | EUID, ALV | supported via EUID and VAT |
| 46 | Portugal | **NIPC**, 9 digits, check digit | IRN | EUID, NIF | supported via EUID and VAT |
| 47 | Peru | **RUC**, 11 digits, check digit | SUNAT | — | none |
| 48 | New Zealand | **NZBN**, 13 digits (GS1 GLN) | NZ Companies Office | — | planned |
| 49 | Kazakhstan | **BIN** (БИН), 12 digits | Ministry of Justice | — | none |
| 50 | Greece | **ΑΦΜ**, 9 digits, check digit | AADE | EUID, ΦΠΑ | supported via EUID and VAT |
| 51 | Iraq | Commercial registration | Ministry of Trade | — | none |
| 52 | Algeria | **NIF** / registre du commerce | CNRC | — | none |
| 53 | Qatar | **CR number** | Ministry of Commerce and Industry | — | none |
| 54 | Hungary | **Cégjegyzékszám** | Court of registration | EUID, adószám | supported via EUID and VAT |
| 55 | Nigeria | **RC number** | Corporate Affairs Commission | TIN | none |
| 56 | Kuwait | **CR number** | Ministry of Commerce | — | none |
| 57 | Ukraine | **ЄДРПОУ**, 8 digits, check digit | State Registry | — | none |
| 58 | Morocco | **ICE**, 15 digits | Shared across administrations | RC, IF | planned |
| 59 | Slovakia | **IČO**, 8 digits | Business Register | EUID, DIČ | supported via EUID and VAT |
| 60 | Ethiopia | TIN / business licence | Ministry of Trade | — | none |
| 61 | Ecuador | **RUC**, 13 digits | SRI | — | none |
| 62 | Dominican Republic | **RNC**, 9 digits | DGII | — | none |
| 63 | Kenya | **PIN** (Pxxxxxxxxx) | Kenya Revenue Authority | Company number | none |
| 64 | Angola | **NIF** | AGT | — | none |
| 65 | Oman | **CR number** | Ministry of Commerce | — | none |
| 66 | Guatemala | **NIT** | SAT | — | none |
| 67 | Bulgaria | **ЕИК**, 9 or 13 digits, check digit | Registry Agency | EUID, ДДС | supported via EUID and VAT |
| 68 | Croatia | **OIB**, 11 digits, ISO 7064 | Ministry of Finance | EUID, PDV | supported via EUID and VAT |
| 69 | Luxembourg | **RCS number** (Bxxxxxx) | RCS | EUID, TVA | supported via EUID and VAT |
| 70 | Uruguay | **RUT**, 12 digits | DGI | — | none |

Also covered, outside the ranking, through EUID and VAT: Estonia, Latvia,
Lithuania, Slovenia, Malta, Cyprus, Iceland, Liechtenstein, and Northern
Ireland (`XI`) under the Windsor Framework.

---

## Countries with no single register

Three of the twenty largest economies issue no one national business
identifier. This is a property of the country, not a gap in this library, and
it changes what an honest answer looks like.

### United States

There is no federal company register. Incorporation happens at state level, and
each Secretary of State issues its own entity number in its own format — fifty
formats and change, with no federal identifier tying them together. What is
federal is narrower:

| Identifier | Issuer | What it identifies | Status |
|---|---|---|---|
| **EIN** | Internal Revenue Service | An employer for tax purposes | **supported** (format) |
| **CIK** | Securities and Exchange Commission | An EDGAR filer — public companies only | none |
| **UEI** | SAM.gov | A federal contractor | none |
| State entity number | Each Secretary of State | A company, in one state | none |

So an American company has no single number, and the EIN is the closest thing
to one — which is why it is the one carried. Supporting state entity numbers
means fifty-odd rules against fifty-odd registers, and the value of each is a
fraction of the value of a national rule elsewhere. That is a deliberate
deferral, not an oversight.

### United Arab Emirates

Each emirate's Department of Economic Development issues its own trade licence
number, and the free zones issue their own on top. The federal Tax Registration
Number (15 digits) is the only country-wide identifier, and it exists for VAT.

### Canada

The federal Business Number is genuinely national, but incorporation can be
federal or provincial, and a provincially incorporated company carries a
provincial corporation number alongside. The Business Number is the right
target; it does not make the provincial numbers disappear.

---

## What comes next

Ordered by economic weight and by how much the issuer publishes. An issuer that
publishes a check algorithm and a bulk register is worth more than a larger
economy that publishes neither, because the rule can be proven.

1. **India — CIN.** 21 structured characters carrying state and year, so the
   rule catches far more than a length check would.
2. **Australia — ABN and ACN.** Both carry a published check.
3. **Switzerland — UID.** One identifier across commercial register, VAT,
   customs and social insurance; a check digit; a public register.
4. **South Korea — BRN.** 10 digits with a published check.
5. **Canada — Business Number**, then **Mexico RFC**, **Argentina CUIT**,
   **Chile RUT**, **Colombia NIT** — the Latin American set shares a family
   resemblance and a check digit each.

Deliberately deferred, with the reason:

- **US state entity numbers** — fifty rules, fifty registers, a fraction of the
  value each.
- **Russia (ИНН/ОГРН)** — the register is public and the check is published;
  the deferral is about how much these are used in the customer base a SaaS
  product actually serves.
- **Countries with no public register** — where nothing is published, a rule
  would be a guess, and a guess that refuses a valid identifier is the worst
  outcome this project recognises.

Nothing here is closed. An identifier moves up the list when someone needs it,
and the entry above is what has to be true before a rule is written: a named
issuer, a published format, and preferably a register to sweep it against.
