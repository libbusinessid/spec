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
    require(length_eq(capture.register, 4), "invalid_length", "euid.fr.register_length"),
    require(ascii_digits(capture.register), "invalid_characters", "euid.fr.register_characters"),

    # The register identifier designates the greffe of registration, and the
    # registrars publish the complete list. Checking membership rather than
    # shape is what makes the country code and the code that follows it agree:
    # FRZZZZ and FRQ were accepted before, and no greffe carries either.
    #
    # The capture is exactly four digits by the check above, and every code
    # here is exactly four digits, so starting with one of them is equalling
    # it. No set membership operation is needed.
    require(
      prefix_in(capture.register, [
        "0101", "0202", "0203", "0301", "0303", "0401", "0501", "0601", "0602", "0603", "0605", "0702",
        "0802", "0901", "1001", "1101", "1104", "1203", "1301", "1303", "1304", "1305", "1402", "1407",
        "1501", "1601", "1704", "1708", "1801", "1901", "2001", "2002", "2104", "2202", "2301", "2401",
        "2402", "2501", "2602", "2701", "2702", "2801", "2901", "2903", "3003", "3102", "3201", "3302",
        "3303", "3402", "3405", "3501", "3502", "3601", "3701", "3801", "3802", "3902", "4001", "4002",
        "4101", "4201", "4202", "4302", "4401", "4402", "4502", "4601", "4701", "4801", "4901", "5001",
        "5002", "5101", "5103", "5201", "5301", "5401", "5402", "5501", "5601", "5602", "5751", "5752",
        "5753", "5802", "5902", "5906", "5910", "5952", "6001", "6002", "6101", "6201", "6202", "6303",
        "6401", "6403", "6502", "6601", "6751", "6752", "6851", "6852", "6901", "6903", "7001", "7102",
        "7106", "7202", "7301", "7401", "7402", "7501", "7601", "7606", "7608", "7701", "7702", "7801",
        "7802", "7803", "7901", "8002", "8101", "8102", "8201", "8302", "8303", "8305", "8401", "8501",
        "8602", "8701", "8801", "8901", "8903", "9001", "9201", "9301", "9401", "9711", "9712", "9721",
        "9731", "9741", "9742", "9761",
      ]),
      "invalid_format",
      "euid.fr.register_unknown",
    ),
  ]

  use_format {
    rule  = format.fr.siren
    input = capture.registration
  }
}

checksum "euid" "fr" {
  rule = apply_checksum(checksum.fr.siren, after_first(subject(), "."))
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
    tier             = "primary"
  }

  source {
    id               = "fr-cngtc-greffes"
    url              = "https://www.data.gouv.fr/datasets/liste-des-greffes"
    authority        = "Conseil national des greffiers des tribunaux de commerce, diffused by Infogreffe"
    title            = "Liste des greffes"
    accessed_at      = "2026-08-24"
    jurisdiction     = "FR"
    language         = "fr"
    notes            = "The 148 greffe codes carried by the register identifier, from the code_greffe column. Every code is exactly four digits and all are distinct; 3102 is Toulouse. The dataset content needs an account on the publisher portal, so the codes are transcribed here rather than fetched by the build."
    license_or_terms = "Licence Ouverte / Open Licence v2.0"
    tier             = "primary"
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
    tier             = "primary"
  }
}
