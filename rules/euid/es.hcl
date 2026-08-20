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
#
# Note on scope. The VAT rule of this country accepts the identifier of a sole
# trader, because such a person invoices under it and refusing it would refuse a
# real business identifier. The register rule stays restricted to the CIF: a sole trader is not entered in the register of companies that feeds the EUID, so no EUID is issued against a DNI. If a Member State does register sole traders under this scheme, that restriction becomes wrong and should be revisited.

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
  ]

  # The registration part is the national number, so the national rule is
  # applied to it rather than restated here.
  use_format {
    rule  = format.es.nif
    input = capture.registration
  }
}

checksum "euid" "es" {
  rule = apply_checksum(checksum.es.nif, after_first(subject(), "."))
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
