# Slovenia - registration number
#
# The registration part of the Slovenia EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "si" "maticna_stevilka" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "si" "maticna_stevilka" {
  checks = [
    require(not(is_empty(subject())), "empty", "si.maticna_stevilka.empty"),
    require(length_eq(subject(), 7), "invalid_length", "si.maticna_stevilka.length"),
    require(ascii_digits(subject()), "invalid_characters", "si.maticna_stevilka.characters"),
  ]
}

dispatcher "maticna_stevilka" {
  aliases           = ["si_maticna_stevilka"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "SI"
    identifier                       = identifier.maticna_stevilka.SI
    allow_unprefixed_without_country = true
  }
}

identifier "maticna_stevilka" "SI" {
  canonicalizer   = canonicalizer.si.maticna_stevilka
  format          = format.si.maticna_stevilka
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "si-ajpes-maticna"
    url              = "https://www.ajpes.si"
    authority        = "Agencija Republike Slovenije za javnopravne evidence in storitve (AJPES)"
    title            = "Poslovni register Slovenije"
    accessed_at      = "2026-08-20"
    jurisdiction     = "SI"
    language         = "sl"
    notes            = "The register publishes the seven digit maticna stevilka."
    license_or_terms = "Slovenian public sector information"
    tier             = "primary"
  }
}
