# Latvia - registration number
#
# The registration part of the Latvia EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "lv" "registracijas_numurs" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "lv" "registracijas_numurs" {
  checks = [
    require(not(is_empty(subject())), "empty", "lv.registracijas_numurs.empty"),
    require(length_eq(subject(), 11), "invalid_length", "lv.registracijas_numurs.length"),
    require(ascii_digits(subject()), "invalid_characters", "lv.registracijas_numurs.characters"),
    require(char_at_in(subject(), 0, "456789"), "invalid_format", "lv.registracijas_numurs.legal_entity"),
  ]
}

checksum "lv" "registracijas_numurs" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(subject(), 0, 10), [9, 1, 4, 8, 3, 10, 2, 5, 7, 6], "left", "digit_value"), 11), [3, 2, 1, 0, 10, 9, 8, 7, 6, 5, 4]),
    subject(), 10,
  )
}

dispatcher "registracijas_numurs" {
  aliases           = ["lv_registracijas_numurs"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "LV"
    identifier                       = identifier.registracijas_numurs.LV
    allow_unprefixed_without_country = true
  }
}

identifier "registracijas_numurs" "LV" {
  canonicalizer   = canonicalizer.lv.registracijas_numurs
  format          = format.lv.registracijas_numurs
  checksum        = checksum.lv.registracijas_numurs
  default_profile = "compatible"

  source {
    id               = "lv-ur-registration-number"
    url              = "https://www.ur.gov.lv"
    authority        = "Latvijas Republikas Uznemumu registrs"
    title            = "Registration number of a legal entity"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LV"
    language         = "lv"
    notes            = "The register publishes an eleven digit registration number whose first digit distinguishes a legal entity from a personal code."
    license_or_terms = "Latvian public sector information"
    tier             = "primary"
  }
  source {
    id               = "lv-number-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LV"
    language         = "en"
    notes            = "Weights 9, 1, 4, 8, 3, 10, 2, 5, 7, 6 over the first ten digits, the check digit being three minus the sum modulo eleven, and a result of ten never being issued."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
