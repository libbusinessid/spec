# D-U-N-S number
#
# Nine digits issued by Dun & Bradstreet, unique to an establishment worldwide.
# It is a global identifier: no country routes it, and the issuer publishes no
# check algorithm over it.

canonicalizer "global" "duns" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    remove_chars(["-", "."]),
  ]
}

format "global" "duns" {
  checks = [
    require(not(is_empty(subject())), "empty", "duns.empty"),
    require(length_eq(subject(), 9), "invalid_length", "duns.length"),
    require(ascii_digits(subject()), "invalid_characters", "duns.characters"),
  ]
}

dispatcher "duns" {
  aliases           = ["dnb", "duns_number"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    identifier = identifier.duns.GLOBAL
  }
}

identifier "duns" "GLOBAL" {
  canonicalizer   = canonicalizer.global.duns
  format          = format.global.duns
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "Dun & Bradstreet publishes the nine digit number and no algorithm over it. A ninth digit check circulated in older documentation and was withdrawn, so none is applied."
  }

  source {
    id               = "global-dnb-duns"
    url              = "https://www.dnb.com/duns.html"
    authority        = "Dun & Bradstreet"
    title            = "D-U-N-S Number"
    accessed_at      = "2026-08-20"
    jurisdiction     = "GLOBAL"
    language         = "en"
    notes            = "Nine digits identifying a single business establishment, issued and maintained by Dun & Bradstreet."
    license_or_terms = "Published by the issuer for public reference"
    tier             = "primary"
  }
}
