//go:build ignore

// Command genfixtures writes the deterministic decoder fixtures used by the
// `load_ruleset` conformance cases and by the minimal reference bundle shipped
// to the engine repositories.
//
// Run it with `go run tools/genfixtures.go`.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"

	conformancev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/conformance/v1"
	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/artifact"
	"github.com/libbusinessid/spec/internal/features"
	"github.com/libbusinessid/spec/internal/limits"
)

func s(v string) *string { return &v }
func u(v uint32) *uint32 { return &v }

// minimalBundle builds the smallest bundle that exercises the dispatcher, the
// canonicalizer, a format assertion and a Luhn checksum.
func minimalBundle() *irv1.RuleBundle {
	canon := &irv1.Program{
		Id:   1,
		Kind: irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION,
		Nodes: []*irv1.Node{
			{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind: irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_TRIM_WHITESPACE,
				}},
			},
			{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind: irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_UPPERCASE_ASCII,
				}},
			},
			{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				InputNodes: []uint32{0, 1},
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind: irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_SEQUENCE,
				}},
			},
		},
		RootNode: 2,
	}
	format := &irv1.Program{
		Id:   2,
		Kind: irv1.ProgramKind_PROGRAM_KIND_FORMAT,
		Nodes: []*irv1.Node{
			{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
				}},
			},
			{
				OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
					Kind:   irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_EQ,
					Length: u(4),
				}},
			},
			{
				OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION,
				InputNodes: []uint32{1},
				Operation: &irv1.Node_AssertionOperation{AssertionOperation: &irv1.AssertionOperation{
					Kind:       irv1.AssertionOpKind_ASSERTION_OP_KIND_REQUIRE,
					ReasonCode: reason(irv1.ReasonCode_REASON_CODE_INVALID_LENGTH),
					MessageKey: s("demo.length"),
				}},
			},
			{
				OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
					Kind: irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_DIGITS,
				}},
			},
			{
				OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION,
				InputNodes: []uint32{3},
				Operation: &irv1.Node_AssertionOperation{AssertionOperation: &irv1.AssertionOperation{
					Kind:       irv1.AssertionOpKind_ASSERTION_OP_KIND_REQUIRE,
					ReasonCode: reason(irv1.ReasonCode_REASON_CODE_INVALID_CHARACTERS),
					MessageKey: s("demo.characters"),
				}},
			},
			{
				OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION,
				InputNodes: []uint32{2, 4},
				Operation: &irv1.Node_AssertionOperation{AssertionOperation: &irv1.AssertionOperation{
					Kind: irv1.AssertionOpKind_ASSERTION_OP_KIND_SEQUENCE,
				}},
			},
		},
		RootNode: 5,
	}
	checksum := &irv1.Program{
		Id:   3,
		Kind: irv1.ProgramKind_PROGRAM_KIND_CHECKSUM,
		Nodes: []*irv1.Node{
			{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
				}},
			},
			{
				OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
					Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_LUHN,
				}},
			},
		},
		RootNode: 1,
	}
	definition := &irv1.IdentifierDefinition{
		Id:                      1,
		Kind:                    "demo",
		CanonicalizationProgram: 1,
		FormatProgram:           2,
		ChecksumProgram:         u(3),
		DefaultProfile:          "compatible",
		Sources: []*irv1.Source{{
			Id:             "libbusinessid-reference-bundle",
			Url:            "https://github.com/libbusinessid/spec",
			Authority:      "LibBusinessID",
			Title:          "Minimal reference bundle",
			AccessedAt:     "2026-08-18",
			Jurisdiction:   "GLOBAL",
			Language:       "en",
			Notes:          "Synthetic demonstration rule: four ASCII digits closed by a Luhn check digit.",
			LicenseOrTerms: "Apache-2.0",
			Tier:           irv1.SourceTier_SOURCE_TIER_PRIMARY,
		}},
	}
	dispatcher := &irv1.IdentifierDispatcher{
		Kind:                       "demo",
		PreCanonicalizationProgram: 1,
		Targets: []*irv1.DispatchTarget{{
			IdentifierDefinitionId: 1,
		}},
	}
	bundle := &irv1.RuleBundle{
		FormatVersion: artifact.SupportedFormatVersion,
		RulesVersion:  "2026.08.0",
		RequiredFeatureIds: []uint32{
			features.CoreGraphV1, features.AsciiAndWhitespaceV1, features.CanonicalizationBasicV1,
			features.IdentifierDispatchV1, features.FormatAssertionsV1, features.ProfilesV1,
			features.ChecksumTristateV1, features.ChecksumLuhnV1, features.ProvenanceV1,
			features.ProvenanceTierV1,
		},
		SourceDigest: make([]byte, 32),
		Identifiers:  []*irv1.IdentifierDefinition{definition},
		Programs:     []*irv1.Program{canon, format, checksum},
		Dispatchers:  []*irv1.IdentifierDispatcher{dispatcher},
	}
	return bundle
}

