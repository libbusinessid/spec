# Latvia - EUID
#
# The registration part is the eleven digit registration number of a legal
# entity, whose first digit is four or above.
#
# A number starting below four is a personal code, which identifies a natural
# person rather than a company. It is not accepted here: an EUID designates an
# entity of a business register, and validating personal codes would put the
# corpus in the business of checking people.
#
# The check digit is three minus the weighted sum, modulo eleven. A result of
# ten is not issued, and the remainder table maps it to ten so that no single
# digit can match it.
#
# Note on scope. The VAT rule of this country accepts the identifier of a sole
# trader, because such a person invoices under it and refusing it would refuse a
# real business identifier. The register rule stays restricted to legal entities for the same reason: the EUID comes from the register of companies, which a person carrying on an activity does not enter. If that proves wrong for some Member State, the restriction should be revisited.

canonicalizer "euid" "lv" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "lv" {
  capture "register" {
    value = before_first(after_first(subject(), "LV"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.lv.empty"),
    require(starts_with(subject(), "LV"), "invalid_format", "euid.lv.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.lv.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.lv.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.lv.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.lv.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.lv.registracijas_numurs
    input = capture.registration
  }
}

checksum "euid" "lv" {
  rule = apply_checksum(checksum.lv.registracijas_numurs, after_first(subject(), "."))
}

identifier "euid" "LV" {
  canonicalizer   = canonicalizer.euid.lv
  format          = format.euid.lv
  checksum        = checksum.euid.lv
  default_profile = "compatible"

  source {
    id               = "lv-ur-registration-number"
    url              = "https://www.ur.gov.lv"
    authority        = "Latvijas Republikas Uznemumu registrs"
    title            = "Registration number of a legal entity"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LV"
    language         = "lv"
    notes            = "The register publishes an eleven digit registration number whose first digit distinguishes a legal entity from a personal code."
    license_or_terms = "Latvian public sector information"
    tier             = "primary"
  }

  source {
    id               = "lv-number-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LV"
    language         = "en"
    notes            = "Weights 9, 1, 4, 8, 3, 10, 2, 5, 7, 6 over the first ten digits, the check digit being three minus the sum modulo eleven, and a result of ten never being issued."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
