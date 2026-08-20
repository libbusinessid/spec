# Czechia - identification number (ICO)
#
# The registration part of the Czechia EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "cz" "ico" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "cz" "ico" {
  checks = [
    require(not(is_empty(subject())), "empty", "cz.ico.empty"),
    require(length_eq(subject(), 8), "invalid_length", "cz.ico.length"),
    require(ascii_digits(subject()), "invalid_characters", "cz.ico.characters"),
  ]
}

checksum "cz" "ico" {
  rule = compare_digit(
    remainder_map(modulo(weighted_sum(slice(subject(), 0, 7), [8, 7, 6, 5, 4, 3, 2], "left", "digit_value"), 11), [1, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1]),
    subject(), 7,
  )
}

dispatcher "ico" {
  aliases           = ["cz_ico", "sk_ico"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  # Czechia and Slovakia both call their register number ICO, and both issue
  # eight digits with the same check. They are two registers all the same, so
  # the kind carries two targets rather than one rule pretending to cover both.
  target {
    country_code = "CZ"
    identifier   = identifier.ico.CZ
  }

  target {
    country_code = "SK"
    identifier   = identifier.ico.SK
  }
}

identifier "ico" "CZ" {
  canonicalizer   = canonicalizer.cz.ico
  format          = format.cz.ico
  checksum        = checksum.cz.ico
  default_profile = "compatible"

  source {
    id               = "cz-ares-ico"
    url              = "https://ares.gov.cz"
    authority        = "Ministerstvo financi Ceske republiky - ARES"
    title            = "Administrativni registr ekonomickych subjektu"
    accessed_at      = "2026-08-20"
    jurisdiction     = "CZ"
    language         = "cs"
    notes            = "The register publishes the eight digit ICO. It does not publish the algorithm closing it."
    license_or_terms = "Czech public sector information"
    tier             = "primary"
  }
  source {
    id               = "cz-ico-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "CZ"
    language         = "en"
    notes            = "Weights 8 to 2 over the first seven digits, remainder modulo 11, the check digit being 1 for a remainder of 0 or 10, 0 for a remainder of 1, and 11 minus the remainder otherwise."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
