# Greece - GEMI number
#
# The registration part of the Greece EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "el" "gemi" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "el" "gemi" {
  checks = [
    require(not(is_empty(subject())), "empty", "el.gemi.empty"),
    require(length_eq(subject(), 12), "invalid_length", "el.gemi.length"),
    require(ascii_digits(subject()), "invalid_characters", "el.gemi.characters"),
  ]
}

dispatcher "gemi" {
  aliases           = ["el_gemi", "gr_gemi"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "EL"
    identifier                       = identifier.gemi.EL
    allow_unprefixed_without_country = true
  }
}

identifier "gemi" "EL" {
  canonicalizer   = canonicalizer.el.gemi
  format          = format.el.gemi
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "el-gemi-number"
    url              = "https://www.businessportal.gr"
    authority        = "Geniko Emporiko Mitroo (GEMI)"
    title            = "General Commercial Registry"
    accessed_at      = "2026-08-20"
    jurisdiction     = "GR"
    language         = "el"
    notes            = "The registry publishes the twelve digit GEMI number."
    license_or_terms = "Greek public sector information"
    tier             = "primary"
  }
}
