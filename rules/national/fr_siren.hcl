# France - SIREN
#
# The SIREN is the nine digit identifier assigned by INSEE to every legal unit
# registered in the Sirene system. It carries a Luhn check digit.

canonicalizer "fr" "siren" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "fr" "siren" {
  checks = [
    require(not(is_empty(subject())), "empty", "fr.siren.empty"),
    require(length_eq(subject(), 9), "invalid_length", "fr.siren.length"),
    require(ascii_digits(subject()), "invalid_characters", "fr.siren.characters"),
  ]
}

checksum "fr" "siren" {
  rule = luhn(subject())
}

dispatcher "siren" {
  aliases           = ["fr_siren"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "FR"
    identifier                       = identifier.siren.FR
    allow_unprefixed_without_country = true
  }
}

identifier "siren" "FR" {
  canonicalizer   = canonicalizer.fr.siren
  format          = format.fr.siren
  checksum        = checksum.fr.siren
  default_profile = "compatible"

  source {
    id               = "fr-insee-siren"
    url              = "https://www.insee.fr/fr/information/2015441"
    authority        = "Institut national de la statistique et des etudes economiques (INSEE)"
    title            = "Le repertoire Sirene - numeros SIREN et SIRET"
    accessed_at      = "2026-08-18"
    jurisdiction     = "FR"
    language         = "fr"
    notes            = "Nine digits assigned by INSEE, closed by a Luhn check digit computed over the nine digits."
    license_or_terms = "Licence Ouverte / Open Licence (Etalab), public sector information"
  }
}
