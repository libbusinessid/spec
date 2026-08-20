# Denmark - CVR number
#
# The registration part of the Denmark EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "dk" "cvr" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "dk" "cvr" {
  checks = [
    require(not(is_empty(subject())), "empty", "dk.cvr.empty"),
    require(length_eq(subject(), 8), "invalid_length", "dk.cvr.length"),
    require(ascii_digits(subject()), "invalid_characters", "dk.cvr.characters"),
    require(not(starts_with(subject(), "0")), "invalid_format", "dk.cvr.leading"),
  ]
}

checksum "dk" "cvr" {
  rule = compare_constant(
    modulo(
      weighted_sum(slice(subject(), 0, 8), [2, 7, 6, 5, 4, 3, 2, 1], "left", "digit_value"),
      11,
    ),
    0,
  )
}

dispatcher "cvr" {
  aliases           = ["dk_cvr", "cvr_nummer"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "DK"
    identifier                       = identifier.cvr.DK
    allow_unprefixed_without_country = true
  }
}

identifier "cvr" "DK" {
  canonicalizer   = canonicalizer.dk.cvr
  format          = format.dk.cvr
  checksum        = checksum.dk.cvr
  default_profile = "compatible"

  source {
    id               = "dk-erhvervsstyrelsen-cvr"
    url              = "https://erhvervsstyrelsen.dk/cvr-numre-p-numre-og-se-numre"
    authority        = "Erhvervsstyrelsen (Danish Business Authority)"
    title            = "CVR-numre, p-numre og se-numre"
    accessed_at      = "2026-08-20"
    jurisdiction     = "DK"
    language         = "da"
    notes            = "The authority publishes the eight digit CVR number as the key of the central business register. It does not publish the algorithm validating it."
    license_or_terms = "Danish public sector information"
    tier             = "primary"
  }
  source {
    id               = "dk-cvr-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "DK"
    language         = "en"
    notes            = "Weights 2, 7, 6, 5, 4, 3, 2, 1 over the eight digits, the whole sum being divisible by eleven. The first digit is never zero."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
