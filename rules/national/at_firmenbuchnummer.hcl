# Austria - Firmenbuchnummer
#
# The registration part of the Austria EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "at" "firmenbuchnummer" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/", " "]),
  ]
}

format "at" "firmenbuchnummer" {
  checks = [
    require(not(is_empty(subject())), "empty", "at.firmenbuchnummer.empty"),
    require(length_between(subject(), 2, 7), "invalid_length", "at.firmenbuchnummer.length"),
    require(ascii_alphanumeric(subject()), "invalid_characters", "at.firmenbuchnummer.characters"),
    require(
      any(
        all(
          length_eq(subject(), 2),
          ascii_digits(slice(subject(), 0, 1)),
          char_at_in(subject(), 1, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
        ),
        all(
          length_eq(subject(), 3),
          ascii_digits(slice(subject(), 0, 2)),
          char_at_in(subject(), 2, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
        ),
        all(
          length_eq(subject(), 4),
          ascii_digits(slice(subject(), 0, 3)),
          char_at_in(subject(), 3, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
        ),
        all(
          length_eq(subject(), 5),
          ascii_digits(slice(subject(), 0, 4)),
          char_at_in(subject(), 4, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
        ),
        all(
          length_eq(subject(), 6),
          ascii_digits(slice(subject(), 0, 5)),
          char_at_in(subject(), 5, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
        ),
        all(
          length_eq(subject(), 7),
          ascii_digits(slice(subject(), 0, 6)),
          char_at_in(subject(), 6, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
        ),
      ),
      "invalid_format",
      "at.firmenbuchnummer.shape",
    ),
  ]
}

dispatcher "firmenbuchnummer" {
  aliases           = ["at_firmenbuchnummer", "fn"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "AT"
    identifier                       = identifier.firmenbuchnummer.AT
    allow_unprefixed_without_country = true
  }
}

identifier "firmenbuchnummer" "AT" {
  canonicalizer   = canonicalizer.at.firmenbuchnummer
  format          = format.at.firmenbuchnummer
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "at-firmenbuch-nummer"
    url              = "https://www.justiz.gv.at"
    authority        = "Bundesministerium fuer Justiz"
    title            = "Firmenbuch"
    accessed_at      = "2026-08-20"
    jurisdiction     = "AT"
    language         = "de"
    notes            = "The register publishes the Firmenbuchnummer as digits followed by a single letter."
    license_or_terms = "Austrian public sector information"
    tier             = "primary"
  }
}
