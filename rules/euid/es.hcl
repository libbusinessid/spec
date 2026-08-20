# Spain - EUID
#
# The registration part is the CIF, which designates a legal person: a leading
# letter, seven digits, and a check character.
#
# The Spanish identifier also covers the DNI and the NIE, which identify natural
# persons. Neither is accepted here, for the reason Latvia is restricted the same
# way: an EUID designates an entity of a business register, and validating the
# identifiers of people is not what this corpus is for.
#
# The check itself is a Luhn over the seven digits and the check character. That
# is not a coincidence: doubling the odd positions of the body, as the Spanish
# rule states it, is doubling the even positions counted from the right, which is
# what Luhn does.
#
# Where the check character is a letter, it encodes the same digit through the
# alphabet JABCDEFGHI. Turning that letter back into a digit needs a mapping the
# IR does not have, so those numbers report `unsupported` rather than a verdict.
# The leading letters that require the letter form - P, Q, R, S and W - therefore
# always land there, and the others are checked.

canonicalizer "euid" "es" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    prepend_country_if_missing(),
  ]
}

format "euid" "es" {
  capture "register" {
    value = before_first(after_first(subject(), "ES"), ".")
  }

  capture "registration" {
    value = after_first(subject(), ".")
  }

  checks = [
    require(not(is_empty(subject())), "empty", "euid.es.empty"),
    require(starts_with(subject(), "ES"), "invalid_format", "euid.es.prefix"),
    require(contains(subject(), "."), "invalid_format", "euid.es.separator"),
    require(not(is_absent(capture.register)), "invalid_format", "euid.es.register"),
    require(length_between(capture.register, 1, 8), "invalid_length", "euid.es.register_length"),
    require(ascii_alphanumeric(capture.register), "invalid_characters", "euid.es.register_characters"),
    require(length_eq(capture.registration, 9), "invalid_length", "euid.es.registration_length"),
    require(ascii_alphanumeric(capture.registration), "invalid_characters", "euid.es.registration_characters"),
    require(
      char_at_in(capture.registration, 0, "ABCDEFGHJNPQRSUVW"),
      "invalid_format",
      "euid.es.registration_legal_entity",
    ),
    require(ascii_digits(slice(capture.registration, 1, 8)), "invalid_characters", "euid.es.registration_body"),
  ]
}

checksum "euid" "es" {
  rule = choose(
    when_checksum(
      char_at_in(after_first(subject(), "."), 8, "0123456789"),
      luhn(slice(after_first(subject(), "."), 1, 9)),
    ),
    unsupported_checksum("checksum_not_published"),
  )
}

identifier "euid" "ES" {
  canonicalizer   = canonicalizer.euid.es
  format          = format.euid.es
  checksum        = checksum.euid.es
  default_profile = "compatible"

  source {
    id               = "es-aeat-nif"
    url              = "https://sede.agenciatributaria.gob.es"
    authority        = "Agencia Estatal de Administracion Tributaria (AEAT)"
    title            = "Numero de identificacion fiscal"
    accessed_at      = "2026-08-20"
    jurisdiction     = "ES"
    language         = "es"
    notes            = "The tax agency publishes the NIF, of which the CIF form designates a legal person: a leading letter, seven digits and a check character."
    license_or_terms = "Spanish public sector information"
    tier             = "primary"
  }

  source {
    id               = "es-cif-check"
    url              = "https://en.wikipedia.org/wiki/VAT_identification_number"
    authority        = "Wikipedia"
    title            = "VAT identification number - national check algorithms"
    accessed_at      = "2026-08-20"
    jurisdiction     = "ES"
    language         = "en"
    notes            = "The check doubles the odd positions of the body and sums the digits of each result, which is the Luhn computation. A letter check encodes the same digit through the alphabet JABCDEFGHI."
    license_or_terms = "CC BY-SA 4.0, cited as a description and not redistributed"
    tier             = "secondary"
  }
}
