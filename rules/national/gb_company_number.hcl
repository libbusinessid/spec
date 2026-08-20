# United Kingdom - Companies House company number
#
# Eight characters, always. An England and Wales company is eight digits; every
# other register places a two character prefix in front of six characters. The
# prefix is what carries the meaning, and the six characters that follow are not
# always digits: a registered society ends in one to four letters, and a handful
# of societies carry letters before the digits as well.
#
# The register publishes no check algorithm over the number, so no checksum is
# applied.

canonicalizer "gb" "company_number" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars(["-", "."]),
  ]
}

format "gb" "company_number" {
  checks = [
    require(not(is_empty(subject())), "empty", "company_number.empty"),
    require(length_eq(subject(), 8), "invalid_length", "company_number.length"),
    require(ascii_alphanumeric(subject()), "invalid_characters", "company_number.characters"),

    # Eight digits, or one of the prefixes the registrar issues. A number whose
    # first two characters are digits without being all digits belongs to no
    # register, so it is refused here rather than by a rule per prefix.
    require(
      any(
        ascii_digits(subject()),
        prefix_in(subject(), [
          # Carried by at least one company on the live register.
          "AC", "CE", "CS", "FC", "FE", "GE", "GS", "IC",
          "IP", "LP", "NC", "NF", "NI", "NL", "NO", "NP",
          "OC", "OE", "PC", "R0", "RC", "RS", "SA", "SC",
          "SE", "SF", "SG", "SI", "SL", "SO", "SP", "SR",
          "SZ", "ZC",
          # Documented by the registrar, carried by no live company.
          "EN", "ES", "GN", "NA", "NR", "NV", "NZ",
        ]),
      ),
      "invalid_format",
      "company_number.prefix",
    ),
  ]
}

dispatcher "company_number" {
  aliases           = ["gb_company_number", "uk_company_number", "crn", "company_registration_number"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "GB"
    identifier                       = identifier.company_number.GB
    allow_unprefixed_without_country = true
  }
}

identifier "company_number" "GB" {
  canonicalizer   = canonicalizer.gb.company_number
  format          = format.gb.company_number
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "Companies House issues the number sequentially within each register and publishes no check character or algorithm over it."
  }

  source {
    id               = "gb-companies-house-data"
    url              = "https://download.companieshouse.gov.uk/en_output.html"
    authority        = "Companies House"
    title            = "Free Company Data Product"
    accessed_at      = "2026-08-20"
    jurisdiction     = "GB"
    language         = "en"
    notes            = "Monthly snapshot of every company on the live register. The 2026-08-01 file carries 5695465 companies, every one of them numbered on exactly eight characters, across the thirty four prefixes accepted here and the ten character shapes the rule admits."
    license_or_terms = "Crown copyright, Open Government Licence v3.0"
    tier             = "primary"
  }

  source {
    id               = "gb-companies-house-uri-guide"
    url              = "https://assets.publishing.service.gov.uk/government/uploads/system/uploads/attachment_data/file/426891/uniformResourceIdentifiersCustomerGuide.pdf"
    authority        = "Companies House"
    title            = "Uniform Resource Identifiers (URI) Customer Guide"
    accessed_at      = "2026-08-20"
    jurisdiction     = "GB"
    language         = "en"
    notes            = "Publishes the prefix tables and the rule that a prefix is written in upper case. Seven of the prefixes it documents, EN ES GN NA NR NV and NZ, carry no company on the live register and are accepted on the strength of this table alone."
    license_or_terms = "Crown copyright, Open Government Licence v3.0"
    tier             = "primary"
  }

  source {
    id               = "gb-hmrc-number-formats"
    url              = "https://www.hmrc.gov.uk/gds/com/attachments/coy_reg_no_formats.doc"
    authority        = "HM Revenue and Customs"
    title            = "Company Registration Number Formats"
    accessed_at      = "2026-08-20"
    jurisdiction     = "GB"
    language         = "en"
    notes            = "States that English and Welsh companies carry eight digits and that the other registers carry a one or two character prefix. It lists R, one character, where Companies House lists R0; the live register settles it, since every one of the ninety three numbers beginning with R has zero in second position."
    license_or_terms = "Crown copyright, Open Government Licence v3.0"
    tier             = "primary"
  }
}
