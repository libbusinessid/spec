# Routing pre-canonicalizer shared by every dispatcher.
#
# A pre-canonicalizer only performs the safe normalization required to route a
# value. It never adds, replaces or interprets a prefix: those transformations
# belong to the canonicalizer of the selected definition.
canonicalizer "dispatch" "identifier" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "/", "_"]),
  ]
}

# Routing pre-canonicalizer for structured identifiers whose separators carry
# meaning, such as the dot of an EUID. It never removes a business character.
canonicalizer "dispatch" "structured" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
  ]
}
