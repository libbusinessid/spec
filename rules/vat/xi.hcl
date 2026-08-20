# Northern Ireland - VAT registration number
#
# The XI prefix identifies a Northern Ireland trader under the Windsor
# Framework. The number itself is a United Kingdom registration number and
# follows the same check.
#
# Nine digits, optionally followed by a three digit branch number.
#
# The published check adds the last two digits to a weighted sum of the first
# seven and asks the total to divide by ninety seven. Those last two digits are
# folded into the weighted sum here, as the weights 10 and 1, so the whole check
# is one operation rather than a sum and an addition the IR has no way to write.
#
# Two variants circulate: the original, and one adding fifty five before the
# modulo. Adding fifty five and asking for a remainder of zero is asking for a
# remainder of forty two, so the rule accepts either remainder rather than
# computing two sums.

canonicalizer "vat" "xi" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "xi" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.xi.empty"),
    require(
      any(length_eq(subject(), 11), length_eq(subject(), 14)),
      "invalid_length",
      "vat.xi.length",
    ),
    require(starts_with(subject(), "XI"), "invalid_format", "vat.xi.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.xi.characters"),
  ]
}

checksum "vat" "xi" {
  rule = any_check(
    compare_constant(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 9), [8, 7, 6, 5, 4, 3, 2, 10, 1], "left", "digit_value"), 97), 0),
    compare_constant(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 9), [8, 7, 6, 5, 4, 3, 2, 10, 1], "left", "digit_value"), 97), 42),
  )
}

identifier "vat" "XI" {
  canonicalizer   = canonicalizer.vat.xi
  format          = format.vat.xi
  checksum        = checksum.vat.xi
  default_profile = "compatible"

  source {
    id               = "eu-vies-number-structure"
    url              = "https://ec.europa.eu/taxation_customs/vies/"
    authority        = "European Commission, Directorate-General for Taxation and Customs Union"
    title            = "VIES - VAT number structure per Member State"
    accessed_at      = "2026-08-20"
    jurisdiction     = "EU"
    language         = "en"
    notes            = "Published length and prefix of each Member State VAT identification number."
    license_or_terms = "European Commission reuse policy, Decision 2011/833/EU"
    tier             = "primary"
  }

  source {
    id               = "xi-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "XI"
    language         = "en"
    notes            = "The same check as the United Kingdom number: weights 8 to 2 over the first seven digits, the last two added, the total divisible by ninety seven, with a variant adding fifty five."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
