# Belgium - enterprise number
#
# Ten digits issued by the Banque-Carrefour des Entreprises: eight sequential
# digits closed by a two digit check equal to 97 minus those eight modulo 97.
# The first digit is 0 or 1.
#
# It is the same number as the VAT number without its BE prefix, and it is the
# registration part of the Belgian EUID. The three share this algorithm, and
# state it once: euid.BE reuses this rule rather than restating it.

canonicalizer "be" "enterprise_number" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    remove_chars([".", "-", "/"]),
  ]
}

format "be" "enterprise_number" {
  checks = [
    require(not(is_empty(subject())), "empty", "be.enterprise_number.empty"),
    require(length_eq(subject(), 10), "invalid_length", "be.enterprise_number.length"),
    require(ascii_digits(subject()), "invalid_characters", "be.enterprise_number.characters"),
    require(
      any(
        starts_with(subject(), "0"),
        starts_with(subject(), "1"),
      ),
      "invalid_format",
      "be.enterprise_number.leading",
    ),
  ]
}

checksum "be" "enterprise_number" {
  rule = compare_slice(
    complement(mod_digits(slice(subject(), 0, 8), 97), 97),
    subject(), 8, 10,
  )
}

dispatcher "enterprise_number" {
  aliases           = ["be_enterprise_number", "ondernemingsnummer", "numero_entreprise"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "BE"
    identifier                       = identifier.enterprise_number.BE
    allow_unprefixed_without_country = true
  }
}

identifier "enterprise_number" "BE" {
  canonicalizer   = canonicalizer.be.enterprise_number
  format          = format.be.enterprise_number
  checksum        = checksum.be.enterprise_number
  default_profile = "compatible"

  source {
    id               = "be-bce-enterprise-number"
    url              = "https://economie.fgov.be/fr/themes/entreprises/banque-carrefour-des"
    authority        = "Service public federal Economie, P.M.E., Classes moyennes et Energie"
    title            = "Structure du numero d'entreprise"
    accessed_at      = "2026-08-20"
    jurisdiction     = "BE"
    language         = "fr"
    notes            = "Positions 1 to 8 are the sequential number, positions 9 and 10 the check, equal to 97 minus the first eight digits modulo 97. The first digit is 0 or 1."
    license_or_terms = "Belgian federal public sector information, reuse permitted with attribution"
    tier             = "primary"
  }
}
