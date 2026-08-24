# Luxembourg - RCS number
#
# The registration part of the Luxembourg EUID is this number, so the EUID rule
# applies this one to the part after the dot instead of restating it. The
# algorithm is stated here, once.

canonicalizer "lu" "rcs_number" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/"]),
  ]
}

format "lu" "rcs_number" {
  checks = [
    require(not(is_empty(subject())), "empty", "lu.rcs_number.empty"),
    require(length_between(subject(), 5, 7), "invalid_length", "lu.rcs_number.length"),
    # The letter marks the section of the register, and the register enrols
    # twelve categories of persons: the law of 19 December 2002 lists them, from
    # commercants personnes physiques to fondations and associations agricoles.
    # This rule accepted B alone, so a sole trader's number was refused - a real
    # entity turned away, which is the worst answer this library can give.
    #
    # No published source states which letters are in use, and the law does not
    # define the number's shape at all. So the letter is constrained to being a
    # letter rather than to a list nobody publishes: a weaker claim needs no
    # source, while the stronger one this replaces never had one.
    require(char_at_in(subject(), 0, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"), "invalid_format", "lu.rcs_number.prefix"),
    require(ascii_digits(slice_from(subject(), 1)), "invalid_characters", "lu.rcs_number.characters"),
  ]
}

dispatcher "rcs_number" {
  aliases           = ["lu_rcs_number"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "LU"
    identifier                       = identifier.rcs_number.LU
    allow_unprefixed_without_country = true
  }
}

identifier "rcs_number" "LU" {
  canonicalizer   = canonicalizer.lu.rcs_number
  format          = format.lu.rcs_number
  default_profile = "compatible"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "The register publishes the shape of the number and no check algorithm, so no checksum is applied rather than one being guessed."
  }

  source {
    id               = "lu-rcsl-number"
    url              = "https://www.lbr.lu"
    authority        = "Luxembourg Business Registers"
    title            = "Registre de commerce et des societes"
    accessed_at      = "2026-08-20"
    jurisdiction     = "LU"
    language         = "fr"
    notes            = "The register publishes a number formed of a section letter followed by digits. Which letters are in use is not published, and the law of 19 December 2002 defines the categories enrolled without defining the number, so the letter is not constrained to a list."
    license_or_terms = "Luxembourg public sector information"
    tier             = "primary"
  }
}
