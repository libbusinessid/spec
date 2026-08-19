# Routing table of the `vat` kind.
#
# Kind aliases, country aliases and value prefixes are three separate spaces.
# The country of a result is always the ISO 3166-1 alpha-2 code of the target,
# even when the business prefix differs (country GR, canonical VAT prefix EL).
dispatcher "vat" {
  aliases           = ["vat_id", "vat_number"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  country_aliases = {
    "EL" = "GR"
    "UK" = "GB"
  }

  target {
    country_code      = "BE"
    accepted_prefixes = ["BE"]
    canonical_prefix  = "BE"
    identifier        = identifier.vat.BE
  }

  target {
    country_code      = "DE"
    accepted_prefixes = ["DE"]
    canonical_prefix  = "DE"
    identifier        = identifier.vat.DE
  }

  target {
    country_code      = "FR"
    accepted_prefixes = ["FR"]
    canonical_prefix  = "FR"
    identifier        = identifier.vat.FR
  }

  target {
    country_code      = "GR"
    accepted_prefixes = ["EL", "GR"]
    canonical_prefix  = "EL"
    identifier        = identifier.vat.GR
  }
}
