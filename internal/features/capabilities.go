// Package features is the single Go registry of the immutable LibBusinessID
// capability IDs and of every concrete IR operation. Both docs/ir.md and
// docs/features.md are generated from this registry; CI rejects any divergence.
//
// The content of a capability ID is frozen forever. A new operation, a new
// variant of an operation or any observable change always receives a new ID.
package features

import "sort"

// Capability IDs of the V1 registry (spec.md 7.4). These values are immutable.
const (
	CoreGraphV1                   uint32 = 1
	AsciiAndWhitespaceV1          uint32 = 2
	CanonicalizationBasicV1       uint32 = 3
	CanonicalizationConditionalV1 uint32 = 4
	IdentifierDispatchV1          uint32 = 5
	StringViewsV1                 uint32 = 10
	CapturesAndCallsV1            uint32 = 11
	FormatAssertionsV1            uint32 = 20
	ProfilesV1                    uint32 = 21
	ChecksumTristateV1            uint32 = 30
	ChecksumLuhnV1                uint32 = 31
	ChecksumMod97V1               uint32 = 32
	ChecksumWeightedV1            uint32 = 33
	ChecksumCompareConstantV1     uint32 = 34
	ChecksumIntegerPredicateV1    uint32 = 35
	ProvenanceV1                  uint32 = 40
	ProvenanceTierV1              uint32 = 41
	ChecksumCustomAlphabetV1      uint32 = 42
)

// Capability describes one immutable capability ID.
type Capability struct {
	// ID is the numeric capability identifier. It is never renumbered.
	ID uint32
	// Name is the immutable symbolic name.
	Name string
	// Summary is the one line description used in the capability table.
	Summary string
	// Content lists the exact and frozen content of the capability, beyond the
	// operations that reference it.
	Content []string
}

