# Estonia - registry code
#
# The registration part of the Estonia EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "ee" "registrikood" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "ee" "registrikood" {
  checks = [
    require(not(is_empty(subject())), "empty", "ee.registrikood.empty"),
    require(length_eq(subject(), 8), "invalid_length", "ee.registrikood.length"),
    require(ascii_digits(subject()), "invalid_characters", "ee.registrikood.characters"),
  ]
}

checksum "ee" "registrikood" {
  rule = choose(
    when_checksum(
      integer_is(modulo(weighted_sum(slice(subject(), 0, 7), [1, 2, 3, 4, 5, 6, 7], "left", "digit_value"), 11), 10),
      compare_digit(
        remainder_map(modulo(weighted_sum(slice(subject(), 0, 7), [3, 4, 5, 6, 7, 8, 9], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0]),
        subject(), 7,
      ),
    ),
    compare_digit(
      remainder_map(modulo(weighted_sum(slice(subject(), 0, 7), [1, 2, 3, 4, 5, 6, 7], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0]),
      subject(), 7,
    ),
  )
}

dispatcher "registrikood" {
  aliases           = ["ee_registrikood"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "EE"
    identifier                       = identifier.registrikood.EE
    allow_unprefixed_without_country = true
  }
}

identifier "registrikood" "EE" {
  canonicalizer   = canonicalizer.ee.registrikood
  format          = format.ee.registrikood
  checksum        = checksum.ee.registrikood
  default_profile = "compatible"

  source {
    id               = "ee-ariregister-registrikood"
    url              = "https://ariregister.rik.ee"
    authority        = "Registrite ja Infosusteemide Keskus (RIK)"
    title            = "e-Business Register"
    accessed_at      = "2026-08-20"
    jurisdiction     = "EE"
    language         = "et"
    notes            = "The register publishes the eight digit registrikood. It does not publish the algorithm closing it."
    license_or_terms = "Estonian public sector information"
    tier             = "primary"
  }
  source {
    id               = "ee-registrikood-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "EE"
    language         = "en"
    notes            = "Weights 1 to 7 over the first seven digits, remainder modulo 11. When that remainder is ten the sum is recomputed with weights 3 to 9 and the check digit is the new remainder modulo ten."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
