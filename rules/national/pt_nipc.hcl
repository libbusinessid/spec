# Portugal - NIPC
#
# The registration part of the Portugal EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "pt" "nipc" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "pt" "nipc" {
  checks = [
    require(not(is_empty(subject())), "empty", "pt.nipc.empty"),
    require(length_eq(subject(), 9), "invalid_length", "pt.nipc.length"),
    require(ascii_digits(subject()), "invalid_characters", "pt.nipc.characters"),
  ]
}

checksum "pt" "nipc" {
  rule = compare_digit(
    remainder_map(
      modulo(
        weighted_sum(slice(subject(), 0, 8), [9, 8, 7, 6, 5, 4, 3, 2], "left", "digit_value"),
        11,
      ),
      [0, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1],
    ),
    subject(), 8,
  )
}

dispatcher "nipc" {
  aliases           = ["pt_nipc"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "PT"
    identifier                       = identifier.nipc.PT
    allow_unprefixed_without_country = true
  }
}

identifier "nipc" "PT" {
  canonicalizer   = canonicalizer.pt.nipc
  format          = format.pt.nipc
  checksum        = checksum.pt.nipc
  default_profile = "compatible"

  source {
    id               = "pt-justica-nipc"
    url              = "https://www2.gov.pt"
    authority        = "Republica Portuguesa - Governo"
    title            = "Numero de identificacao de pessoa coletiva"
    accessed_at      = "2026-08-20"
    jurisdiction     = "PT"
    language         = "pt"
    notes            = "The state portal publishes the nine digit NIPC. It does not publish the algorithm closing it."
    license_or_terms = "Portuguese public sector information"
    tier             = "primary"
  }
  source {
    id               = "pt-nipc-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "PT"
    language         = "en"
    notes            = "Weights 9 to 2 over the first eight digits, remainder modulo 11, the check digit being 11 minus the remainder when the remainder is at least two and zero otherwise."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
