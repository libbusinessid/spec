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
}