// capabilities is the frozen V1 registry, ordered by ascending ID.
var capabilities = []Capability{
	{
		ID:      CoreGraphV1,
		Name:    "CORE_GRAPH_V1",
		Summary: "programs, topological nodes, typed values",
		Content: []string{
			"`RuleBundle`, `Program`, `Node` and the `ValueType` enum.",
			"Programs are typed acyclic graphs whose nodes only reference strictly lower indices.",
			"`root_node` designates the single result node of a program.",
			"Structural limits of docs/ir.md section \"Execution limits\".",
			"Rejection of an absent `oneof operation`, of an `UNSPECIFIED` enum value and of any unknown Protobuf field at any depth.",
		},
	},
	{
		ID:      AsciiAndWhitespaceV1,
		Name:    "ASCII_AND_WHITESPACE_V1",
		Summary: "ASCII classes and the frozen whitespace table",
		Content: []string{
			"ASCII digit class `U+0030..U+0039`.",
			"ASCII upper letter class `U+0041..U+005A`, ASCII lower letter class `U+0061..U+007A`.",
			"The frozen `whitespace_v1` table: U+0009..U+000D, U+0020, U+0085, U+00A0, U+1680, U+2000..U+200A, U+2028, U+2029, U+202F, U+205F, U+3000, U+FEFF.",
			"`uppercase_ascii` maps only `a..z` to `A..Z` and never consults a locale.",
			"Code point based positions and lengths.",
		},
	},
	{
		ID:      CanonicalizationBasicV1,
		Name:    "CANONICALIZATION_BASIC_V1",
		Summary: "trim, uppercase, remove, prefix, pad, insert",
		Content: []string{
			"Sequential canonicalization programs and the `SEQUENCE` step.",
			"Unconditional canonicalization steps listed in docs/ir.md.",
			"A canonicalization program never truncates and never fails on user input.",
		},
	},
	{
		ID:      CanonicalizationConditionalV1,
		Name:    "CANONICALIZATION_CONDITIONAL_V1",
		Summary: "`when` and predicates over the current value",
		Content: []string{
			"The `WHEN` canonicalization step.",
			"Evaluation of a predicate against the value current at the moment the step runs.",
			"No memoization is observable across canonicalization steps.",
		},
	},
	{
		ID:      IdentifierDispatchV1,
		Name:    "IDENTIFIER_DISPATCH_V1",
		Summary: "kind/country/prefixes, separated alias spaces and two phase selection",
		Content: []string{
			"`IdentifierDispatcher`, `CountryAlias`, `DispatchTarget` and the pre-canonicalization program.",
			"The nine step dispatch algorithm of docs/ir.md.",
			"Separated alias spaces for kinds, countries and prefixes.",
			"`GLOBAL` targets expressed by the absence of `country_code`.",
			"The `country_code()` string constructor and the `PREPEND_COUNTRY_IF_MISSING` canonicalization step.",
		},
	},
	{
		ID:      StringViewsV1,
		Name:    "STRING_VIEWS_V1",
		Summary: "slice, before/after, concat, absence",
		Content: []string{
			"Possibly absent string views and their propagation rules.",
			"The string view constructors listed in docs/ir.md.",
			"`IS_ABSENT`, the only predicate that observes absence as true.",
		},
	},
	{
		ID:      CapturesAndCallsV1,
		Name:    "CAPTURES_AND_CALLS_V1",
		Summary: "captures and program reuse",
		Content: []string{
			"`Program.captures` and the per format capture limit.",
			"`CallOperation` for `use_format` and `apply_checksum`.",
			"The acyclic, typed call graph of static depth at most 32.",
			"`Program.subject_node` and the subject supplied by a caller.",
		},
	},
	{
		ID:      FormatAssertionsV1,
		Name:    "FORMAT_ASSERTIONS_V1",
		Summary: "predicates and ordered assertions",
		Content: []string{
			"`AssertionOperation` with `SEQUENCE` and `REQUIRE`.",
			"Ordered evaluation: the first failing assertion determines the reason code and the message key.",
			"The non ASCII specific predicates listed in docs/ir.md.",
			"Reason codes usable by `REQUIRE`, restricted to codes that prove invalidity.",
		},
	},
	{
		ID:      ProfilesV1,
		Name:    "PROFILES_V1",
		Summary: "compatible and strict_current",
		Content: []string{
			"`IdentifierDefinition.default_profile` restricted to `compatible` and `strict_current`.",
			"The `PROFILE_IS` predicate.",
			"`compatible` is the normative default and never restricts the canonicalization shared by both profiles.",
		},
	},
	{
		ID:      ChecksumTristateV1,
		Name:    "CHECKSUM_TRISTATE_V1",
		Summary: "valid, invalid, unsupported and branching",
		Content: []string{
			"`ChecksumOperation` and the tri-state checksum outcome.",
			"`CHOOSE`, `WHEN`, `ALL_CHECKS`, `ANY_CHECK` and `UNSUPPORTED`.",
			"`COMPARE_DIGIT` and `COMPARE_SLICE`.",
			"The integer constructors `DIGITS_TO_INTEGER`, `MOD_DIGITS`, `MODULO`, `COMPLEMENT` and `REMAINDER_MAP` with their checked arithmetic bounds.",
			"Propagation of an indeterminate integer to `unsupported`/`unsupported_checksum`.",
			"`IdentifierDefinition.absent_checksum_reason`.",
		},
	},
	{
		ID:      ChecksumLuhnV1,
		Name:    "CHECKSUM_LUHN_V1",
		Summary: "Luhn",
		Content: []string{
			"The `LUHN` checksum operation over ASCII digits only.",
		},
	},
	{
		ID:      ChecksumMod97V1,
		Name:    "CHECKSUM_MOD97_V1",
		Summary: "ISO 7064 / modulo 97",
		Content: []string{
			"The `ISO7064_MOD97_10` checksum operation with base 36 expansion of ASCII letters.",
		},
	},
	{
		ID:      ChecksumWeightedV1,
		Name:    "CHECKSUM_WEIGHTED_V1",
		Summary: "weighted sums, alignments and remainders",
		Content: []string{
			"The `WEIGHTED_SUM` integer operation.",
			"The `WeightAlignment` values `LEFT`, `RIGHT` and `CYCLE`.",
			"The `CharMapping` values `DIGIT_VALUE` and `ALNUM_BASE36`.",
		},
	},
	{
		ID:      ChecksumCompareConstantV1,
		Name:    "CHECKSUM_COMPARE_CONSTANT_V1",
		Summary: "comparison against a literal constant",
		Content: []string{
			"The `COMPARE_CONSTANT` checksum operation.",
			"It closes the gap left by `COMPARE_DIGIT` and `COMPARE_SLICE`, which can only compare a computed integer against part of the value being checked. A rule stating that a remainder must equal zero has nothing in the value to compare against.",
		},
	},
	{
		ID:      ChecksumIntegerPredicateV1,
		Name:    "CHECKSUM_INTEGER_PREDICATE_V1",
		Summary: "branching on the value of an integer",
		Content: []string{
			"The `INTEGER_IS` predicate.",
			"Every other predicate reads a string, so a checksum could compare a remainder but never branch on it. Registers that recompute their sum with a second set of weights when the first remainder reaches a given value need exactly that.",
		},
	},
	{
		ID:      ProvenanceV1,
		Name:    "PROVENANCE_V1",
		Summary: "sources linked to definitions",
		Content: []string{
			"`Source` and `IdentifierDefinition.sources`, sorted by source id.",
			"Every rule able to reject an input carries at least one source.",
		},
	},
	{
		ID:      ProvenanceTierV1,
		Name:    "PROVENANCE_TIER_V1",
		Summary: "how close a source sits to the authority",
		Content: []string{
			"`Source.tier` and the `SourceTier` enumeration.",
			"It is a capability of its own rather than an addition to `PROVENANCE_V1`, whose content is frozen: a field added to a frozen capability reaches an engine as an unknown field, which reads as a forged bundle rather than as the version gap it is.",
		},
	},
	{
		ID:      ChecksumCustomAlphabetV1,
		Name:    "CHECKSUM_CUSTOM_ALPHABET_V1",
		Summary: "weighted sums over an issuer's own alphabet",
		Content: []string{
			"The `CharMapping` value `CUSTOM_ALPHABET`, and the `alphabet` field of `IntegerOperation` that carries its ordered code points.",
			"The value of a code point is its index in that alphabet. A code point absent from it makes the sum indeterminate, exactly as a letter does under `DIGIT_VALUE`.",
			"It exists because an issuer's alphabet is often neither base 10 nor base 36. Letters that are misread as digits get dropped, and every letter after the gap shifts: the Chinese unified social credit code omits I, O, S, V and Z, so its `J` is 18 where `ALNUM_BASE36` makes it 19. Without this, such a checksum cannot be stated at all, and a published algorithm that cannot be stated has to be reported `unsupported`.",
		},
	},
}

