# Belgium - EUID
#
# The registration part is the Belgian enterprise number of the Banque-Carrefour
# des Entreprises: ten digits whose first is 0 or 1, closed by a two digit check
# equal to 97 minus the first eight digits modulo 97. It is the same number as
# the VAT number without its BE prefix, so the two rules share an algorithm but
# not a definition: the enterprise number is issued by the BCE, the VAT number
# by the tax administration.

canonicalizer "euid" "be" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "be" {
  capture "register" {
    value = before_first(after_first(subject(), "BE"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.be.empty"),
    require(starts_with(subject(), "BE"), "invalid_format", "euid.be.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.be.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.be.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.be.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.be.register_characters"),
  ]

  # The registration part is the enterprise number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.be.enterprise_number
    input = capture.registration
  }
}

checksum "euid" "be" {
  rule = apply_checksum(checksum.be.enterprise_number, after_first(subject(), "."))
}

identifier "euid" "BE" {
  canonicalizer   = canonicalizer.euid.be
  format          = format.euid.be
  checksum        = checksum.euid.be
  default_profile = "compatible"

  source {
    id               = "be-bce-enterprise-number"
    url              = "https://economie.fgov.be/fr/themes/entreprises/banque-carrefour-des"
    authority        = "Service public federal Economie, P.M.E., Classes moyennes et Energie"
    title            = "Structure du numero d'entreprise"
    accessed_at      = "2026-08-20"
    jurisdiction     = "BE"
    language         = "fr"
    notes            = "Positions 1 to 8 are the sequential number, positions 9 and 10 the check, equal to 97 minus the first eight digits modulo 97. The first digit is 0 or 1."
    license_or_terms = "Belgian federal public sector information, reuse permitted with attribution"
    tier             = "primary"
  }
}
