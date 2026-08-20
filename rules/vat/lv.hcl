# Latvia - VAT identification number (PVN)
#
# Eleven digits. A first digit of four or above marks a legal person; below
# that, the number is the personal code of someone carrying on an economic
# activity, who registers for VAT under it.
#
# Both are business identifiers. The check applies to the legal person form; a
# personal code closes on its own algorithm, which is not published as a VAT
# check, so it reports unsupported.

canonicalizer "vat" "lv" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "lv" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.lv.empty"),
    require(length_eq(subject(), 13), "invalid_length", "vat.lv.length"),
    require(starts_with(subject(), "LV"), "invalid_format", "vat.lv.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.lv.characters"),
  ]
}

checksum "vat" "lv" {
  rule = choose(
    when_checksum(
      not(char_at_in(slice_from(subject(), 2), 0, "456789")),
      unsupported_checksum("checksum_not_published"),
    ),
    compare_digit(
      remainder_map(
        modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 10), [9, 1, 4, 8, 3, 10, 2, 5, 7, 6], "left", "digit_value"), 11),
        [3, 2, 1, 0, 10, 9, 8, 7, 6, 5, 4],
      ),
      slice_from(subject(), 2), 10,
    ),
  )
}

identifier "vat" "LV" {
  canonicalizer   = canonicalizer.vat.lv
  format          = format.vat.lv
  checksum        = checksum.vat.lv
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
    id               = "lv-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LV"
    language         = "en"
    notes            = "Weights 9, 1, 4, 8, 3, 10, 2, 5, 7, 6 over the first ten digits, the check digit being three minus the sum modulo eleven, a result of ten never being issued."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
