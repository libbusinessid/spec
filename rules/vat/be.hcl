# Belgium - VAT identification number (BTW-nr / No TVA)
#
# The Belgian VAT number is the enterprise number of the Crossroads Bank for
# Enterprises prefixed by BE. It holds ten digits, starts with 0 or 1 and is
# closed by a modulo 97 check on its first eight digits.

canonicalizer "vat" "be" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
    prepend_country_if_missing(),
    when(
      all(length_eq(value(), 11), ascii_digits(slice_from(value(), 2))),
      insert(2, "0"),
    ),
  ]
}

format "vat" "be" {
  checks = [
    require(not(is_empty(subject())), "empty", "vat.be.empty"),
    require(length_eq(subject(), 12), "invalid_length", "vat.be.length"),
    require(starts_with(subject(), "BE"), "invalid_format", "vat.be.prefix"),
    require(ascii_digits(slice_from(subject(), 2)), "invalid_characters", "vat.be.characters"),
    require(char_at_in(subject(), 2, "01"), "invalid_format", "vat.be.enterprise_prefix"),
  ]
}

checksum "vat" "be" {
  rule = compare_slice(
    complement(mod_digits(slice(subject(), 2, 10), 97), 97),
    subject(), 10, 12,
  )
}

identifier "vat" "BE" {
  canonicalizer   = canonicalizer.vat.be
  format          = format.vat.be
  checksum        = checksum.vat.be
  default_profile = "compatible"

  source {
    id               = "be-fps-finance-vat"
    url              = "https://finances.belgium.be/fr/entreprises/tva/assujettissement-a-la-tva/numero-de-tva"
    authority        = "Service public federal Finances (SPF Finances)"
    title            = "Numero de TVA belge"
    accessed_at      = "2026-08-18"
    jurisdiction     = "BE"
    language         = "fr"
    notes            = "BE followed by the ten digit enterprise number. Numbers issued before 2008 held nine digits and are completed by a leading zero. The last two digits are 97 minus the first eight digits modulo 97."
    license_or_terms = "Public sector information published by the Belgian federal administration"
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
  }
}
