# Spain - VAT identification number (NIF)
#
# The NIF covers three shapes, and all three identify someone carrying on an
# economic activity:
#
#   - the CIF, a leading letter then seven digits and a check character, for a
#     legal person;
#   - the DNI, eight digits and a letter, which a self employed worker uses as
#     their tax number - the tax agency states that for Spanish nationals the
#     tax identification number is the number of their identity document;
#   - the NIE, X, Y or Z then seven digits and a letter, the same for a foreign
#     national.
#
# All three are accepted. A sole trader invoices under their DNI, so refusing it
# would refuse a real business identifier - the false negative section 1.2 calls
# the most serious defect of the project. Nothing in the shape distinguishes the
# number of a trader from that of a private individual; the caller knows the
# context, the format does not.
#
# The check is a Luhn wherever it is a digit. Where it is a letter it encodes a
# value through an alphabet - JABCDEFGHI for a CIF, TRWAGMYFPDXBNJZSQVHLCKE for
# a DNI - and turning a letter back into a number needs a mapping the IR does
# not have, so those report unsupported.

canonicalizer "vat" "es" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "es" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.es.empty"),
    require(length_eq(subject(), 11), "invalid_length", "vat.es.length"),
    require(starts_with(subject(), "ES"), "invalid_format", "vat.es.prefix"),
    require(ascii_alphanumeric(slice_from(subject(), 2)), "invalid_characters", "vat.es.characters"),
    require(
      any(
        // A legal person: letter, seven digits, check character.
        all(
          char_at_in(subject(), 2, "ABCDEFGHJNPQRSUVW"),
          ascii_digits(slice(subject(), 3, 10)),
        ),
        // A resident sole trader: eight digits then a letter.
        all(
          ascii_digits(slice(subject(), 2, 10)),
          char_at_in(subject(), 10, "TRWAGMYFPDXBNJZSQVHLCKE"),
        ),
        // A foreign sole trader: X, Y or Z then seven digits and a letter.
        all(
          char_at_in(subject(), 2, "XYZ"),
          ascii_digits(slice(subject(), 3, 10)),
          char_at_in(subject(), 10, "TRWAGMYFPDXBNJZSQVHLCKE"),
        ),
      ),
      "invalid_format",
      "vat.es.shape",
    ),
  ]
}

checksum "vat" "es" {
  rule = choose(
    when_checksum(
      all(
        char_at_in(subject(), 2, "ABCDEFGHJNPQRSUVW"),
        char_at_in(subject(), 10, "0123456789"),
      ),
      luhn(slice(subject(), 3, 11)),
    ),
    unsupported_checksum("checksum_not_published"),
  )
}

identifier "vat" "ES" {
  canonicalizer   = canonicalizer.vat.es
  format          = format.vat.es
  checksum        = checksum.vat.es
  default_profile = "compatible"

  source {
    id               = "eu-vies-number-structure"
    url              = "https://ec.europa.eu/taxation_customs/vies/"
    authority        = "European Commission, Directorate-General for Taxation and Customs Union"
    title            = "VIES - VAT number structure per Member State"
    accessed_at      = "2026-08-20"
    jurisdiction     = "EU"
    language         = "en"
    notes            = "Published length and prefix of each Member State VAT identification number."
    license_or_terms = "European Commission reuse policy, Decision 2011/833/EU"
    tier             = "primary"
  }

  source {
    id               = "es-vat-check"
    url              = "https://sede.agenciatributaria.gob.es/Sede/censos-nif-domicilio-fiscal/solicitar-nif.html"
    authority        = "Agencia Estatal de Administracion Tributaria (AEAT)"
    title            = "Como solicitar un NIF"
    accessed_at      = "2026-08-20"
    jurisdiction     = "ES"
    language         = "es"
    notes            = "Anyone carrying on a business or professional activity must hold a NIF. For a Spanish national that number is the one on their identity document, so a sole trader trades under a DNI."
    license_or_terms = "Spanish public sector information"
    tier             = "primary"
  }
}
