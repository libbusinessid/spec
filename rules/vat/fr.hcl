# France - VAT identification number (numero de TVA intracommunautaire)
#
# The French VAT number is FR, a two character key and the nine digit SIREN.
# When the key is numeric it is (12 + 3 * (SIREN mod 97)) mod 97. Keys holding
# a letter follow an unpublished scheme and are therefore never declared
# invalid: the checksum step reports checksum_not_published. The embedded SIREN
# always carries its own Luhn check digit, so both checks are combined: a proven
# SIREN failure wins over an unknown key scheme.

canonicalizer "vat" "fr" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
  ]
}

format "vat" "fr" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.fr.empty"),
    require(length_eq(subject(), 13), "invalid_length", "vat.fr.length"),
    require(starts_with(subject(), "FR"), "invalid_format", "vat.fr.prefix"),
    require(
      any(
        ascii_digits(slice(subject(), 2, 4)),
        all(
          profile_is("compatible"),
          ascii_alphanumeric(slice(subject(), 2, 4)),
        ),
      ),
      "invalid_characters",
      "vat.fr.key_characters",
    ),
    require(ascii_digits(slice_from(subject(), 4)), "invalid_characters", "vat.fr.siren_characters"),
  ]

  use_format {
    rule  = format.fr.siren
    input = slice_from(subject(), 4)
  }
}

checksum "vat" "fr" {
  rule = all_checks(
    apply_checksum(checksum.fr.siren, slice_from(subject(), 4)),
    choose(
      when_checksum(
        ascii_digits(slice(subject(), 2, 4)),
        compare_slice(
          remainder_map(
            mod_digits(slice_from(subject(), 4), 97),
            [
              12, 15, 18, 21, 24, 27, 30, 33, 36, 39, 42, 45, 48, 51, 54, 57, 60, 63,
              66, 69, 72, 75, 78, 81, 84, 87, 90, 93, 96, 2, 5, 8, 11, 14, 17, 20, 23,
              26, 29, 32, 35, 38, 41, 44, 47, 50, 53, 56, 59, 62, 65, 68, 71, 74, 77,
              80, 83, 86, 89, 92, 95, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37,
              40, 43, 46, 49, 52, 55, 58, 61, 64, 67, 70, 73, 76, 79, 82, 85, 88, 91,
              94, 0, 3, 6, 9
            ],
          ),
          subject(), 2, 4,
        ),
      ),
      unsupported_checksum("checksum_not_published"),
    ),
  )
}

identifier "vat" "FR" {
  canonicalizer   = canonicalizer.vat.fr
  format          = format.vat.fr
  checksum        = checksum.vat.fr
  default_profile = "compatible"

  source {
    id               = "fr-dgfip-vat"
    url              = "https://www.impots.gouv.fr/professionnel/le-numero-de-tva-intracommunautaire"
    authority        = "Direction generale des Finances publiques (DGFiP)"
    title            = "Le numero de TVA intracommunautaire"
    accessed_at      = "2026-08-18"
    jurisdiction     = "FR"
    language         = "fr"
    notes            = "FR, a two character computation key and the nine digit SIREN. The numeric key is (12 + 3 * (SIREN mod 97)) mod 97. Alphanumeric keys exist and their derivation is not published. The embedded SIREN keeps its own Luhn check digit."
    license_or_terms = "Licence Ouverte / Open Licence (Etalab), public sector information"
    tier             = "primary"
  }

  source {
    id               = "eu-vies-number-structure"
    url              = "https://ec.europa.eu/taxation_customs/vies/"
    authority        = "European Commission, Directorate-General for Taxation and Customs Union"
    title            = "VIES - VAT number structure per Member State"
    accessed_at      = "2026-08-18"
    jurisdiction     = "EU"
    language         = "en"
    notes            = "Cross-check of the published length and prefix of each Member State VAT identification number."
    license_or_terms = "European Commission reuse policy, Decision 2011/833/EU"
    tier             = "primary"
  }
}
