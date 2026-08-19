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
}
