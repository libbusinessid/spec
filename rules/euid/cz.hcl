# # Czechia - EUID\n#\n# The registration part is the eight digit ICO of the public register of legal\n# entities. Its last digit closes a weighted modulo 11 over the first seven.

canonicalizer "euid" "cz" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "cz" {
  capture "register" {
    value = before_first(after_first(subject(), "CZ"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.cz.empty"),
    require(starts_with(subject(), "CZ"), "invalid_format", "euid.cz.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.cz.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.cz.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.cz.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.cz.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.cz.ico
    input = capture.registration
  }
}

checksum "euid" "cz" {
  rule = apply_checksum(checksum.cz.ico, after_first(subject(), "."))
}

identifier "euid" "CZ" {
  canonicalizer   = canonicalizer.euid.cz
  format          = format.euid.cz
  checksum        = checksum.euid.cz
  default_profile = "compatible"

  source {
    id               = "cz-ares-ico"
    url              = "https://ares.gov.cz"
    authority        = "Ministerstvo financi Ceske republiky - ARES"
    title            = "Administrativni registr ekonomickych subjektu"
    accessed_at      = "2026-08-20"
    jurisdiction     = "CZ"
    language         = "cs"
    notes            = "The register publishes the eight digit ICO. It does not publish the algorithm closing it."
    license_or_terms = "Czech public sector information"
    tier             = "primary"
  }

  source {
    id               = "cz-ico-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "CZ"
    language         = "en"
    notes            = "Weights 8 to 2 over the first seven digits, remainder modulo 11, the check digit being 1 for a remainder of 0 or 10, 0 for a remainder of 1, and 11 minus the remainder otherwise."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
