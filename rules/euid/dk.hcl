# # Denmark - EUID\n#\n# The registration part is the eight digit CVR number. Unlike the other\n# registers of this batch it carries no separate check digit: the weighted sum\n# of all eight digits must itself be divisible by eleven.

canonicalizer "euid" "dk" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "dk" {
  capture "register" {
    value = before_first(after_first(subject(), "DK"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.dk.empty"),
    require(starts_with(subject(), "DK"), "invalid_format", "euid.dk.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.dk.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.dk.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.dk.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.dk.register_characters"),
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.dk.cvr
    input = capture.registration
  }
}

checksum "euid" "dk" {
  rule = apply_checksum(checksum.dk.cvr, after_first(subject(), "."))
}

identifier "euid" "DK" {
  canonicalizer   = canonicalizer.euid.dk
  format          = format.euid.dk
  checksum        = checksum.euid.dk
  default_profile = "compatible"

  source {
    id               = "dk-erhvervsstyrelsen-cvr"
    url              = "https://erhvervsstyrelsen.dk/cvr-numre-p-numre-og-se-numre"
    authority        = "Erhvervsstyrelsen (Danish Business Authority)"
    title            = "CVR-numre, p-numre og se-numre"
    accessed_at      = "2026-08-20"
    jurisdiction     = "DK"
    language         = "da"
    notes            = "The authority publishes the eight digit CVR number as the key of the central business register. It does not publish the algorithm validating it."
    license_or_terms = "Danish public sector information"
    tier             = "primary"
  }

  source {
    id               = "dk-cvr-mod11"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "DK"
    language         = "en"
    notes            = "Weights 2, 7, 6, 5, 4, 3, 2, 1 over the eight digits, the whole sum being divisible by eleven. The first digit is never zero."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
