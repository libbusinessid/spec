# United Kingdom - VAT registration number
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

canonicalizer "vat" "gb" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "gb" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.gb.empty"),
    require(
      any(length_eq(subject(), 11), length_eq(subject(), 14)),
      "invalid_length",
      "vat.gb.length",
    ),
    require(starts_with(subject(), "GB"), "invalid_format", "vat.gb.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.gb.characters"),
  ]
}

checksum "vat" "gb" {
  rule = any_check(
    compare_constant(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 9), [8, 7, 6, 5, 4, 3, 2, 10, 1], "left", "digit_value"), 97), 0),
    compare_constant(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 9), [8, 7, 6, 5, 4, 3, 2, 10, 1], "left", "digit_value"), 97), 42),
  )
}

identifier "vat" "GB" {
  canonicalizer   = canonicalizer.vat.gb
  format          = format.vat.gb
  checksum        = checksum.vat.gb
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
    id               = "gb-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "GB"
    language         = "en"
    notes            = "Weights 8 to 2 over the first seven digits, the last two digits added, the total divisible by ninety seven. A later variant adds fifty five before the modulo."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
