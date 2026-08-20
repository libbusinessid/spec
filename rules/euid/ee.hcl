# Estonia - EUID
#
# The registration part is the eight digit registrikood. Its last digit closes a
# weighted modulo 11 over the first seven, and when that remainder reaches ten
# the sum is recomputed with a second set of weights - which is why the IR needs
# a predicate able to branch on the value of an integer.

canonicalizer "euid" "ee" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "ee" {
  capture "register" {
    value = before_first(after_first(subject(), "EE"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.ee.empty"),
    require(starts_with(subject(), "EE"), "invalid_format", "euid.ee.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.ee.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.ee.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.ee.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.ee.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.ee.registrikood
    input = capture.registration
  }
}

checksum "euid" "ee" {
  rule = apply_checksum(checksum.ee.registrikood, after_first(subject(), "."))
}

identifier "euid" "EE" {
  canonicalizer   = canonicalizer.euid.ee
  format          = format.euid.ee
  checksum        = checksum.euid.ee
  default_profile = "compatible"

  source {
    id               = "ee-ariregister-registrikood"
    url              = "https://ariregister.rik.ee"
    authority        = "Registrite ja Infosusteemide Keskus (RIK)"
    title            = "e-Business Register"
    accessed_at      = "2026-08-20"
    jurisdiction     = "EE"
    language         = "et"
    notes            = "The register publishes the eight digit registrikood. It does not publish the algorithm closing it."
    license_or_terms = "Estonian public sector information"
    tier             = "primary"
  }

  source {
    id               = "ee-registrikood-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "EE"
    language         = "en"
    notes            = "Weights 1 to 7 over the first seven digits, remainder modulo 11. When that remainder is ten the sum is recomputed with weights 3 to 9 and the check digit is the new remainder modulo ten."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
