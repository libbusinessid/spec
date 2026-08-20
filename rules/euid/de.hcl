# Germany - EUID
#
# The registration part is the Handelsregister entry, which the courts publish
# as a register kind and a number - HRB 12345 and the like - with the local
# court carried by the register identifier of the EUID rather than by the
# number.
#
# There is no single national shape to assert: the kind, the number and their
# separators vary by Land, and the courts publish no algorithm over them. The
# rule therefore checks what is common to every entry - a non empty
# alphanumeric registration within the length the register uses - and states
# that it verifies nothing further, rather than asserting a national format
# that does not exist. The historical Go library accepted any value holding one
# alphanumeric character, which is weaker still and says so nowhere.

canonicalizer "euid" "de" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "de" {
  capture "register" {
    value = before_first(after_first(subject(), "DE"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.de.empty"),
    require(starts_with(subject(), "DE"), "invalid_format", "euid.de.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.de.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.de.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.de.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.de.register_characters"),
    require(not(is_empty(capture.registration)), "empty", "euid.de.registration_empty"),
    require(length_between(capture.registration, 1, 20), "invalid_length", "euid.de.registration_length"),
    require(ascii_alphanumeric(capture.registration), "invalid_characters", "euid.de.registration_characters"),
  ]
}

identifier "euid" "DE" {
  canonicalizer   = canonicalizer.euid.de
  format          = format.euid.de
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "A Handelsregister entry carries no check character, and the courts publish no algorithm over the number."
  }

  source {
    id               = "de-handelsregister"
    url              = "https://www.handelsregister.de"
    authority        = "Justizministerien der Laender"
    title            = "Gemeinsames Registerportal der Laender"
    accessed_at      = "2026-08-20"
    jurisdiction     = "DE"
    language         = "de"
    notes            = "The portal publishes register entries as a register kind and a number held by a local court. It states no single national shape for the number and no algorithm over it."
    license_or_terms = "German public sector information"
    tier             = "primary"
  }
}
