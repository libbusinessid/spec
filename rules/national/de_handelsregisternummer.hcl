# Germany - Handelsregisternummer
#
# The registration part of the Germany EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "de" "handelsregisternummer" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "de" "handelsregisternummer" {
  checks = [
    require(not(is_empty(subject())), "empty", "de.handelsregisternummer.empty"),
    require(not(is_empty(subject())), "empty", "de.handelsregisternummer.empty"),
    require(length_between(subject(), 1, 20), "invalid_length", "de.handelsregisternummer.length"),
    require(ascii_alphanumeric(subject()), "invalid_characters", "de.handelsregisternummer.characters"),
  ]
}

dispatcher "handelsregisternummer" {
  aliases           = ["de_handelsregisternummer", "hrb"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "DE"
    identifier                       = identifier.handelsregisternummer.DE
    allow_unprefixed_without_country = true
  }
}

identifier "handelsregisternummer" "DE" {
  canonicalizer   = canonicalizer.de.handelsregisternummer
  format          = format.de.handelsregisternummer
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "de-handelsregister"
    url              = "https://www.handelsregister.de"
    authority        = "Justizministerien der Laender"
    title            = "Gemeinsames Registerportal der Laender"
    accessed_at      = "2026-08-20"
    jurisdiction     = "DE"
    language         = "de"
    notes            = "The portal publishes register entries as a register kind and a number held by a local court. It states no single national shape for the number and no algorithm over it."
    license_or_terms = "German public sector information"
    tier             = "primary"
  }
}
