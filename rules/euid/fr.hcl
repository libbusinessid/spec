# France - EUID (European Unique Identifier)
#
# Commission Implementing Regulation (EU) 2015/884 builds the EUID from the
# country code, the register identifier and the registration number, separated
# by a dot. In France the registration number of the trade and companies
# register is the SIREN, so the registration part is validated by reusing the
# SIREN format and checksum rules on a captured view.

canonicalizer "euid" "fr" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "fr" {
  capture "register" {
    value = before_first(after_first(subject(), "FR"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.fr.empty"),
    require(starts_with(subject(), "FR"), "invalid_format", "euid.fr.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.fr.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.fr.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.fr.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.fr.register_characters"),
  ]

  use_format {
    rule  = format.fr.siren
    input = capture.registration
  }
}

checksum "euid" "fr" {
  rule = apply_checksum(checksum.fr.siren, after_first(subject(), "."))
}

dispatcher "euid" {
  pre_canonicalizer = canonicalizer.dispatch.structured

  target {
    country_code      = "FR"
    accepted_prefixes = ["FR"]
    canonical_prefix  = "FR"
    identifier        = identifier.euid.FR
  }
}

identifier "euid" "FR" {
  canonicalizer   = canonicalizer.euid.fr
  format          = format.euid.fr
  checksum        = checksum.euid.fr
  default_profile = "compatible"

  source {
    id               = "eu-2015-884-euid"
    url              = "https://eur-lex.europa.eu/eli/reg_impl/2015/884/oj"
    authority        = "European Commission"
    title            = "Commission Implementing Regulation (EU) 2015/884 laying down technical specifications and procedures required for the system of interconnection of registers"
    accessed_at      = "2026-08-18"
    jurisdiction     = "EU"
    language         = "en"
    notes            = "The EUID is composed of the country code, the register identifier and the registration number separated by a dot."
    license_or_terms = "EUR-Lex reuse policy, Decision 2011/833/EU"
  }

  source {
    id               = "fr-insee-siren"
    url              = "https://www.insee.fr/fr/information/2015441"
    authority        = "Institut national de la statistique et des etudes economiques (INSEE)"
    title            = "Le repertoire Sirene - numeros SIREN et SIRET"
    accessed_at      = "2026-08-18"
    jurisdiction     = "FR"
    language         = "fr"
    notes            = "The French registration number of the trade and companies register is the SIREN."
    license_or_terms = "Licence Ouverte / Open Licence (Etalab), public sector information"
  }
}