func reason(r irv1.ReasonCode) *irv1.ReasonCode { return &r }

func clone(b *irv1.RuleBundle) *irv1.RuleBundle {
	return proto.Clone(b).(*irv1.RuleBundle)
}

func write(path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("%s: %d bytes\n", path, len(data))
}

func marshal(b *irv1.RuleBundle) []byte {
	raw, err := artifact.Marshal(b)
	if err != nil {
		panic(err)
	}
	return raw
}

//nolint:funlen // one flat table of decoder fixtures.
func main() {
	root := "testdata/bundles"
	base := minimalBundle()
	valid := marshal(base)
	if _, err := artifact.LoadRuleset(valid); err != nil {
		panic("the minimal reference bundle must be valid: " + err.Error())
	}
	write(filepath.Join(root, "minimal_valid.binpb"), valid)

	// The minimal conformance suite that accompanies the reference bundle.
	suite := &conformancev1.ConformanceBundle{
		FormatVersion: artifact.SupportedFormatVersion,
		RulesVersion:  "2026.08.0",
		SourceDigest:  make([]byte, 32),
		Cases: []*conformancev1.ConformanceCase{
			{
				Id: "demo-valid-001", Description: "Four digits closed by a Luhn check digit",
				Kind: "demo", Input: "1230", Profile: "compatible",
				Operation: conformancev1.Operation_OPERATION_VALIDATE,
				Expected: &conformancev1.ExpectedOutcome{Value: &conformancev1.ExpectedOutcome_ValidationReport{
					ValidationReport: &conformancev1.ExpectedValidationReport{
						Kind: "demo", InputValue: "1230", CanonicalValue: "1230",
						Profile: "compatible", RulesVersion: "2026.08.0",
						FormatVersion: artifact.SupportedFormatVersion,
						Format: &conformancev1.ExpectedStep{
							Level:      conformancev1.ValidationLevel_VALIDATION_LEVEL_FORMAT,
							Status:     conformancev1.StepStatus_STEP_STATUS_VALID,
							ReasonCode: irv1.ReasonCode_REASON_CODE_OK,
						},
						Checksum: &conformancev1.ExpectedStep{
							Level:      conformancev1.ValidationLevel_VALIDATION_LEVEL_CHECKSUM,
							Status:     conformancev1.StepStatus_STEP_STATUS_VALID,
							ReasonCode: irv1.ReasonCode_REASON_CODE_OK,
						},
					},
				}},
				Tags:                []string{"reference", "synthetic", "valid"},
				DataClassification:  "synthetic",
				RedistributionBasis: "Synthetic demonstration value of the reference bundle.",
			},
			{
				Id: "demo-invalid-checksum-002", Description: "A mutated check digit is invalid",
				Kind: "demo", Input: "1231", Profile: "compatible",
				Operation: conformancev1.Operation_OPERATION_VALIDATE,
				Expected: &conformancev1.ExpectedOutcome{Value: &conformancev1.ExpectedOutcome_ValidationReport{
					ValidationReport: &conformancev1.ExpectedValidationReport{
						Kind: "demo", InputValue: "1231", CanonicalValue: "1231",
						Profile: "compatible", RulesVersion: "2026.08.0",
						FormatVersion: artifact.SupportedFormatVersion,
						Format: &conformancev1.ExpectedStep{
							Level:      conformancev1.ValidationLevel_VALIDATION_LEVEL_FORMAT,
							Status:     conformancev1.StepStatus_STEP_STATUS_VALID,
							ReasonCode: irv1.ReasonCode_REASON_CODE_OK,
						},
						Checksum: &conformancev1.ExpectedStep{
							Level:      conformancev1.ValidationLevel_VALIDATION_LEVEL_CHECKSUM,
							Status:     conformancev1.StepStatus_STEP_STATUS_INVALID,
							ReasonCode: irv1.ReasonCode_REASON_CODE_INVALID_CHECKSUM,
						},
					},
				}},
				Tags:                []string{"invalid", "reference", "synthetic"},
				DataClassification:  "synthetic",
				RedistributionBasis: "Synthetic demonstration value of the reference bundle.",
			},
		},
	}
	suiteBytes, err := artifact.Marshal(suite)
	if err != nil {
		panic(err)
	}
	write(filepath.Join(root, "minimal_conformance.binpb"), suiteBytes)

	// Truncated payload.
	write(filepath.Join(root, "truncated.binpb"), valid[:len(valid)/2])

	// Empty payload.
	write(filepath.Join(root, "empty.binpb"), nil)

	// Unknown field at the root, encoded with an unused field number.
	unknown := append([]byte(nil), valid...)
	unknown = append(unknown, 0xF8, 0x3E, 0x01) // field 999, varint 1
	write(filepath.Join(root, "unknown_field_root.binpb"), unknown)

	// Unsupported format version.
	fv := clone(base)
	fv.FormatVersion = 2
	write(filepath.Join(root, "unsupported_format_version.binpb"), marshal(fv))

	// Unknown capability id.
	feat := clone(base)
	feat.RequiredFeatureIds = append(feat.RequiredFeatureIds, 9999)
	write(filepath.Join(root, "unknown_feature.binpb"), marshal(feat))

	// A capability used but not declared.
	undeclared := clone(base)
	undeclared.RequiredFeatureIds = []uint32{features.CoreGraphV1}
	write(filepath.Join(root, "undeclared_feature.binpb"), marshal(undeclared))

	// Digest of the wrong length.
	digest := clone(base)
	digest.SourceDigest = make([]byte, 16)
	write(filepath.Join(root, "short_digest.binpb"), marshal(digest))

	// Empty business version.
	version := clone(base)
	version.RulesVersion = ""
	write(filepath.Join(root, "empty_rules_version.binpb"), marshal(version))

	// A node without any operation.
	noop := clone(base)
	noop.Programs[1].Nodes[1].Operation = nil
	write(filepath.Join(root, "missing_operation.binpb"), marshal(noop))

	// A node referencing a higher index.
	forward := clone(base)
	forward.Programs[1].Nodes[1].InputNodes = []uint32{5}
	write(filepath.Join(root, "node_forward_reference.binpb"), marshal(forward))

	// A node referencing an index outside the program.
	outOfRange := clone(base)
	outOfRange.Programs[1].RootNode = 99
	write(filepath.Join(root, "node_out_of_range.binpb"), marshal(outOfRange))

	// An operand of the wrong type.
	typeMismatch := clone(base)
	typeMismatch.Programs[1].Nodes[1].InputNodes = []uint32{0}
	typeMismatch.Programs[1].Nodes[1].OutputType = irv1.ValueType_VALUE_TYPE_STRING
	write(filepath.Join(root, "type_mismatch.binpb"), marshal(typeMismatch))

	// An UNSPECIFIED enum value.
	unspecified := clone(base)
	unspecified.Programs[0].Kind = irv1.ProgramKind_PROGRAM_KIND_UNSPECIFIED
	write(filepath.Join(root, "unspecified_enum.binpb"), marshal(unspecified))

	// A self recursive call.
	cycle := clone(base)
	cycle.Programs[2].Nodes = append(cycle.Programs[2].Nodes, &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
		InputNodes: []uint32{0},
		Operation: &irv1.Node_CallOperation{CallOperation: &irv1.CallOperation{
			Kind:      irv1.CallOpKind_CALL_OP_KIND_CHECKSUM,
			ProgramId: 3,
		}},
	})
	cycle.Programs[2].RootNode = 2
	write(filepath.Join(root, "call_cycle.binpb"), marshal(cycle))

	// A call towards an unknown program.
	unknownCall := clone(base)
	unknownCall.Programs[2].Nodes = append(unknownCall.Programs[2].Nodes, &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
		InputNodes: []uint32{0},
		Operation: &irv1.Node_CallOperation{CallOperation: &irv1.CallOperation{
			Kind:      irv1.CallOpKind_CALL_OP_KIND_CHECKSUM,
			ProgramId: 42,
		}},
	})
	unknownCall.Programs[2].RootNode = 2
	write(filepath.Join(root, "unknown_call_target.binpb"), marshal(unknownCall))

	// A definition no dispatcher references.
	orphan := clone(base)
	orphan.Identifiers = append(orphan.Identifiers, &irv1.IdentifierDefinition{
		Id:                      2,
		Kind:                    "demo",
		CountryCode:             s("FR"),
		CanonicalizationProgram: 1,
		FormatProgram:           2,
		ChecksumProgram:         u(3),
		DefaultProfile:          "compatible",
	})
	write(filepath.Join(root, "orphan_definition.binpb"), marshal(orphan))

	// The same prefix claimed by two targets.
	duplicatePrefix := clone(base)
	duplicatePrefix.Identifiers[0].CountryCode = s("BE")
	duplicatePrefix.Identifiers = append(duplicatePrefix.Identifiers, &irv1.IdentifierDefinition{
		Id:                      2,
		Kind:                    "demo",
		CountryCode:             s("FR"),
		CanonicalizationProgram: 1,
		FormatProgram:           2,
		ChecksumProgram:         u(3),
		DefaultProfile:          "compatible",
	})
	duplicatePrefix.Dispatchers[0].Targets = []*irv1.DispatchTarget{
		{CountryCode: s("BE"), AcceptedPrefixes: []string{"XX"}, IdentifierDefinitionId: 1},
		{CountryCode: s("FR"), AcceptedPrefixes: []string{"XX"}, IdentifierDefinitionId: 2},
	}
	write(filepath.Join(root, "duplicate_prefix.binpb"), marshal(duplicatePrefix))

	// A reason code that cannot prove an invalidity.
	badReason := clone(base)
	badReason.Programs[1].Nodes[2].GetAssertionOperation().ReasonCode =
		reason(irv1.ReasonCode_REASON_CODE_UNSUPPORTED_KIND)
	write(filepath.Join(root, "forbidden_reason_code.binpb"), marshal(badReason))

	// A parameter that does not belong to the operation.
	strayParam := clone(base)
	strayParam.Programs[1].Nodes[3].GetPredicateOperation().Text = s("0123456789")
	write(filepath.Join(root, "stray_parameter.binpb"), marshal(strayParam))

	// An unbounded digits_to_integer.
	unbounded := clone(base)
	unbounded.Programs[2].Nodes = []*irv1.Node{
		{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
			}},
		},
		{
			OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
				Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_DIGITS_TO_INTEGER,
			}},
		},
		{
			OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
			InputNodes: []uint32{1, 0},
			Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
				Kind:  irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT,
				Index: u(0),
			}},
		},
	}
	unbounded.Programs[2].RootNode = 2
	write(filepath.Join(root, "unbounded_digits_to_integer.binpb"), marshal(unbounded))

	// An out of range modulus.
	badModulus := clone(base)
	badModulus.Programs[2].Nodes = []*irv1.Node{
		{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
			}},
		},
		{
			OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
				Kind:    irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS,
				Modulus: int64Ptr(1),
			}},
		},
		{
			OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
			InputNodes: []uint32{1, 0},
			Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
				Kind:  irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT,
				Index: u(0),
			}},
		},
	}
	badModulus.Programs[2].RootNode = 2
	write(filepath.Join(root, "modulus_out_of_range.binpb"), marshal(badModulus))

	// A left_pad asking for more code points than a slice bound allows. An
	// engine that compiles the rules ahead of time sizes its work buffer from
	// this number, so an unbounded length is an allocation primitive.
	longPad := clone(base)
	longPad.Programs[1].Nodes = append(longPad.Programs[1].Nodes,
		&irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
			Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
				Kind:   irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD,
				Text:   s("0"),
				Length: u(limits.MaxIndex + 1),
			}},
		})
	longPad.Programs[1].RootNode = uint32(len(longPad.Programs[1].Nodes) - 1)
	write(filepath.Join(root, "left_pad_length.binpb"), marshal(longPad))

	// A message key that is present and empty. Absence and emptiness are
	// indistinguishable in an idiomatic API, so two engines could report
	// differently on the same bundle.
	emptyKey := clone(base)
	marked := false
	for _, p := range emptyKey.GetPrograms() {
		for _, n := range p.GetNodes() {
			if a := n.GetAssertionOperation(); a != nil {
				a.MessageKey = s("")
				marked = true
				break
			}
		}
		if marked {
			break
		}
	}
	if !marked {
		panic("genfixtures: the base bundle holds no assertion to give an empty message key")
	}
	write(filepath.Join(root, "empty_message_key.binpb"), marshal(emptyKey))

	// A predicate comparing against a constant beyond the accepted range. The
	// checksum side of COMPARE_CONSTANT was bounded when the two opcodes landed;
	// the predicate side was not.
	wideConstant := clone(base)
	wideConstant.Programs[2].Nodes = append(wideConstant.Programs[2].Nodes,
		&irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
				Kind:    irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS,
				Modulus: int64Ptr(97),
			}},
		},
		&irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
			InputNodes: []uint32{uint32(len(wideConstant.Programs[2].Nodes))},
			Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
				Kind:     irv1.PredicateOpKind_PREDICATE_OP_KIND_INTEGER_IS,
				Constant: int64Ptr(limits.MaxConstant + 1),
			}},
		})
	// The bundle must declare what it uses, or the refusal comes from the
	// undeclared capability rather than from the bound under test.
	wideConstant.RequiredFeatureIds = append(wideConstant.RequiredFeatureIds,
		features.ChecksumIntegerPredicateV1)
	sort.Slice(wideConstant.RequiredFeatureIds, func(i, j int) bool {
		return wideConstant.RequiredFeatureIds[i] < wideConstant.RequiredFeatureIds[j]
	})
	write(filepath.Join(root, "predicate_constant.binpb"), marshal(wideConstant))

	// The alphabet of a custom mapping. Five refusals, one fixture each: the Go
	// engine implemented all five and pointed out that none of them was in the
	// corpus, so nothing held the other engines to the same answers. The
	// repeated code point is the one that matters most: it would give one
	// character two values, and which one an engine returned would depend on how
	// it searched.
	alphabetFixture := func(name string, mapping irv1.CharMapping, alphabet *string) {
		b := clone(base)
		b.Programs[2].Nodes = append(b.Programs[2].Nodes,
			&irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
					Kind:      irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM,
					Weights:   []int64{1, 2},
					Alignment: irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT.Enum(),
					Mapping:   mapping.Enum(),
					Alphabet:  alphabet,
				}},
			})
		b.RequiredFeatureIds = append(b.RequiredFeatureIds,
			features.ChecksumWeightedV1, features.ChecksumCustomAlphabetV1)
		sort.Slice(b.RequiredFeatureIds, func(i, j int) bool {
			return b.RequiredFeatureIds[i] < b.RequiredFeatureIds[j]
		})
		write(filepath.Join(root, name), marshal(b))
	}
	repeated := "0123401234"
	empty := ""
	tooMany := strings.Builder{}
	for i := range limits.MaxAlphabetRunes + 1 {
		tooMany.WriteRune(rune('\u0100' + i))
	}
	tooManyRunes := tooMany.String()
	digits := "0123456789"
	custom := irv1.CharMapping_CHAR_MAPPING_CUSTOM_ALPHABET
	alphabetFixture("alphabet_repeated.binpb", custom, &repeated)
	alphabetFixture("alphabet_empty.binpb", custom, &empty)
	alphabetFixture("alphabet_too_many.binpb", custom, &tooManyRunes)
	alphabetFixture("alphabet_missing.binpb", custom, nil)
	alphabetFixture("alphabet_unread.binpb", irv1.CharMapping_CHAR_MAPPING_DIGIT_VALUE, &digits)

	// A graph whose every node reads the previous one twice. The node count is
	// bounded, the graph is acyclic, the depth is fine - and a generator that
	// inlines repeated operands emits 2^n instances. The TypeScript engine hit
	// this the moment it stopped interpreting.
	explosive := clone(base)
	{
		// Node 0 of a checksum program is the subject, which is the string this
		// chain needs; the last node is a checksum outcome.
		nodes := explosive.Programs[2].Nodes
		first := uint32(0)
		for range 40 {
			nodes = append(nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				InputNodes: []uint32{first, first},
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_CONCAT,
				}},
			})
			first = uint32(len(nodes) - 1)
		}
		explosive.Programs[2].Nodes = nodes
		// The root has to reach the chain, or a generator emits none of it and
		// the fixture proves nothing. The first version pointed the root at the
		// original checksum node and left forty doubling nodes unreachable: the
		// Go engine refused it at check 25 for an undeclared capability while
		// this one refused it at check 14, and the case could not tell them
		// apart because both answers are invalid_ruleset.
		explosive.Programs[2].RootNode = first
	}
	// CONCAT belongs to STRING_VIEWS_V1, which the base bundle does not declare.
	// Without it the refusal comes from the undeclared capability rather than
	// from the bound under test.
	explosive.RequiredFeatureIds = append(explosive.RequiredFeatureIds, features.StringViewsV1)
	sort.Slice(explosive.RequiredFeatureIds, func(i, j int) bool {
		return explosive.RequiredFeatureIds[i] < explosive.RequiredFeatureIds[j]
	})
	write(filepath.Join(root, "program_expansion.binpb"), marshal(explosive))

	// A source stating a tier outside the enumeration. UNSPECIFIED is not a
	// refusal - it means the source states no tier - but a value nothing can
	// read is a forged bundle.
	badTier := clone(base)
	badTier.Identifiers[0].Sources[0].Tier = irv1.SourceTier(47)
	write(filepath.Join(root, "source_tier_unknown.binpb"), marshal(badTier))

	// A rules_version carrying a control character. The Go engine found this by
	// fuzzing: the version is interpolated into generated sources, and a NUL is
	// not valid Go. Only emptiness was refused.
	badVersion := clone(base)
	badVersion.RulesVersion = "2026.08.0\x00"
	write(filepath.Join(root, "rules_version_shape.binpb"), marshal(badVersion))

	// A WHEN branch outside a CHOOSE.
	strayWhen := clone(base)
	strayWhen.Programs[2].Nodes = append(strayWhen.Programs[2].Nodes,
		&irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
				Kind: irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_DIGITS,
			}},
		},
		&irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
			InputNodes: []uint32{2, 1},
			Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
				Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_WHEN,
			}},
		})
	strayWhen.Programs[2].RootNode = 3
	write(filepath.Join(root, "stray_when_branch.binpb"), marshal(strayWhen))

	// A GLOBAL target declaring a prefix.
	globalPrefix := clone(base)
	globalPrefix.Dispatchers[0].Targets[0].AcceptedPrefixes = []string{"XX"}
	write(filepath.Join(root, "global_target_with_prefix.binpb"), marshal(globalPrefix))
}

func int64Ptr(v int64) *int64 { return &v }
