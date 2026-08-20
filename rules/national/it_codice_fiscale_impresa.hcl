# Italy - codice fiscale of a company
#
# The registration part of the Italy EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "it" "codice_fiscale_impresa" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "it" "codice_fiscale_impresa" {
  checks = [
    require(not(is_empty(subject())), "empty", "it.codice_fiscale_impresa.empty"),
    require(length_eq(subject(), 11), "invalid_length", "it.codice_fiscale_impresa.length"),
    require(ascii_digits(subject()), "invalid_characters", "it.codice_fiscale_impresa.characters"),
  ]
}

checksum "it" "codice_fiscale_impresa" {
  rule = luhn(subject())
}

dispatcher "codice_fiscale_impresa" {
  aliases           = ["it_codice_fiscale_impresa"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "IT"
    identifier                       = identifier.codice_fiscale_impresa.IT
    allow_unprefixed_without_country = true
  }
}

identifier "codice_fiscale_impresa" "IT" {
  canonicalizer   = canonicalizer.it.codice_fiscale_impresa
  format          = format.it.codice_fiscale_impresa
  checksum        = checksum.it.codice_fiscale_impresa
  default_profile = "compatible"

  source {
    id               = "it-registro-imprese-number"
    url              = "https://it.wikipedia.org/wiki/Partita_IVA"
    authority        = "Wikipedia, citing the Italian ministerial decree of 23 December 1976"
    title            = "Partita IVA - struttura e carattere di controllo"
    accessed_at      = "2026-08-20"
    jurisdiction     = "IT"
    language         = "it"
    notes            = "Eleven digits, the last a Luhn check over the eleven digits. The normative reference is the decreto ministeriale of 23 December 1976, which the issuing administration does not publish online; this entry records where the algorithm was read, not an authority for it."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
