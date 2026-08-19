# France - SIRET
#
# The SIRET identifies an establishment: the nine digits of the SIREN of the
# legal unit, followed by a five digit NIC. The Luhn check runs over the whole
# fourteen digits.
#
# La Poste is a documented derogation. Its establishments were numbered before
# the Luhn constraint applied to them, and INSEE accepts a La Poste SIRET whose
# digits sum to a multiple of five. That alternative cannot be expressed in the
# V1 IR: a checksum can only compare a computed integer against a digit or a
# slice of the value, never against a literal constant. The rule therefore
# reports `valid` when the Luhn holds and `unsupported` when it does not, rather
# than `invalid`. Refusing a genuine La Poste SIRET would be a false negative,
# which the tri-state exists to prevent.

canonicalizer "fr" "siret" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "fr" "siret" {
  checks = [
    require(not(is_empty(subject())), "empty", "fr.siret.empty"),
    require(length_eq(subject(), 14), "invalid_length", "fr.siret.length"),
    require(ascii_digits(subject()), "invalid_characters", "fr.siret.characters"),
  ]
}

checksum "fr" "siret" {
  rule = choose(
    when_checksum(
      starts_with(subject(), "356000000"),
      any_check(
        luhn(subject()),
        unsupported_checksum("checksum_not_published"),
      ),
    ),
    luhn(subject()),
  )
}

dispatcher "siret" {
  aliases           = ["fr_siret"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "FR"
    identifier                       = identifier.siret.FR
    allow_unprefixed_without_country = true
  }
}

identifier "siret" "FR" {
  canonicalizer   = canonicalizer.fr.siret
  format          = format.fr.siret
  checksum        = checksum.fr.siret
  default_profile = "compatible"

  source {
    id               = "fr-insee-siren"
    url              = "https://www.insee.fr/fr/information/2015441"
    authority        = "Institut national de la statistique et des etudes economiques (INSEE)"
    title            = "Le repertoire Sirene - numeros SIREN et SIRET"
    accessed_at      = "2026-08-18"
    jurisdiction     = "FR"
    language         = "fr"
    notes            = "Fourteen digits: the nine digit SIREN of the legal unit followed by a five digit NIC, closed by a Luhn check over the fourteen digits. Establishments of La Poste follow a documented derogation."
    license_or_terms = "Licence Ouverte / Open Licence (Etalab), public sector information"
  }
}
