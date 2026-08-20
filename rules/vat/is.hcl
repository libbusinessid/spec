# Iceland - VAT identification number (VSK number)
#
# Five or six digits for the VSK registration itself, and ten for the national
# identifier a business may register under.
#
# Only the ten digit form carries a check, the same weighted modulo eleven
# Norway uses, with a remainder of one leaving no check digit. The shorter forms
# report unsupported: the authority publishes no algorithm over them.

canonicalizer "vat" "is" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "is" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.is.empty"),
    require(
      any(length_eq(subject(), 7), length_eq(subject(), 8), length_eq(subject(), 12)),
      "invalid_length",
      "vat.is.length",
    ),
    require(starts_with(subject(), "IS"), "invalid_format", "vat.is.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.is.characters"),
  ]
}

checksum "vat" "is" {
  rule = choose(
    when_checksum(
      not(length_eq(subject(), 12)),
      unsupported_checksum("checksum_not_published"),
    ),
    compare_digit(
      remainder_map(
        modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 8), [3, 2, 7, 6, 5, 4, 3, 2], "left", "digit_value"), 11),
        [0, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1],
      ),
      slice_from(subject(), 2), 8,
    ),
  )
}

identifier "vat" "IS" {
  canonicalizer   = canonicalizer.vat.is
  format          = format.vat.is
  checksum        = checksum.vat.is
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
    id               = "is-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "IS"
    language         = "en"
    notes            = "Weights 3, 2, 7, 6, 5, 4, 3, 2 over the first eight digits of the ten digit form, the check digit being eleven minus the remainder, zero when that reaches eleven, and never issued when it reaches ten."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
