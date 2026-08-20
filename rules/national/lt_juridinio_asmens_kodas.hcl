# Lithuania - legal entity code
#
# The registration part of the Lithuania EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "lt" "juridinio_asmens_kodas" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "lt" "juridinio_asmens_kodas" {
  checks = [
    require(not(is_empty(subject())), "empty", "lt.juridinio_asmens_kodas.empty"),
    require(length_eq(subject(), 9), "invalid_length", "lt.juridinio_asmens_kodas.length"),
    require(ascii_digits(subject()), "invalid_characters", "lt.juridinio_asmens_kodas.characters"),
  ]
}

checksum "lt" "juridinio_asmens_kodas" {
  rule = choose(
    when_checksum(
      integer_is(modulo(weighted_sum(slice(subject(), 0, 9), [1, 2, 3, 4, 5, 6, 7, 8, 9], "left", "digit_value"), 11), 10),
      compare_constant(
        remainder_map(modulo(weighted_sum(slice(subject(), 0, 9), [3, 4, 5, 6, 7, 8, 9, 1, 2], "left", "digit_value"), 11), [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0]),
        0,
      ),
    ),
    compare_constant(modulo(weighted_sum(slice(subject(), 0, 9), [1, 2, 3, 4, 5, 6, 7, 8, 9], "left", "digit_value"), 11), 0),
  )
}

dispatcher "juridinio_asmens_kodas" {
  aliases           = ["lt_juridinio_asmens_kodas"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "LT"
    identifier                       = identifier.juridinio_asmens_kodas.LT
    allow_unprefixed_without_country = true
  }
}

identifier "juridinio_asmens_kodas" "LT" {
  canonicalizer   = canonicalizer.lt.juridinio_asmens_kodas
  format          = format.lt.juridinio_asmens_kodas
  checksum        = checksum.lt.juridinio_asmens_kodas
  default_profile = "compatible"

  source {
    id               = "lt-registrucentras-kodas"
    url              = "https://www.registrucentras.lt"
    authority        = "Valstybes imone Registru centras"
    title            = "Juridiniu asmenu registras"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LT"
    language         = "lt"
    notes            = "The register publishes the nine digit code of a legal person. It does not publish the algorithm validating it."
    license_or_terms = "Lithuanian public sector information"
    tier             = "primary"
  }
  source {
    id               = "lt-kodas-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LT"
    language         = "en"
    notes            = "Weights 1 to 9 over the nine digits, the remainder modulo 11 having to vanish. When it reaches ten the sum is recomputed with the weights rotated to 3 to 9 then 1 and 2, and the new remainder modulo ten must vanish."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