// All returns the frozen capability registry ordered by ascending ID.
func All() []Capability {
	out := make([]Capability, len(capabilities))
	copy(out, capabilities)
	return out
}

// byID indexes the registry for constant time lookups.
var byID = func() map[uint32]Capability {
	m := make(map[uint32]Capability, len(capabilities))
	for _, c := range capabilities {
		m[c.ID] = c
	}
	return m
}()

// Lookup returns the capability with the given ID.
func Lookup(id uint32) (Capability, bool) {
	c, ok := byID[id]
	return c, ok
}

// Known reports whether the capability ID belongs to the V1 registry.
func Known(id uint32) bool {
	_, ok := byID[id]
	return ok
}

// Set collects capability IDs and returns them sorted and deduplicated.
type Set struct {
	ids map[uint32]struct{}
}

// NewSet returns an empty capability set.
func NewSet() *Set { return &Set{ids: make(map[uint32]struct{})} }

// Add records the capability IDs.
func (s *Set) Add(ids ...uint32) {
	for _, id := range ids {
		s.ids[id] = struct{}{}
	}
}

// Contains reports whether the set holds the capability ID.
func (s *Set) Contains(id uint32) bool {
	_, ok := s.ids[id]
	return ok
}

// Sorted returns the capability IDs in ascending order.
func (s *Set) Sorted() []uint32 {
	out := make([]uint32, 0, len(s.ids))
	for id := range s.ids {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
