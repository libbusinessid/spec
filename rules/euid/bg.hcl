# Bulgaria - EUID
#
# The registration part is the BULSTAT identifier, nine digits for a legal
# entity and thirteen when a branch is numbered under it. The check closes the
# first nine either way, and a remainder of ten sends the sum through a second
# set of weights - the same shape Estonia and Lithuania use.

canonicalizer "euid" "bg" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "bg" {
  capture "register" {
    value = before_first(after_first(subject(), "BG"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.bg.empty"),
    require(starts_with(subject(), "BG"), "invalid_format", "euid.bg.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.bg.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.bg.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.bg.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.bg.register_characters"),
    require(any(length_eq(capture.registration, 9), length_eq(capture.registration, 13)), "invalid_length", "euid.bg.registration_length"),
    require(ascii_digits(capture.registration), "invalid_characters", "euid.bg.registration_characters"),
  ]
}

checksum "euid" "bg" {
  rule = choose(
    when_checksum(
      integer_is(modulo(weighted_sum(slice(after_first(subject(), "."), 0, 8), [1, 2, 3, 4, 5, 6, 7, 8], "left", "digit_value"), 11), 10),
      compare_digit(remainder_map(modulo(weighted_sum(slice(after_first(subject(), "."), 0, 8), [3, 4, 5, 6, 7, 8, 9, 10], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0]), after_first(subject(), "."), 8),
    ),
    compare_digit(remainder_map(modulo(weighted_sum(slice(after_first(subject(), "."), 0, 8), [1, 2, 3, 4, 5, 6, 7, 8], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0]), after_first(subject(), "."), 8),
  )
}

identifier "euid" "BG" {
  canonicalizer   = canonicalizer.euid.bg
  format          = format.euid.bg
  checksum        = checksum.euid.bg
  default_profile = "compatible"

  source {
    id               = "bg-registryagency-bulstat"
    url              = "https://portal.registryagency.bg"
    authority        = "Agentsiya po vpisvaniyata (Registry Agency)"
    title            = "BULSTAT register"
    accessed_at      = "2026-08-20"
    jurisdiction     = "BG"
    language         = "bg"
    notes            = "The agency publishes the BULSTAT identifier as nine digits, extended to thirteen for a branch."
    license_or_terms = "Bulgarian public sector information"
    tier             = "primary"
  }

  source {
    id               = "bg-bulstat-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "BG"
    language         = "en"
    notes            = "Weights 1 to 8 over the first eight digits, remainder modulo 11. A remainder of ten sends the sum through weights 3 to 10, and a second ten yields a check digit of zero."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
