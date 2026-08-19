# Global - Legal Entity Identifier (LEI)
#
# ISO 17442-1 defines a twenty character alphanumeric identifier whose last two
# characters are check digits verified with the ISO 7064 MOD 97-10 procedure
# applied to the base 36 expansion of the whole string.
#
# The LEI has no country component: it is dispatched through a GLOBAL target.
# A well formed country context is normalized and kept in the result, but never
# participates in the routing.

canonicalizer "lei" "generic" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars(["-"]),
  ]
}

format "lei" "generic" {
  checks = [
    require(not(is_empty(subject())), "empty", "lei.empty"),
    require(length_eq(subject(), 20), "invalid_length", "lei.length"),
    require(ascii_alphanumeric(subject()), "invalid_characters", "lei.characters"),
    require(ascii_digits(slice(subject(), 18, 20)), "invalid_characters", "lei.check_digits"),
  ]
}

checksum "lei" "generic" {
  rule = iso7064_mod97_10(subject())
}

dispatcher "lei" {
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    identifier = identifier.lei.GLOBAL
  }
}

identifier "lei" "GLOBAL" {
  canonicalizer   = canonicalizer.lei.generic
  format          = format.lei.generic
  checksum        = checksum.lei.generic
  default_profile = "compatible"

  source {
    id               = "gleif-lei-structure"
    url              = "https://www.gleif.org/en/about-lei/iso-17442-the-lei-code-structure"
    authority        = "Global Legal Entity Identifier Foundation (GLEIF)"
    title            = "ISO 17442 - the LEI code structure"
    accessed_at      = "2026-08-18"
    jurisdiction     = "GLOBAL"
    language         = "en"
    notes            = "Twenty alphanumeric characters. Characters 1 to 4 are the LOU prefix, 5 and 6 are reserved, 7 to 18 identify the entity and 19 and 20 are ISO 7064 MOD 97-10 check digits."
    license_or_terms = "GLEIF publishes the LEI data and its documentation under CC0 1.0"
  }
}
