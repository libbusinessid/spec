# Greece - VAT identification number (AFM)
#
# The Greek VAT number is issued for the country GR but its VAT prefix is EL.
# The dispatcher accepts both prefixes and the canonicalizer rewrites GR into
# the canonical EL. The check digit weights the first eight digits by the
# powers of two from 256 down to 2, reduces the sum modulo 11 and maps the
# remainder 10 to 0.

canonicalizer "vat" "gr" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    replace_prefix("GR", "EL"),
    prepend_country_if_missing(),
  ]
}

format "vat" "gr" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.gr.empty"),
    require(length_eq(subject(), 11), "invalid_length", "vat.gr.length"),
    require(starts_with(subject(), "EL"), "invalid_format", "vat.gr.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.gr.characters"),
  ]
}

checksum "vat" "gr" {
  rule = compare_digit(
    remainder_map(
      modulo(
        weighted_sum(
          slice(subject(), 2, 10),
          [256, 128, 64, 32, 16, 8, 4, 2],
          "left",
          "digit_value",
        ),
        11,
      ),
      [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0],
    ),
    subject(), 10,
  )
}

identifier "vat" "GR" {
  canonicalizer   = canonicalizer.vat.gr
  format          = format.vat.gr
  checksum        = checksum.vat.gr
  default_profile = "compatible"

  source {
    id               = "gr-aade-afm"
    url              = "https://www.aade.gr/epiheiriseis/forologikes-ypiresies/mitroo/anazitisi-basikon-stoiheion-mitrooy-epiheiriseon"
    authority        = "Independent Authority for Public Revenue (AADE)"
    title            = "Tax registration number (AFM) and VAT identification"
    accessed_at      = "2026-08-18"
    jurisdiction     = "GR"
    language         = "el"
    notes            = "Nine digits. The check digit weights the first eight digits by 256, 128, 64, 32, 16, 8, 4 and 2, reduces the sum modulo 11 and maps a remainder of 10 to 0."
    license_or_terms = "Public sector information published by the Greek tax administration"
  }

  source {
    id               = "eu-vies-number-structure"
    url              = "https://ec.europa.eu/taxation_customs/vies/"
    authority        = "European Commission, Directorate-General for Taxation and Customs Union"
    title            = "VIES - VAT number structure per Member State"
    accessed_at      = "2026-08-18"
    jurisdiction     = "EU"
    language         = "en"
    notes            = "The Greek VAT prefix is EL although the ISO 3166-1 alpha-2 country code is GR."
    license_or_terms = "European Commission reuse policy, Decision 2011/833/EU"
  }
}
