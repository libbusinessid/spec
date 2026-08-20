# Malta - VAT identification number
#
# Eight digits: six of number and a two digit check taken modulo thirty seven.

canonicalizer "vat" "mt" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "mt" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.mt.empty"),
    require(length_eq(subject(), 10), "invalid_length", "vat.mt.length"),
    require(starts_with(subject(), "MT"), "invalid_format", "vat.mt.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.mt.characters"),
  ]
}

checksum "vat" "mt" {
  rule = compare_slice(
    complement(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 6), [3, 4, 6, 7, 8, 9], "left", "digit_value"), 37), 37),
    slice_from(subject(), 2), 6, 8,
  )
}

identifier "vat" "MT" {
  canonicalizer   = canonicalizer.vat.mt
  format          = format.vat.mt
  checksum        = checksum.vat.mt
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
    id               = "mt-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "MT"
    language         = "en"
    notes            = "Weights 3, 4, 6, 7, 8, 9 over the first six digits, the two check digits being thirty seven minus the sum modulo thirty seven."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
