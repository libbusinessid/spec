# Latvia - EUID
#
# The registration part is the eleven digit registration number of a legal
# entity, whose first digit is four or above.
#
# A number starting below four is a personal code, which identifies a natural
# person rather than a company. It is not accepted here: an EUID designates an
# entity of a business register, and validating personal codes would put the
# corpus in the business of checking people.
#
# The check digit is three minus the weighted sum, modulo eleven. A result of
# ten is not issued, and the remainder table maps it to ten so that no single
# digit can match it.

canonicalizer "euid" "lv" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "lv" {
  capture "register" {
    value = before_first(after_first(subject(), "LV"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.lv.empty"),
    require(starts_with(subject(), "LV"), "invalid_format", "euid.lv.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.lv.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.lv.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.lv.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.lv.register_characters"),
    require(length_eq(capture.registration, 11), "invalid_length", "euid.lv.registration_length"),
    require(ascii_digits(capture.registration), "invalid_characters", "euid.lv.registration_characters"),
    require(char_at_in(capture.registration, 0, "456789"), "invalid_format", "euid.lv.registration_legal_entity"),
  ]
}

checksum "euid" "lv" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(after_first(subject(), "."), 0, 10), [9, 1, 4, 8, 3, 10, 2, 5, 7, 6], "left", "digit_value"), 11), [3, 2, 1, 0, 10, 9, 8, 7, 6, 5, 4]),
    after_first(subject(), "."), 10,
  )
}

identifier "euid" "LV" {
  canonicalizer   = canonicalizer.euid.lv
  format          = format.euid.lv
  checksum        = checksum.euid.lv
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
