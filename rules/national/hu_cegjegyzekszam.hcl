# Hungary - company registration number
#
# The registration part of the Hungary EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "hu" "cegjegyzekszam" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars(["-", "/"]),
  ]
}

format "hu" "cegjegyzekszam" {
  checks = [
    require(not(is_empty(subject())), "empty", "hu.cegjegyzekszam.empty"),
    require(length_eq(subject(), 10), "invalid_length", "hu.cegjegyzekszam.length"),
    require(ascii_digits(subject()), "invalid_characters", "hu.cegjegyzekszam.characters"),
  ]
}

dispatcher "cegjegyzekszam" {
  aliases           = ["hu_cegjegyzekszam"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "HU"
    identifier                       = identifier.cegjegyzekszam.HU
    allow_unprefixed_without_country = true
  }
}

identifier "cegjegyzekszam" "HU" {
  canonicalizer   = canonicalizer.hu.cegjegyzekszam
  format          = format.hu.cegjegyzekszam
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "hu-ceginformacio"
    url              = "https://www.e-cegjegyzek.hu"
    authority        = "Igazsagugyi Miniszterium"
    title            = "Cegjegyzek"
    accessed_at      = "2026-08-20"
    jurisdiction     = "HU"
    language         = "hu"
    notes            = "The register publishes the ten digit cegjegyzekszam, printed with dashes."
    license_or_terms = "Hungarian public sector information"
    tier             = "primary"
  }
}
