# Slovakia - VAT identification number (IC DPH)
#
# Ten digits read as a single number, which must divide by eleven. The iterative
# remainder the published description uses is the same value as the whole number
# taken modulo eleven, and ten digits stay well inside the integer bounds.

canonicalizer "vat" "sk" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "sk" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.sk.empty"),
    require(length_eq(subject(), 12), "invalid_length", "vat.sk.length"),
    require(starts_with(subject(), "SK"), "invalid_format", "vat.sk.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.sk.characters"),
  ]
}

checksum "vat" "sk" {
  rule = compare_constant(
    modulo(digits_to_integer(slice(slice_from(subject(), 2), 0, 10)), 11),
    0,
  )
}

identifier "vat" "SK" {
  canonicalizer   = canonicalizer.vat.sk
  format          = format.vat.sk
  checksum        = checksum.vat.sk
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
    id               = "sk-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "SK"
    language         = "en"
    notes            = "The ten digits read as one number, which must be divisible by eleven."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
