# Lithuania - EUID
#
# The registration part is the nine digit juridinio asmens kodas. The weighted
# modulo 11 runs over all nine digits and must vanish, and when the first
# remainder reaches ten the sum is recomputed with a rotated set of weights.

canonicalizer "euid" "lt" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "lt" {
  capture "register" {
    value = before_first(after_first(subject(), "LT"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.lt.empty"),
    require(starts_with(subject(), "LT"), "invalid_format", "euid.lt.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.lt.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.lt.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.lt.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.lt.register_characters"),
    require(length_eq(capture.registration, 9), "invalid_length", "euid.lt.registration_length"),
    require(ascii_digits(capture.registration), "invalid_characters", "euid.lt.registration_characters"),
  ]
}

checksum "euid" "lt" {
  rule = choose(
    when_checksum(
      integer_is(modulo(weighted_sum(slice(after_first(subject(), "."), 0, 9), [1, 2, 3, 4, 5, 6, 7, 8, 9], "left", "digit_value"), 11), 10),
      compare_constant(
        remainder_map(modulo(weighted_sum(slice(after_first(subject(), "."), 0, 9), [3, 4, 5, 6, 7, 8, 9, 1, 2], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0]),
        0,
      ),
    ),
    compare_constant(modulo(weighted_sum(slice(after_first(subject(), "."), 0, 9), [1, 2, 3, 4, 5, 6, 7, 8, 9], "left", "digit_value"), 11), 0),
  )
}

identifier "euid" "LT" {
  canonicalizer   = canonicalizer.euid.lt
  format          = format.euid.lt
  checksum        = checksum.euid.lt
  default_profile = "compatible"

  source {
    id               = "lt-registrucentras-kodas"
    url              = "https://www.registrucentras.lt"
    authority        = "Valstybes imone Registru centras"
    title            = "Juridiniu asmenu registras"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LT"
    language         = "lt"
    notes            = "The register publishes the nine digit code of a legal person. It does not publish the algorithm validating it."
    license_or_terms = "Lithuanian public sector information"
    tier             = "primary"
  }

  source {
    id               = "lt-kodas-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LT"
    language         = "en"
    notes            = "Weights 1 to 9 over the nine digits, the remainder modulo 11 having to vanish. When it reaches ten the sum is recomputed with the weights rotated to 3 to 9 then 1 and 2, and the new remainder modulo ten must vanish."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
