//go:build ignore

// Command genfixtures writes the deterministic decoder fixtures used by the
// `load_ruleset` conformance cases and by the minimal reference bundle shipped
// to the engine repositories.
//
// Run it with `go run tools/genfixtures.go`.
//
// Every hostile fixture must be invalid for exactly one reason: repair the fault
// the case is named for, and the bundle must load. Four of them failed that, and
// each time the same way - message_key, subject_node_circular, and
// program_expansion twice. Every load failure answers invalid_ruleset, so a
// second, unrelated fault makes the case pass for an engine that never
// implemented the rule under test, which is the engine the case exists to catch.
// Nothing here can detect it: write the repair as a test beside the fixture,
// watch it fail against the broken version, and only then trust the case.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/features"
	"github.com/entid-org/spec/internal/limits"
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
			Id:             "entid-reference-bundle",
			Url:            "https://github.com/entid-org/spec",
			Authority:      "LibEntID",
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
	// The padding node belongs to the canonicalization program: it was placed in
	// the format program and rooted there, so a canonicalization operation was
	// refused for being in a format program and the length of its name never
	// counted. Bring the length under the bound and the bundle stayed refused,
	// for a reason the case is not about - an engine with no bound on left_pad
	// passed it. The Swift engine measured that.
	//
	// So the node joins the canonicalization sequence, reachable from a root of
	// the kind that program requires, and its length is the only thing wrong.
	longPad := clone(base)
	{
		// The base bundle uses program 1 both as the definition's
		// canonicalization and as the dispatcher's pre-canonicalization, and
		// LEFT_PAD is forbidden in a pre-canonicalization program. So the
		// padding gets a canonicalization program of its own, referenced by the
		// definition, and the pre-canonicalizer keeps program 1.
		sequence := proto.Clone(base.Programs[0].Nodes[base.Programs[0].RootNode]).(*irv1.Node)
		sequence.InputNodes = []uint32{0}
		longPad.Programs = append(longPad.Programs, &irv1.Program{
			Id:   4,
			Kind: irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION,
			Nodes: []*irv1.Node{
				{
					OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
					Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
						Kind:   irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD,
						Text:   s("0"),
						Length: u(limits.MaxIndex + 1),
					}},
				},
				sequence,
			},
			RootNode: 1,
		})
		longPad.Identifiers[0].CanonicalizationProgram = 4
	}
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
		// The root has to reach the chain, or a generator emits none of it and
		// the fixture proves nothing. The first version pointed the root at the
		// original checksum node and left forty doubling nodes unreachable: the
		// Go engine refused it at check 25 for an undeclared capability while
		// this one refused it at check 14, and the case could not tell them
		// apart because both answers are invalid_ruleset.
		//
		// The second version pointed the root at the chain instead, and traded
		// one independent fault for another: a checksum program must be rooted
		// in a checksum outcome, and the chain produces a string. The TypeScript
		// engine measured it. So the chain feeds a checksum node appended after
		// it - reachable from the root, strictly lower operand indices, and the
		// root still produces what its program kind requires.
		root := proto.Clone(explosive.Programs[2].Nodes[explosive.Programs[2].RootNode]).(*irv1.Node)
		root.InputNodes = []uint32{first}
		nodes = append(nodes, root)
		explosive.Programs[2].Nodes = nodes
		explosive.Programs[2].RootNode = uint32(len(nodes) - 1)
	}
	// CONCAT belongs to STRING_VIEWS_V1, which the base bundle does not declare.
	// Without it the refusal comes from the undeclared capability rather than
	// from the bound under test.
	explosive.RequiredFeatureIds = append(explosive.RequiredFeatureIds, features.StringViewsV1)
	sort.Slice(explosive.RequiredFeatureIds, func(i, j int) bool {
		return explosive.RequiredFeatureIds[i] < explosive.RequiredFeatureIds[j]
	})
	write(filepath.Join(root, "program_expansion.binpb"), marshal(explosive))

	// A subject node built from the subject it defines. A generator emitting its
	// subtree recurses forever; an interpreter exhausts its budget. Found by an
	// engine that refused it rather than assuming the spec had thought of it.
	circularSubject := clone(base)
	{
		p := circularSubject.Programs[1]
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind:  irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM,
				Start: proto.Uint32(0),
			}},
		})
		subject := uint32(len(p.Nodes) - 1)
		p.SubjectNode = &subject
	}
	// The subject node is frozen into CAPTURES_AND_CALLS_V1 by features.md
	// section 11, so the fixture must declare it: without it, check 25 refuses
	// the bundle on its own, both checks answer invalid_ruleset, and the case is
	// satisfied by an engine that never implemented check 15's clause - the
	// engine it exists to catch. Three engines decoded these bytes and reported
	// it, after message_key and program_expansion had the same shape.
	circularSubject.RequiredFeatureIds = append(circularSubject.RequiredFeatureIds,
		features.StringViewsV1, features.CapturesAndCallsV1)
	sort.Slice(circularSubject.RequiredFeatureIds, func(i, j int) bool {
		return circularSubject.RequiredFeatureIds[i] < circularSubject.RequiredFeatureIds[j]
	})
	write(filepath.Join(root, "subject_node_circular.binpb"), marshal(circularSubject))

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

	// The same rule reached from the one side no fixture covered: a WHEN branch
	// that nothing references. The root case above was already refused, so the
	// clause forbidding a WHEN outside a CHOOSE had no case at all - the
	// TypeScript engine measured that the thirty five answers did not move when
	// the clause was added, which is what a rule with no case looks like.
	//
	// The program keeps its original root, so it is the absence of a parent and
	// nothing else that makes this bundle invalid.
	unreferencedWhen := clone(base)
	unreferencedWhen.Programs[2].Nodes = append(unreferencedWhen.Programs[2].Nodes,
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
	write(filepath.Join(root, "when_unreferenced.binpb"), marshal(unreferencedWhen))

	// prefix_in values out of order. ir.md section 9 puts them under the
	// normative order and requires a refusal, and this loader did not check it:
	// invisible while every engine scanned the list, load bearing the moment
	// section 14 required the lookup not to be linear, because a binary search
	// over an unsorted list answers wrongly rather than slowly.
	//
	// The base bundle carries no prefix_in, so the node is appended rather than
	// mutated: the first version reversed a list it never found and produced a
	// fixture that loaded clean.
	unsortedValues := clone(base)
	unsortedValues.Programs[1].Nodes = append(unsortedValues.Programs[1].Nodes,
		&irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
				Kind:   irv1.PredicateOpKind_PREDICATE_OP_KIND_PREFIX_IN,
				Values: []string{"CD", "AB"},
			}},
		})
	write(filepath.Join(root, "prefix_in_unsorted.binpb"), marshal(unsortedValues))

	// A prefix_in mixing element lengths. Sorted and deduplicated, so only the
	// mixed lengths are wrong: ["AB", "ABA"] against "ABCD" is where a search
	// for the greatest element not after the input finds "ABA", not a prefix,
	// while "AB" is one. All four prefix_in nodes of the published bundle hold a
	// single length, so nothing else in the corpus reaches this.
	mixedLengths := clone(base)
	mixedLengths.Programs[1].Nodes = append(mixedLengths.Programs[1].Nodes,
		&irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
				Kind:   irv1.PredicateOpKind_PREDICATE_OP_KIND_PREFIX_IN,
				Values: []string{"AB", "ABA"},
			}},
		})
	write(filepath.Join(root, "prefix_in_mixed_lengths.binpb"), marshal(mixedLengths))

	// A GLOBAL target declaring a prefix.
	globalPrefix := clone(base)
	globalPrefix.Dispatchers[0].Targets[0].AcceptedPrefixes = []string{"XX"}
	write(filepath.Join(root, "global_target_with_prefix.binpb"), marshal(globalPrefix))
}

func int64Ptr(v int64) *int64 { return &v }
