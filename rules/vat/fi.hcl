# Finland - VAT identification number (ALV-numero)
#
# The eight digits of the Y-tunnus. A remainder of one admits no check digit and
# such a number is never issued, so the remainder table sends that case to ten,
# which no single digit can equal.

canonicalizer "vat" "fi" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "fi" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.fi.empty"),
    require(length_eq(subject(), 10), "invalid_length", "vat.fi.length"),
    require(starts_with(subject(), "FI"), "invalid_format", "vat.fi.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.fi.characters"),
  ]
}

checksum "vat" "fi" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 7), [7, 9, 10, 5, 8, 4, 2], "left", "digit_value"), 11), [0, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1]),
    slice_from(subject(), 2), 7,
  )
}

identifier "vat" "FI" {
  canonicalizer   = canonicalizer.vat.fi
  format          = format.vat.fi
  checksum        = checksum.vat.fi
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
    id               = "fi-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "FI"
    language         = "en"
    notes            = "Weights 7, 9, 10, 5, 8, 4, 2 over the first seven digits. A remainder of zero gives zero, above one gives eleven minus it, and one is not issued."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
