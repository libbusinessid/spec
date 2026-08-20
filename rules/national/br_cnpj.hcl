# Brazil - CNPJ
#
# Fourteen positions: an eight position root identifying the company, four
# identifying the establishment, and two numeric check digits.
#
# From 2026 the first twelve positions are alphanumeric. Existing CNPJ keep
# their value and their check digits unchanged, so a rule has to accept both
# forms: one written for digits alone would refuse every establishment
# registered from now on, and the first alphanumeric CNPJ has already been
# issued.
#
# The check digits are two rounds of modulus 11 over weights descending from 9
# to 2, and a remainder below 2 gives a check digit of 0. The value of a
# position is its ASCII code minus 48, so a digit keeps its own value and A is
# 17 - neither base 36, where A is 10, nor anything the digit mapping can read.
# The alphabet below is exactly that offset written out.

canonicalizer "br" "cnpj" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "/", "-"]),
  ]
}

format "br" "cnpj" {
  checks = [
    require(not(is_empty(subject())), "empty", "cnpj.empty"),
    require(length_eq(subject(), 14), "invalid_length", "cnpj.length"),
    require(ascii_alphanumeric(subject()), "invalid_characters", "cnpj.characters"),
    require(ascii_digits(slice(subject(), 12, 14)), "invalid_characters", "cnpj.check_digits"),
  ]
}

checksum "br" "cnpj" {
  rule = all_checks(
    compare_digit(
      remainder_map(
        modulo(
          weighted_sum(
            slice(subject(), 0, 12),
            [5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2],
            "left",
            "custom_alphabet",
            "0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ",
          ),
          11,
        ),
        [0, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1],
      ),
      subject(),
      12,
    ),
    compare_digit(
      remainder_map(
        modulo(
          weighted_sum(
            slice(subject(), 0, 13),
            [6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2],
            "left",
            "custom_alphabet",
            "0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ",
          ),
          11,
        ),
        [0, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1],
      ),
      subject(),
      13,
    ),
  )
}

dispatcher "cnpj" {
  aliases           = ["br_cnpj"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code                     = "BR"
    identifier                       = identifier.cnpj.BR
    allow_unprefixed_without_country = true
  }
}

identifier "cnpj" "BR" {
  canonicalizer   = canonicalizer.br.cnpj
  format          = format.br.cnpj
  checksum        = checksum.br.cnpj
  default_profile = "compatible"

  source {
    id               = "br-rfb-cnpj-alfanumerico"
    url              = "https://www.gov.br/receitafederal/pt-br/acesso-a-informacao/acoes-e-programas/programas-e-atividades/cnpj-alfanumerico"
    authority        = "Receita Federal do Brasil"
    title            = "CNPJ Alfanumerico"
    accessed_at      = "2026-08-20"
    jurisdiction     = "BR"
    language         = "pt"
    notes            = "Keeps the fourteen positions and makes the first twelve alphanumeric from 2026, the two check digits staying numeric. It states that the value of a position for the check calculation is its ASCII code minus 48, so a digit keeps its value and A is 17, and that existing CNPJ keep their value and their check digits unchanged. The first alphanumeric CNPJ issued, 00.000.000/E08G-12, is carried in the corpus."
    license_or_terms = "Published by the issuing authority for public reference"
    tier             = "primary"
  }
}
