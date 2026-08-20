# Luxembourg - VAT identification number
#
# Eight digits, the last two being the first six read as a number and taken
# modulo eighty nine.

canonicalizer "vat" "lu" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "lu" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.lu.empty"),
    require(length_eq(subject(), 10), "invalid_length", "vat.lu.length"),
    require(starts_with(subject(), "LU"), "invalid_format", "vat.lu.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.lu.characters"),
  ]
}

checksum "vat" "lu" {
  rule = compare_slice(
    modulo(digits_to_integer(slice(slice_from(subject(), 2), 0, 6)), 89),
    slice_from(subject(), 2), 6, 8,
  )
}

identifier "vat" "LU" {
  canonicalizer   = canonicalizer.vat.lu
  format          = format.vat.lu
  checksum        = checksum.vat.lu
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
    id               = "lu-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LU"
    language         = "en"
    notes            = "The first six digits read as a number, taken modulo eighty nine, give the last two digits."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
