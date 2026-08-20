# Japan - Corporate Number
#
# Thirteen digits assigned by the National Tax Agency: a twelve digit base
# number, which for a company incorporated under the Companies Act is the
# registration number the Ministry of Justice issues, preceded by one check
# digit.
#
# The agency states the check digit as nine minus the remainder modulo nine of
# twice the sum of the even positions of the base number plus the sum of its odd
# positions, counting from the right.

canonicalizer "jp" "corporate_number" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    remove_chars(["-", "."]),
  ]
}

format "jp" "corporate_number" {
  checks = [
    require(not(is_empty(subject())), "empty", "corporate_number.empty"),
    require(length_eq(subject(), 13), "invalid_length", "corporate_number.length"),
    require(ascii_digits(subject()), "invalid_characters", "corporate_number.characters"),
  ]
}

checksum "jp" "corporate_number" {
  # Stated as the agency states it, the check digit would have to be compared
  # against a computed complement, and the complement is nine when the remainder
  # is zero rather than the zero a plain complement would give.
  #
  # The same statement rearranged avoids both. Whatever the remainder, adding
  # the check digit to the weighted sum lands on a multiple of nine: nine minus
  # a remainder r cancels r, and the case r equal to zero adds nine, which is
  # itself a multiple. So the whole thirteen digits, with the even positions
  # from the right weighted 2 and the odd ones weighted 1, must leave a multiple
  # of nine. The check digit sits at position 13, an odd position, so it takes
  # weight 1 and needs no special treatment.
  rule = compare_constant(
    modulo(
      weighted_sum(
        subject(),
        [1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1],
        "left",
        "digit_value",
      ),
      9,
    ),
    0,
  )
}

dispatcher "corporate_number" {
  aliases           = ["jp_corporate_number", "houjin_bangou", "hojin_bango"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "JP"
    identifier                       = identifier.corporate_number.JP
    allow_unprefixed_without_country = true
  }
}

identifier "corporate_number" "JP" {
  canonicalizer   = canonicalizer.jp.corporate_number
  format          = format.jp.corporate_number
  checksum        = checksum.jp.corporate_number
  default_profile = "compatible"

  source {
    id               = "jp-nta-check-digit"
    url              = "https://www.houjin-bangou.nta.go.jp/documents/checkdigit.pdf"
    authority        = "National Tax Agency"
    title            = "Calculation of the check digit"
    accessed_at      = "2026-08-20"
    jurisdiction     = "JP"
    language         = "ja"
    notes            = "States the thirteen digit structure, that the base number of a company incorporated under the Companies Act is the twelve digit registration number of the Ministry of Justice, and the check digit as nine minus the remainder modulo nine of twice the sum of the even positions plus the sum of the odd ones, counting from the right. Its worked example takes the base 700110005901 to the corporate number 8700110005901."
    license_or_terms = "Published by the issuing authority for public reference"
    tier             = "primary"
  }
}
