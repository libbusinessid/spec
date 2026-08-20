# Finland - business identity code (Y-tunnus)
#
# The registration part of the Finland EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "fi" "y_tunnus" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "/"]),
  ]
}

format "fi" "y_tunnus" {
  checks = [
    require(not(is_empty(subject())), "empty", "fi.y_tunnus.empty"),
    require(length_eq(subject(), 8), "invalid_length", "fi.y_tunnus.length"),
    require(ascii_digits(subject()), "invalid_characters", "fi.y_tunnus.characters"),
  ]
}

checksum "fi" "y_tunnus" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(subject(), 0, 7), [7, 9, 10, 5, 8, 4, 2], "left", "digit_value"), 11), [0, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1]),
    subject(), 7,
  )
}

dispatcher "y_tunnus" {
  aliases           = ["fi_y_tunnus", "business_id"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "FI"
    identifier                       = identifier.y_tunnus.FI
    allow_unprefixed_without_country = true
  }
}

identifier "y_tunnus" "FI" {
  canonicalizer   = canonicalizer.fi.y_tunnus
  format          = format.fi.y_tunnus
  checksum        = checksum.fi.y_tunnus
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
