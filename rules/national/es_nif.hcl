# Spain - NIF
#
# The registration part of the Spain EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "es" "nif" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "es" "nif" {
  checks = [
    require(not(is_empty(subject())), "empty", "es.nif.empty"),
    require(length_eq(subject(), 9), "invalid_length", "es.nif.length"),
    require(ascii_alphanumeric(subject()), "invalid_characters", "es.nif.characters"),
    require(
      char_at_in(subject(), 0, "ABCDEFGHJNPQRSUVW"),
      "invalid_format",
      "es.nif.legal_entity",
    ),
    require(ascii_digits(slice(subject(), 1, 8)), "invalid_characters", "es.nif.body"),
  ]
}

checksum "es" "nif" {
  rule = choose(
    when_checksum(
      char_at_in(subject(), 8, "0123456789"),
      luhn(slice(subject(), 1, 9)),
    ),
    unsupported_checksum("checksum_not_published"),
  )
}

dispatcher "nif" {
  aliases           = ["es_nif"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "ES"
    identifier                       = identifier.nif.ES
    allow_unprefixed_without_country = true
  }
}

identifier "nif" "ES" {
  canonicalizer   = canonicalizer.es.nif
  format          = format.es.nif
  checksum        = checksum.es.nif
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
