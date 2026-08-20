# Routing table of the `euid` kind.
#
# Commission Implementing Regulation (EU) 2015/884 builds the EUID from the
# country code of the register, the register identifier and the registration
# number. The country prefix is therefore part of the value, and each target
# claims its own.
dispatcher "euid" {
  pre_canonicalizer = canonicalizer.dispatch.structured

  target {
    country_code      = "BE"
    accepted_prefixes = ["BE"]
    canonical_prefix  = "BE"
    identifier        = identifier.euid.BE
  }

  target {
    country_code      = "FR"
    accepted_prefixes = ["FR"]
    canonical_prefix  = "FR"
    identifier        = identifier.euid.FR
  }

  target {
    country_code      = "IT"
    accepted_prefixes = ["IT"]
    canonical_prefix  = "IT"
    identifier        = identifier.euid.IT
  }

  target {
    country_code      = "SE"
    accepted_prefixes = ["SE"]
    canonical_prefix  = "SE"
    identifier        = identifier.euid.SE
  }

  target {
    country_code      = "CZ"
    accepted_prefixes = ["CZ"]
    canonical_prefix  = "CZ"
    identifier        = identifier.euid.CZ
  }

  target {
    country_code      = "DK"
    accepted_prefixes = ["DK"]
    canonical_prefix  = "DK"
    identifier        = identifier.euid.DK
  }

  target {
    country_code      = "PT"
    accepted_prefixes = ["PT"]
    canonical_prefix  = "PT"
    identifier        = identifier.euid.PT
  }

  target {
    country_code      = "SK"
    accepted_prefixes = ["SK"]
    canonical_prefix  = "SK"
    identifier        = identifier.euid.SK
  }

  target {
    country_code      = "EE"
    accepted_prefixes = ["EE"]
    canonical_prefix  = "EE"
    identifier        = identifier.euid.EE
  }

  target {
    country_code      = "LT"
    accepted_prefixes = ["LT"]
    canonical_prefix  = "LT"
    identifier        = identifier.euid.LT
  }

  target {
    country_code      = "RO"
    accepted_prefixes = ["RO"]
    canonical_prefix  = "RO"
    identifier        = identifier.euid.RO
  }

  target {
    country_code      = "CY"
    accepted_prefixes = ["CY"]
    canonical_prefix  = "CY"
    identifier        = identifier.euid.CY
  }

  target {
    country_code      = "EL"
    accepted_prefixes = ["EL"]
    canonical_prefix  = "EL"
    identifier        = identifier.euid.EL
  }

  target {
    country_code      = "HR"
    accepted_prefixes = ["HR"]
    canonical_prefix  = "HR"
    identifier        = identifier.euid.HR
  }

  target {
    country_code      = "HU"
    accepted_prefixes = ["HU"]
    canonical_prefix  = "HU"
    identifier        = identifier.euid.HU
  }

  target {
    country_code      = "IE"
    accepted_prefixes = ["IE"]
    canonical_prefix  = "IE"
    identifier        = identifier.euid.IE
  }

  target {
    country_code      = "LU"
    accepted_prefixes = ["LU"]
    canonical_prefix  = "LU"
    identifier        = identifier.euid.LU
  }

  target {
    country_code      = "MT"
    accepted_prefixes = ["MT"]
    canonical_prefix  = "MT"
    identifier        = identifier.euid.MT
  }

  target {
    country_code      = "NL"
    accepted_prefixes = ["NL"]
    canonical_prefix  = "NL"
    identifier        = identifier.euid.NL
  }

  target {
    country_code      = "PL"
    accepted_prefixes = ["PL"]
    canonical_prefix  = "PL"
    identifier        = identifier.euid.PL
  }

  target {
    country_code      = "SI"
    accepted_prefixes = ["SI"]
    canonical_prefix  = "SI"
    identifier        = identifier.euid.SI
  }

  target {
    country_code      = "AT"
    accepted_prefixes = ["AT"]
    canonical_prefix  = "AT"
    identifier        = identifier.euid.AT
  }

  target {
    country_code      = "DE"
    accepted_prefixes = ["DE"]
    canonical_prefix  = "DE"
    identifier        = identifier.euid.DE
  }

  target {
    country_code      = "BG"
    accepted_prefixes = ["BG"]
    canonical_prefix  = "BG"
    identifier        = identifier.euid.BG
  }

  target {
    country_code      = "FI"
    accepted_prefixes = ["FI"]
    canonical_prefix  = "FI"
    identifier        = identifier.euid.FI
  }

  target {
    country_code      = "LV"
    accepted_prefixes = ["LV"]
    canonical_prefix  = "LV"
    identifier        = identifier.euid.LV
  }

  target {
    country_code      = "ES"
    accepted_prefixes = ["ES"]
    canonical_prefix  = "ES"
    identifier        = identifier.euid.ES
  }
}
