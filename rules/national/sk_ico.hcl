# Slovakia - identification number (ICO)
#
# The registration part of the Slovakia EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "sk" "ico" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "sk" "ico" {
  checks = [
    require(not(is_empty(subject())), "empty", "sk.ico.empty"),
    require(length_eq(subject(), 8), "invalid_length", "sk.ico.length"),
    require(ascii_digits(subject()), "invalid_characters", "sk.ico.characters"),
  ]
}

checksum "sk" "ico" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(subject(), 0, 7), [8, 7, 6, 5, 4, 3, 2], "left", "digit_value"), 11), [1, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1]),
    subject(), 7,
  )
}

identifier "ico" "SK" {
  canonicalizer   = canonicalizer.sk.ico
  format          = format.sk.ico
  checksum        = checksum.sk.ico
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
