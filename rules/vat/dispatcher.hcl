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

  target {
    country_code      = "IT"
    accepted_prefixes = ["IT"]
    canonical_prefix  = "IT"
    identifier        = identifier.vat.IT
  }

  target {
    country_code      = "DK"
    accepted_prefixes = ["DK"]
    canonical_prefix  = "DK"
    identifier        = identifier.vat.DK
  }

  target {
    country_code      = "FI"
    accepted_prefixes = ["FI"]
    canonical_prefix  = "FI"
    identifier        = identifier.vat.FI
  }

  target {
    country_code      = "SI"
    accepted_prefixes = ["SI"]
    canonical_prefix  = "SI"
    identifier        = identifier.vat.SI
  }

  target {
    country_code      = "PT"
    accepted_prefixes = ["PT"]
    canonical_prefix  = "PT"
    identifier        = identifier.vat.PT
  }

  target {
    country_code      = "PL"
    accepted_prefixes = ["PL"]
    canonical_prefix  = "PL"
    identifier        = identifier.vat.PL
  }

  target {
    country_code      = "EE"
    accepted_prefixes = ["EE"]
    canonical_prefix  = "EE"
    identifier        = identifier.vat.EE
  }

  target {
    country_code      = "MT"
    accepted_prefixes = ["MT"]
    canonical_prefix  = "MT"
    identifier        = identifier.vat.MT
  }

  target {
    country_code      = "LU"
    accepted_prefixes = ["LU"]
    canonical_prefix  = "LU"
    identifier        = identifier.vat.LU
  }
}
