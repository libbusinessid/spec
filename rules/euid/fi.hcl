# Finland - EUID
#
# The registration part is the Y-tunnus, seven digits closed by a check digit.
#
# A remainder of one is a hole in the numbering: no check digit exists for it,
# so such a number is never issued. The remainder table maps that case to ten,
# which no single digit can equal, and the comparison fails - which is the
# intended outcome expressed with the operations the IR has.

canonicalizer "euid" "fi" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "fi" {
  capture "register" {
    value = before_first(after_first(subject(), "FI"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.fi.empty"),
    require(starts_with(subject(), "FI"), "invalid_format", "euid.fi.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.fi.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.fi.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.fi.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.fi.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.fi.y_tunnus
    input = capture.registration
  }
}

checksum "euid" "fi" {
  rule = apply_checksum(checksum.fi.y_tunnus, after_first(subject(), "."))
}

identifier "euid" "FI" {
  canonicalizer   = canonicalizer.euid.fi
  format          = format.euid.fi
  checksum        = checksum.euid.fi
  default_profile = "compatible"

  source {
    id               = "fi-prh-ytunnus"
    url              = "https://www.prh.fi"
    authority        = "Patentti- ja rekisterihallitus (PRH)"
    title            = "Yritys- ja yhteisotunnus"
    accessed_at      = "2026-08-20"
    jurisdiction     = "FI"
    language         = "fi"
    notes            = "The register publishes the Y-tunnus as seven digits followed by a check digit."
    license_or_terms = "Finnish public sector information"
    tier             = "primary"
  }

  source {
    id               = "fi-ytunnus-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "FI"
    language         = "en"
    notes            = "Weights 7, 9, 10, 5, 8, 4, 2 over the first seven digits, remainder modulo 11. A remainder of zero gives a check digit of zero, a remainder above one gives eleven minus it, and a remainder of one is not issued."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
