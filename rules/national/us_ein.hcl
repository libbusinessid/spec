# United States - Employer Identification Number
#
# Nine digits issued by the Internal Revenue Service, printed as XX-XXXXXXX. The
# first two digits are a campus prefix, and the service publishes no check
# algorithm over the number.

canonicalizer "us" "ein" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    remove_chars(["-"]),
  ]
}

format "us" "ein" {
  checks = [
    require(not(is_empty(subject())), "empty", "ein.empty"),
    require(length_eq(subject(), 9), "invalid_length", "ein.length"),
    require(ascii_digits(subject()), "invalid_characters", "ein.characters"),
  ]
}

dispatcher "ein" {
  aliases           = ["us_ein", "federal_tax_id"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "US"
    identifier                       = identifier.ein.US
    allow_unprefixed_without_country = true
  }
}

identifier "ein" "US" {
  canonicalizer   = canonicalizer.us.ein
  format          = format.us.ein
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The Internal Revenue Service publishes the nine digit number and its campus prefixes, and no algorithm closing it."
  }

  source {
    id               = "us-irs-ein"
    url              = "https://www.irs.gov/businesses/small-businesses-self-employed/employer-id-numbers"
    authority        = "Internal Revenue Service (IRS)"
    title            = "Employer ID Numbers"
    accessed_at      = "2026-08-20"
    jurisdiction     = "US"
    language         = "en"
    notes            = "Nine digits printed as XX-XXXXXXX, the first two being the campus prefix of the issuing office."
    license_or_terms = "United States federal government work, public domain"
    tier             = "primary"
  }
}
