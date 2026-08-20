# Italy - EUID
#
# The registration part is the eleven digit number of the registro delle
# imprese, which is also the partita IVA. Its last digit is a Luhn check over
# the eleven digits, defined by the ministerial decree of 23 December 1976.
#
# The decree itself is not published online by the issuing administration, so
# the algorithm is carried on a secondary source that cites it. The format is
# stated by the register operator.

canonicalizer "euid" "it" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "it" {
  capture "register" {
    value = before_first(after_first(subject(), "IT"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.it.empty"),
    require(starts_with(subject(), "IT"), "invalid_format", "euid.it.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.it.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.it.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.it.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.it.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.it.codice_fiscale_impresa
    input = capture.registration
  }
}

checksum "euid" "it" {
  rule = apply_checksum(checksum.it.codice_fiscale_impresa, after_first(subject(), "."))
}

identifier "euid" "IT" {
  canonicalizer   = canonicalizer.euid.it
  format          = format.euid.it
  checksum        = checksum.euid.it
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
