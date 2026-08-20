# China - Unified Social Credit Code
#
# Eighteen characters identifying a legal person or other organisation, issued
# under GB 32100-2015. The first is the registering authority, the second the
# entity category, the next six the administrative division, the next nine the
# organisation code, and the last a check character.
#
# The alphabet holds thirty one code points: the ten digits and the upper case
# letters except I, O, S, V and Z, which are dropped because they are read as
# digits. That shift is why the check cannot be expressed over base 36: the
# alphabet makes J worth 18 where base 36 makes it 19.
#
# The check character is the complement modulo 31 of a weighted sum over the
# first seventeen positions, weighted by the powers of three modulo 31.

canonicalizer "cn" "uscc" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars(["-", "."]),
  ]
}

format "cn" "uscc" {
  checks = [
    require(not(is_empty(subject())), "empty", "uscc.empty"),
    require(length_eq(subject(), 18), "invalid_length", "uscc.length"),
    require(ascii_charset(subject(), "0123456789ABCDEFGHJKLMNPQRTUWXY"), "invalid_characters", "uscc.characters"),
  ]
}

checksum "cn" "uscc" {
  # The standard states the check character as the complement modulo 31 of the
  # sum over the first seventeen positions. Stated that way it would have to be
  # compared against a character, and compare_slice reads a number: the check
  # character is often a letter.
  #
  # The same statement rearranged needs no such comparison. The complement is
  # what makes the total vanish, so summing all eighteen positions with the
  # check character weighted 1 must leave a multiple of 31. Verified against
  # every code in the corpus, and against each single character mutation of
  # them.
  rule = compare_constant(
    modulo(
      weighted_sum(
        subject(),
        [1, 3, 9, 27, 19, 26, 16, 17, 20, 29, 25, 13, 8, 24, 10, 30, 28, 1],
        "left",
        "custom_alphabet",
        "0123456789ABCDEFGHJKLMNPQRTUWXY",
      ),
      31,
    ),
    0,
  )
}

dispatcher "uscc" {
  aliases           = ["cn_uscc", "unified_social_credit_code", "usci"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "CN"
    identifier                       = identifier.uscc.CN
    allow_unprefixed_without_country = true
  }
}

identifier "uscc" "CN" {
  canonicalizer   = canonicalizer.cn.uscc
  format          = format.cn.uscc
  checksum        = checksum.cn.uscc
  default_profile = "compatible"

  source {
    id               = "cn-gb-32100-2015"
    url              = "https://openstd.samr.gov.cn/bzgk/gb/newGbInfo?hcno=24691C25985C1073D3A7C85629378AC0"
    authority        = "Standardization Administration of China"
    title            = "GB 32100-2015 Coding rules for the unified social credit identifier of legal entities and other organizations"
    accessed_at      = "2026-08-20"
    jurisdiction     = "CN"
    language         = "zh"
    notes            = "Mandatory national standard published 2015-09-17 and in force since 2015-10-01. It fixes the eighteen character structure, the thirty one code point alphabet that omits I, O, S, V and Z, and the check character as the complement modulo 31 of a sum weighted by the powers of three modulo 31."
    license_or_terms = "Published by the issuing authority for public reference"
    tier             = "primary"
  }
}
