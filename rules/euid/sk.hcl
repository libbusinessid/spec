# # Slovakia - EUID\n#\n# The registration part is the eight digit ICO of the business register. It\n# shares its modulo 11 check with the Czech ICO, both descending from the same\n# Czechoslovak numbering.

canonicalizer "euid" "sk" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "sk" {
  capture "register" {
    value = before_first(after_first(subject(), "SK"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.sk.empty"),
    require(starts_with(subject(), "SK"), "invalid_format", "euid.sk.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.sk.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.sk.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.sk.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.sk.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.sk.ico
    input = capture.registration
  }
}

checksum "euid" "sk" {
  rule = apply_checksum(checksum.sk.ico, after_first(subject(), "."))
}

identifier "euid" "SK" {
  canonicalizer   = canonicalizer.euid.sk
  format          = format.euid.sk
  checksum        = checksum.euid.sk
  default_profile = "compatible"

  source {
    id               = "sk-orsr-ico"
    url              = "https://www.orsr.sk"
    authority        = "Ministerstvo spravodlivosti Slovenskej republiky"
    title            = "Obchodny register Slovenskej republiky"
    accessed_at      = "2026-08-20"
    jurisdiction     = "SK"
    language         = "sk"
    notes            = "The register publishes the eight digit ICO. It does not publish the algorithm closing it."
    license_or_terms = "Slovak public sector information"
    tier             = "primary"
  }

  source {
    id               = "sk-ico-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "SK"
    language         = "en"
    notes            = "Same weighted modulo 11 as the Czech ICO: weights 8 to 2 over the first seven digits."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
