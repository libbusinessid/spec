# Bulgaria - unified identification code (EIK)
#
# The registration part of the Bulgaria EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "bg" "eik" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "bg" "eik" {
  checks = [
    require(not(is_empty(subject())), "empty", "bg.eik.empty"),
    require(any(length_eq(subject(), 9), length_eq(subject(), 13)), "invalid_length", "bg.eik.length"),
    require(ascii_digits(subject()), "invalid_characters", "bg.eik.characters"),
  ]
}

checksum "bg" "eik" {
  rule = choose(
    when_checksum(
      integer_is(modulo(weighted_sum(slice(subject(), 0, 8), [1, 2, 3, 4, 5, 6, 7, 8], "left", "digit_value"), 11), 10),
      compare_digit(remainder_map(modulo(weighted_sum(slice(subject(), 0, 8), [3, 4, 5, 6, 7, 8, 9, 10], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0]), subject(), 8),
    ),
    compare_digit(remainder_map(modulo(weighted_sum(slice(subject(), 0, 8), [1, 2, 3, 4, 5, 6, 7, 8], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0]), subject(), 8),
  )
}

dispatcher "eik" {
  aliases           = ["bg_eik", "bulstat"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "BG"
    identifier                       = identifier.eik.BG
    allow_unprefixed_without_country = true
  }
}

identifier "eik" "BG" {
  canonicalizer   = canonicalizer.bg.eik
  format          = format.bg.eik
  checksum        = checksum.bg.eik
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
