# Czechia - VAT identification number (DIC)
#
# Eight digits for a legal person, the ICO of the register, and nine or ten for
# a self employed person, whose tax number is built on their birth number. All
# three are business identifiers, and a sole trader invoices under one of them.
#
# The check applies to the eight digit form. The longer ones close on the birth
# number algorithm, which is not published as a tax number check, so they report
# unsupported.

canonicalizer "vat" "cz" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "cz" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.cz.empty"),
    require(length_between(subject(), 10, 12), "invalid_length", "vat.cz.length"),
    require(starts_with(subject(), "CZ"), "invalid_format", "vat.cz.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.cz.characters"),
  ]
}

checksum "vat" "cz" {
  rule = choose(
    when_checksum(
      not(length_eq(subject(), 10)),
      unsupported_checksum("checksum_not_published"),
    ),
    compare_digit(
      remainder_map(
        modulo(weighted_sum(slice(slice_from(subject(), 2), 0, 7), [8, 7, 6, 5, 4, 3, 2], "left", "digit_value"), 11),
        [1, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1],
      ),
      slice_from(subject(), 2), 7,
    ),
  )
}

identifier "vat" "CZ" {
  canonicalizer   = canonicalizer.vat.cz
  format          = format.vat.cz
  checksum        = checksum.vat.cz
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
    id               = "cz-vat-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "CZ"
    language         = "en"
    notes            = "Weights 8 to 2 over the first seven digits, the check digit being one for a remainder of zero or ten, zero for a remainder of one, and eleven minus the remainder otherwise."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
