# # Portugal - EUID\n#\n# The registration part is the nine digit NIPC. Its last digit closes a\n# weighted modulo 11 over the first eight, and a remainder below two yields a\n# check digit of zero rather than a value above nine.

canonicalizer "euid" "pt" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "pt" {
  capture "register" {
    value = before_first(after_first(subject(), "PT"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.pt.empty"),
    require(starts_with(subject(), "PT"), "invalid_format", "euid.pt.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.pt.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.pt.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.pt.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.pt.register_characters"),
    require(length_eq(capture.registration, 9), "invalid_length", "euid.pt.registration_length"),
    require(ascii_digits(capture.registration), "invalid_characters", "euid.pt.registration_characters"),
  ]
}

checksum "euid" "pt" {
  rule = compare_digit(
    remainder_map(
      modulo(
        weighted_sum(slice(after_first(subject(), "."), 0, 8), [9, 8, 7, 6, 5, 4, 3, 2], "left", "digit_value"),
        11,
      ),
      [0, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1],
    ),
    after_first(subject(), "."), 8,
  )
}

identifier "euid" "PT" {
  canonicalizer   = canonicalizer.euid.pt
  format          = format.euid.pt
  checksum        = checksum.euid.pt
  default_profile = "compatible"

  source {
    id               = "pt-justica-nipc"
    url              = "https://www2.gov.pt"
    authority        = "Republica Portuguesa - Governo"
    title            = "Numero de identificacao de pessoa coletiva"
    accessed_at      = "2026-08-20"
    jurisdiction     = "PT"
    language         = "pt"
    notes            = "The state portal publishes the nine digit NIPC. It does not publish the algorithm closing it."
    license_or_terms = "Portuguese public sector information"
    tier             = "primary"
  }

  source {
    id               = "pt-nipc-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "PT"
    language         = "en"
    notes            = "Weights 9 to 2 over the first eight digits, remainder modulo 11, the check digit being 11 minus the remainder when the remainder is at least two and zero otherwise."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
