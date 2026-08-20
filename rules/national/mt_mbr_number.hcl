# Malta - registration number
#
# The registration part of the Malta EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "mt" "mbr_number" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "mt" "mbr_number" {
  checks = [
    require(not(is_empty(subject())), "empty", "mt.mbr_number.empty"),
    require(length_between(subject(), 5, 7), "invalid_length", "mt.mbr_number.length"),
    require(starts_with(subject(), "C"), "invalid_format", "mt.mbr_number.prefix"),
    require(ascii_digits(slice_from(subject(), 1)), "invalid_characters", "mt.mbr_number.characters"),
  ]
}

dispatcher "mbr_number" {
  aliases           = ["mt_mbr_number"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "MT"
    identifier                       = identifier.mbr_number.MT
    allow_unprefixed_without_country = true
  }
}

identifier "mbr_number" "MT" {
  canonicalizer   = canonicalizer.mt.mbr_number
  format          = format.mt.mbr_number
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "mt-mbr-number"
    url              = "https://mbr.mt"
    authority        = "Malta Business Registry"
    title            = "Company registration number"
    accessed_at      = "2026-08-20"
    jurisdiction     = "MT"
    language         = "en"
    notes            = "The registry publishes a number formed of the letter C followed by digits."
    license_or_terms = "Maltese public sector information"
    tier             = "primary"
  }
}
