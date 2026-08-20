# Netherlands - KvK number
#
# The registration part of the Netherlands EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "nl" "kvk" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "nl" "kvk" {
  checks = [
    require(not(is_empty(subject())), "empty", "nl.kvk.empty"),
    require(length_eq(subject(), 8), "invalid_length", "nl.kvk.length"),
    require(ascii_digits(subject()), "invalid_characters", "nl.kvk.characters"),
  ]
}

dispatcher "kvk" {
  aliases           = ["nl_kvk", "kvk_nummer"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "NL"
    identifier                       = identifier.kvk.NL
    allow_unprefixed_without_country = true
  }
}

identifier "kvk" "NL" {
  canonicalizer   = canonicalizer.nl.kvk
  format          = format.nl.kvk
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "nl-kvk-nummer"
    url              = "https://www.kvk.nl"
    authority        = "Kamer van Koophandel (KVK)"
    title            = "Handelsregister - KVK-nummer"
    accessed_at      = "2026-08-20"
    jurisdiction     = "NL"
    language         = "nl"
    notes            = "The chamber publishes the eight digit KVK number."
    license_or_terms = "Dutch public sector information"
    tier             = "primary"
  }
}
