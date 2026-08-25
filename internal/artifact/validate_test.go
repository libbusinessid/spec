package artifact_test

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/features"
)

func baseBundle(t *testing.T) *irv1.RuleBundle {
	t.Helper()
	bundle := &irv1.RuleBundle{}
	if err := proto.Unmarshal(loadMinimal(t), bundle); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func expectRejection(t *testing.T, bundle *irv1.RuleBundle, kind artifact.ErrorKind, substring string) {
	t.Helper()
	raw, err := artifact.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	_, err = artifact.LoadRuleset(raw)
	if err == nil {
		t.Fatalf("expected a rejection mentioning %q", substring)
	}
	typed, ok := err.(*artifact.Error) //nolint:errorlint // the loader returns this exact type
	if !ok {
		t.Fatalf("the loader must return a typed error, got %T", err)
	}
	if typed.Kind != kind {
		t.Fatalf("expected kind %q, got %q (%s)", kind, typed.Kind, typed.Detail)
	}
	if !strings.Contains(typed.Detail, substring) {
		t.Fatalf("expected %q, got %q", substring, typed.Detail)
	}
}

func str(v string) *string { return &v }
func u32(v uint32) *uint32 { return &v }
func i64(v int64) *int64   { return &v }

func TestBundleRejections(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(b *irv1.RuleBundle)
		kind      artifact.ErrorKind
		substring string
	}{
		{"zero program id", func(b *irv1.RuleBundle) { b.Programs[0].Id = 0 },
			artifact.ErrInvalid, "program ids start at 1"},
		{"duplicate program id", func(b *irv1.RuleBundle) { b.Programs[1].Id = b.Programs[0].GetId() },
			artifact.ErrInvalid, "duplicate program id"},
		{"self referencing node", func(b *irv1.RuleBundle) { b.Programs[1].Nodes[1].InputNodes = []uint32{1} },
			artifact.ErrInvalid, "which is not lower"},
		{"empty program", func(b *irv1.RuleBundle) { b.Programs[0].Nodes = nil },
			artifact.ErrInvalid, "holds no node"},
		{"canonicalization root", func(b *irv1.RuleBundle) { b.Programs[0].RootNode = 0 },
			artifact.ErrInvalid, "rooted in a SEQUENCE"},
		{"canonicalization subject", func(b *irv1.RuleBundle) { b.Programs[0].SubjectNode = u32(0) },
			artifact.ErrInvalid, "must not declare a subject"},
		{"canonicalization capture", func(b *irv1.RuleBundle) {
			b.Programs[0].Captures = []*irv1.Capture{{Name: "x", Node: 0}}
		}, artifact.ErrInvalid, "must not declare captures"},
		{"format root", func(b *irv1.RuleBundle) { b.Programs[1].RootNode = 0 },
			artifact.ErrInvalid, "rooted in an assertion SEQUENCE"},
		{"checksum root type", func(b *irv1.RuleBundle) { b.Programs[2].RootNode = 0 },
			artifact.ErrInvalid, "must produce a checksum outcome"},
		{"checksum capture", func(b *irv1.RuleBundle) {
			b.Programs[2].Captures = []*irv1.Capture{{Name: "x", Node: 0}}
		}, artifact.ErrInvalid, "must not declare captures"},
		{"subject out of range", func(b *irv1.RuleBundle) { b.Programs[1].SubjectNode = u32(99) },
			artifact.ErrInvalid, "out of range subject node"},
		{"subject wrong type", func(b *irv1.RuleBundle) { b.Programs[1].SubjectNode = u32(1) },
			artifact.ErrInvalid, "non string subject node"},
		{"unnamed capture", func(b *irv1.RuleBundle) {
			b.Programs[1].Captures = []*irv1.Capture{{Name: "", Node: 0}}
		}, artifact.ErrInvalid, "unnamed capture"},
		{"duplicate capture", func(b *irv1.RuleBundle) {
			b.Programs[1].Captures = []*irv1.Capture{{Name: "x", Node: 0}, {Name: "x", Node: 0}}
		}, artifact.ErrInvalid, "twice"},
		{"capture out of range", func(b *irv1.RuleBundle) {
			b.Programs[1].Captures = []*irv1.Capture{{Name: "x", Node: 99}}
		}, artifact.ErrInvalid, "out of range capture node"},
		{"capture wrong type", func(b *irv1.RuleBundle) {
			b.Programs[1].Captures = []*irv1.Capture{{Name: "x", Node: 1}}
		}, artifact.ErrInvalid, "non string capture"},
		{"unknown opcode", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes[1].GetPredicateOperation().Kind = irv1.PredicateOpKind(120)
		}, artifact.ErrInvalid, "unknown predicate operation"},
		{"wrong output type", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes[0].OutputType = irv1.ValueType_VALUE_TYPE_INTEGER
		}, artifact.ErrInvalid, "declares output"},
		// Appended rather than substituted for node 0. Replacing a node the
		// SEQUENCE root reads breaks the operand type too, and the operand types
		// are check 11 while the categories are check 16: the bundle would be
		// refused for the wrong fault, and the case would stop proving the rule
		// it names. The corpus has been bitten by exactly that five times.
		{"category not allowed", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes = append(b.Programs[0].Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
				Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
					Kind:       irv1.ChecksumOpKind_CHECKSUM_OP_KIND_UNSUPPORTED,
					ReasonCode: reasonPtr(irv1.ReasonCode_REASON_CODE_UNSUPPORTED_CHECKSUM),
				}},
			})
		}, artifact.ErrInvalid, "is not allowed in a"},
		{"subject in canonicalization", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes = append(b.Programs[0].Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
				}},
			})
		}, artifact.ErrInvalid, "is not allowed in a"},
		{"too few operands", func(b *irv1.RuleBundle) { b.Programs[1].Nodes[1].InputNodes = nil },
			artifact.ErrInvalid, "expects at least"},
		{"too many operands", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes[2].InputNodes = []uint32{1, 1}
		}, artifact.ErrInvalid, "accepts at most"},
		{"missing required parameter", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes[1].GetPredicateOperation().Length = nil
		}, artifact.ErrInvalid, "requires the parameter"},
		{"constant too long", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind: irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND,
					Text: str(strings.Repeat("x", 5000)),
				}},
			}
		}, artifact.ErrInvalid, "exceeds"},
		{"empty constant", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind: irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND,
					Text: str(""),
				}},
			}
		}, artifact.ErrInvalid, "must not be empty"},
		{"replace prefix identity", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind:        irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REPLACE_PREFIX,
					Text:        str("GR"),
					Replacement: str("GR"),
				}},
			}
		}, artifact.ErrInvalid, "by itself"},
		{"left_pad multi code point", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind:   irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD,
					Text:   str("00"),
					Length: u32(9),
				}},
			}
		}, artifact.ErrInvalid, "one padding code point"},
		{"left_pad zero length", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind:   irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD,
					Text:   str("0"),
					Length: u32(0),
				}},
			}
		}, artifact.ErrInvalid, "at least 1"},
		// An engine that compiles the rules ahead of time sizes its buffer from
		// this number, so an unbounded length is an allocation primitive.
		{"left_pad length above the limit", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind:   irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD,
					Text:   str("0"),
					Length: u32(4_097),
				}},
			}
		}, artifact.ErrInvalid, "exceeds the limit"},
		{"empty message key", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes[2].GetAssertionOperation().MessageKey = str("")
		}, artifact.ErrInvalid, "must not be empty"},
		{"index above the limit", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes[1].GetPredicateOperation().Length = u32(9999)
		}, artifact.ErrInvalid, "exceeds the limit"},
		{"length_between order", func(b *irv1.RuleBundle) {
			op := b.Programs[1].Nodes[1].GetPredicateOperation()
			op.Kind = irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_BETWEEN
			op.Length = nil
			op.MinLength = u32(9)
			op.MaxLength = u32(2)
		}, artifact.ErrInvalid, "greater than maximum"},
		{"non ascii charset", func(b *irv1.RuleBundle) {
			op := b.Programs[1].Nodes[1].GetPredicateOperation()
			op.Kind = irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_CHARSET
			op.Length = nil
			op.Text = str("é")
		}, artifact.ErrInvalid, "must be ASCII"},
		{"unknown profile", func(b *irv1.RuleBundle) {
			op := b.Programs[1].Nodes[1].GetPredicateOperation()
			op.Kind = irv1.PredicateOpKind_PREDICATE_OP_KIND_PROFILE_IS
			op.Length = nil
			op.Text = str("loose")
			b.Programs[1].Nodes[1].InputNodes = nil
		}, artifact.ErrInvalid, "unknown profile"},
		{"empty prefix_in value", func(b *irv1.RuleBundle) {
			op := b.Programs[1].Nodes[1].GetPredicateOperation()
			op.Kind = irv1.PredicateOpKind_PREDICATE_OP_KIND_PREFIX_IN
			op.Length = nil
			op.Values = []string{""}
		}, artifact.ErrInvalid, "empty value"},
		{"invalid identifier kind", func(b *irv1.RuleBundle) { b.Identifiers[0].Kind = "Demo" },
			artifact.ErrInvalid, "invalid kind"},
		{"invalid identifier country", func(b *irv1.RuleBundle) { b.Identifiers[0].CountryCode = str("fr") },
			artifact.ErrInvalid, "invalid country"},
		{"zero identifier id", func(b *irv1.RuleBundle) { b.Identifiers[0].Id = 0 },
			artifact.ErrInvalid, "identifier ids start at 1"},
		{"unknown program reference", func(b *irv1.RuleBundle) { b.Identifiers[0].FormatProgram = 42 },
			artifact.ErrInvalid, "unknown format program"},
		{"wrong program kind", func(b *irv1.RuleBundle) { b.Identifiers[0].FormatProgram = 1 },
			artifact.ErrInvalid, "of kind"},
		{"checksum and absence", func(b *irv1.RuleBundle) {
			b.Identifiers[0].AbsentChecksumReason = reasonPtr(irv1.ReasonCode_REASON_CODE_UNSUPPORTED_CHECKSUM)
		}, artifact.ErrInvalid, "both a checksum program and an absence reason"},
		{"neither checksum nor absence", func(b *irv1.RuleBundle) { b.Identifiers[0].ChecksumProgram = nil },
			artifact.ErrInvalid, "neither a checksum program nor an absence reason"},
		{"invalid absence reason", func(b *irv1.RuleBundle) {
			b.Identifiers[0].ChecksumProgram = nil
			b.Identifiers[0].AbsentChecksumReason = reasonPtr(irv1.ReasonCode_REASON_CODE_INVALID_CHECKSUM)
		}, artifact.ErrInvalid, "invalid absent checksum reason"},
		{"invalid default profile", func(b *irv1.RuleBundle) { b.Identifiers[0].DefaultProfile = "loose" },
			artifact.ErrInvalid, "invalid default profile"},
		{"source without id", func(b *irv1.RuleBundle) { b.Identifiers[0].Sources[0].Id = "" },
			artifact.ErrInvalid, "source without id"},
		{"duplicate source", func(b *irv1.RuleBundle) {
			b.Identifiers[0].Sources = append(b.Identifiers[0].Sources, b.Identifiers[0].Sources[0])
		}, artifact.ErrInvalid, "twice"},
		{"unsorted sources", func(b *irv1.RuleBundle) {
			second := proto.Clone(b.Identifiers[0].Sources[0]).(*irv1.Source)
			second.Id = "a-first"
			b.Identifiers[0].Sources = append(b.Identifiers[0].Sources, second)
		}, artifact.ErrInvalid, "sort its sources"},
		{"invalid dispatcher kind", func(b *irv1.RuleBundle) { b.Dispatchers[0].Kind = "Demo" },
			artifact.ErrInvalid, "invalid kind"},
		{"invalid kind alias", func(b *irv1.RuleBundle) { b.Dispatchers[0].KindAliases = []string{"Demo"} },
			artifact.ErrInvalid, "invalid kind alias"},
		{"unsorted kind aliases", func(b *irv1.RuleBundle) { b.Dispatchers[0].KindAliases = []string{"b", "a"} },
			artifact.ErrInvalid, "sort its kind aliases"},
		{"alias shadowing the kind", func(b *irv1.RuleBundle) { b.Dispatchers[0].KindAliases = []string{"demo"} },
			artifact.ErrInvalid, "claimed twice"},
		{"invalid pre-canonicalizer", func(b *irv1.RuleBundle) { b.Dispatchers[0].PreCanonicalizationProgram = 2 },
			artifact.ErrInvalid, "invalid pre-canonicalization program"},
		{"forbidden pre-canonicalization step", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind: irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND,
					Text: str("X"),
				}},
			}
		}, artifact.ErrInvalid, "in its pre-canonicalization program"},
		{"no target", func(b *irv1.RuleBundle) { b.Dispatchers[0].Targets = nil },
			artifact.ErrInvalid, "declares no target"},
		{"invalid country alias", func(b *irv1.RuleBundle) {
			b.Dispatchers[0].CountryAliases = []*irv1.CountryAlias{{Alias: "abc", CountryCode: "FR"}}
		}, artifact.ErrInvalid, "invalid country alias"},
		{"self country alias", func(b *irv1.RuleBundle) {
			b.Dispatchers[0].CountryAliases = []*irv1.CountryAlias{{Alias: "FR", CountryCode: "FR"}}
		}, artifact.ErrInvalid, "to itself"},
		{"duplicate country alias", func(b *irv1.RuleBundle) {
			b.Dispatchers[0].CountryAliases = []*irv1.CountryAlias{
				{Alias: "UK", CountryCode: "GB"}, {Alias: "UK", CountryCode: "IE"},
			}
		}, artifact.ErrInvalid, "twice"},
		{"unsorted country aliases", func(b *irv1.RuleBundle) {
			b.Dispatchers[0].CountryAliases = []*irv1.CountryAlias{
				{Alias: "UK", CountryCode: "GB"}, {Alias: "AB", CountryCode: "IE"},
			}
		}, artifact.ErrInvalid, "sort its country aliases"},
		{"unknown definition in a target", func(b *irv1.RuleBundle) {
			b.Dispatchers[0].Targets[0].IdentifierDefinitionId = 42
		}, artifact.ErrInvalid, "unknown definition"},
		{"invalid prefix", func(b *irv1.RuleBundle) {
			b.Identifiers[0].CountryCode = str("BE")
			b.Dispatchers[0].Targets[0].CountryCode = str("BE")
			b.Dispatchers[0].Targets[0].AcceptedPrefixes = []string{"B-E"}
		}, artifact.ErrInvalid, "invalid prefix"},
		{"unsorted prefixes", func(b *irv1.RuleBundle) {
			b.Identifiers[0].CountryCode = str("BE")
			b.Dispatchers[0].Targets[0].CountryCode = str("BE")
			b.Dispatchers[0].Targets[0].AcceptedPrefixes = []string{"BZ", "BE"}
		}, artifact.ErrInvalid, "sort the prefixes"},
		{"canonical prefix outside", func(b *irv1.RuleBundle) {
			b.Identifiers[0].CountryCode = str("BE")
			b.Dispatchers[0].Targets[0].CountryCode = str("BE")
			b.Dispatchers[0].Targets[0].AcceptedPrefixes = []string{"BE"}
			b.Dispatchers[0].Targets[0].CanonicalPrefix = str("BX")
		}, artifact.ErrInvalid, "canonical prefix outside"},
		{"country target on a global definition", func(b *irv1.RuleBundle) {
			b.Dispatchers[0].Targets[0].CountryCode = str("BE")
		}, artifact.ErrInvalid, "references the definition"},
		{"global target on a country definition", func(b *irv1.RuleBundle) {
			b.Identifiers[0].CountryCode = str("BE")
		}, artifact.ErrInvalid, "references a country definition"},
		{"invalid target country", func(b *irv1.RuleBundle) {
			b.Identifiers[0].CountryCode = str("BE")
			b.Dispatchers[0].Targets[0].CountryCode = str("be")
		}, artifact.ErrInvalid, "invalid target country"},
		{"unspecified value type", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes[0].OutputType = irv1.ValueType_VALUE_TYPE_UNSPECIFIED
		}, artifact.ErrInvalid, "declares output"},
		{"unknown weighted alignment", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes[1] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
					Kind:      irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM,
					Weights:   []int64{1, 2},
					Alignment: alignPtr(irv1.WeightAlignment_WEIGHT_ALIGNMENT_UNSPECIFIED),
					Mapping:   mappingPtr(irv1.CharMapping_CHAR_MAPPING_DIGIT_VALUE),
				}},
			}
		}, artifact.ErrInvalid, "unspecified alignment"},
		{"unknown weighted mapping", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes[1] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
					Kind:      irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM,
					Weights:   []int64{1, 2},
					Alignment: alignPtr(irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT),
					Mapping:   mappingPtr(irv1.CharMapping_CHAR_MAPPING_UNSPECIFIED),
				}},
			}
		}, artifact.ErrInvalid, "unspecified mapping"},
		{"weight magnitude", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes[1] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
					Kind:      irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM,
					Weights:   []int64{9_000_000},
					Alignment: alignPtr(irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT),
					Mapping:   mappingPtr(irv1.CharMapping_CHAR_MAPPING_DIGIT_VALUE),
				}},
			}
		}, artifact.ErrInvalid, "magnitude limit"},
		{"remainder table size", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes[1] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
					Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS, Modulus: i64(97),
				}},
			}
			b.Programs[2].Nodes = append(b.Programs[2].Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
				InputNodes: []uint32{1},
				Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
					Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_REMAINDER_MAP,
				}},
			})
			b.Programs[2].RootNode = 2
		}, artifact.ErrInvalid, "requires the parameter"},
		{"compare_slice too wide", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes[1] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
					Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS, Modulus: i64(97),
				}},
			}
			b.Programs[2].Nodes = append(b.Programs[2].Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
				InputNodes: []uint32{1, 0},
				Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
					Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_SLICE, Start: u32(0), End: u32(30),
				}},
			})
			b.Programs[2].RootNode = 2
		}, artifact.ErrInvalid, "more than 18 digits"},
		{"compare_slice empty", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes[1] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
					Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS, Modulus: i64(97),
				}},
			}
			b.Programs[2].Nodes = append(b.Programs[2].Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
				InputNodes: []uint32{1, 0},
				Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
					Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_SLICE, Start: u32(2), End: u32(2),
				}},
			})
			b.Programs[2].RootNode = 2
		}, artifact.ErrInvalid, "lower than end"},
		{"slice bounds", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				InputNodes: nil,
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_VALUE,
				}},
			}
			b.Programs[1].Nodes = append(b.Programs[1].Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_SLICE, Start: u32(9), End: u32(2),
				}},
			})
		}, artifact.ErrInvalid, "greater than end"},
		{"call kind mismatch", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes = append(b.Programs[2].Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_CallOperation{CallOperation: &irv1.CallOperation{
					Kind: irv1.CallOpKind_CALL_OP_KIND_CHECKSUM, ProgramId: 2,
				}},
			})
			b.Programs[2].RootNode = 2
		}, artifact.ErrInvalid, "calls program"},
		{"call to program zero", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes = append(b.Programs[2].Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_CallOperation{CallOperation: &irv1.CallOperation{
					Kind: irv1.CallOpKind_CALL_OP_KIND_CHECKSUM,
				}},
			})
			b.Programs[2].RootNode = 2
		}, artifact.ErrInvalid, "references program 0"},
		{"unsorted identifiers", func(b *irv1.RuleBundle) {
			b.Identifiers[0].CountryCode = str("FR")
			second := proto.Clone(b.Identifiers[0]).(*irv1.IdentifierDefinition)
			second.Id = 2
			second.CountryCode = str("BE")
			b.Identifiers = append(b.Identifiers, second)
			b.Dispatchers[0].Targets = []*irv1.DispatchTarget{
				{CountryCode: str("FR"), IdentifierDefinitionId: 1},
				{CountryCode: str("BE"), IdentifierDefinitionId: 2},
			}
		}, artifact.ErrInvalid, "normative serialization order"},
		{"unsorted dispatchers", func(b *irv1.RuleBundle) {
			second := proto.Clone(b.Dispatchers[0]).(*irv1.IdentifierDispatcher)
			second.Kind = "alpha"
			b.Dispatchers = append(b.Dispatchers, second)
		}, artifact.ErrInvalid, "not sorted by kind"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bundle := baseBundle(t)
			tc.mutate(bundle)
			expectRejection(t, bundle, tc.kind, tc.substring)
		})
	}
}

func reasonPtr(r irv1.ReasonCode) *irv1.ReasonCode          { return &r }
func mappingPtr(m irv1.CharMapping) *irv1.CharMapping       { return &m }
func alignPtr(a irv1.WeightAlignment) *irv1.WeightAlignment { return &a }

func TestCapabilityNamesAppearInRejection(t *testing.T) {
	bundle := baseBundle(t)
	bundle.RequiredFeatureIds = []uint32{features.CoreGraphV1}
	expectRejection(t, bundle, artifact.ErrInvalid, "without declaring it")
}

// TestBoundedParametersAreChecked walks every bounded parameter of every
// operation family: an index above the limit or an over long message key is
// always refused, whichever operation carries it.
func TestBoundedParametersAreChecked(t *testing.T) {
	const over = uint32(9999)
	longKey := strings.Repeat("k", 5000)
	tests := []struct {
		name      string
		mutate    func(b *irv1.RuleBundle)
		substring string
	}{
		{"string start", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes = append(b.Programs[1].Nodes, sliceNode(over, over))
		}, "index 9999 exceeds"},
		{"string end", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes = append(b.Programs[1].Nodes, sliceNode(0, over))
		}, "index 9999 exceeds"},
		{"predicate index", func(b *irv1.RuleBundle) {
			op := b.Programs[1].Nodes[1].GetPredicateOperation()
			op.Kind = irv1.PredicateOpKind_PREDICATE_OP_KIND_CHAR_AT_IN
			op.Length = nil
			op.Index = u32(over)
			op.Text = str("0123456789")
		}, "index 9999 exceeds"},
		{"predicate min length", func(b *irv1.RuleBundle) {
			op := b.Programs[1].Nodes[1].GetPredicateOperation()
			op.Kind = irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_BETWEEN
			op.Length = nil
			op.MinLength = u32(over)
			op.MaxLength = u32(over)
		}, "index 9999 exceeds"},
		{"predicate lengths entry", func(b *irv1.RuleBundle) {
			op := b.Programs[1].Nodes[1].GetPredicateOperation()
			op.Kind = irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_IN
			op.Length = nil
			op.Lengths = []uint32{over}
		}, "length 9999 exceeds"},
		{"canonicalization index", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind:  irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_INSERT,
					Index: u32(over), Text: str("0"),
				}},
			}
		}, "index 9999 exceeds"},
		{"canonicalization length", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind:   irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD,
					Length: u32(over), Text: str("0"),
				}},
			}
		}, "index 9999 exceeds"},
		{"checksum index", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes[1] = modDigitsNode()
			b.Programs[2].Nodes = append(b.Programs[2].Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
				InputNodes: []uint32{1, 0},
				Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
					Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT, Index: u32(over),
				}},
			})
			b.Programs[2].RootNode = 2
		}, "index 9999 exceeds"},
		{"checksum start", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes[1] = modDigitsNode()
			b.Programs[2].Nodes = append(b.Programs[2].Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
				InputNodes: []uint32{1, 0},
				Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
					Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_SLICE, Start: u32(over), End: u32(over),
				}},
			})
			b.Programs[2].RootNode = 2
		}, "index 9999 exceeds"},
		{"assertion message key", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes[2].GetAssertionOperation().MessageKey = str(longKey)
		}, "message key exceeds"},
		{"checksum message key", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes[1].GetChecksumOperation().MessageKey = str(longKey)
		}, "message key exceeds"},
		{"prefix_in value too long", func(b *irv1.RuleBundle) {
			op := b.Programs[1].Nodes[1].GetPredicateOperation()
			op.Kind = irv1.PredicateOpKind_PREDICATE_OP_KIND_PREFIX_IN
			op.Length = nil
			op.Values = []string{strings.Repeat("x", 5000)}
		}, "constant exceeds"},
		{"canonicalization constant too long", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
				Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind: irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REPLACE_PREFIX,
					Text: str("GR"), Replacement: str(strings.Repeat("x", 5000)),
				}},
			}
		}, "constant exceeds"},
		{"string constant too long", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes[0] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_CONSTANT, Text: str(strings.Repeat("x", 5000)),
				}},
			}
		}, "constant exceeds"},
		{"empty view delimiter", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes = append(b.Programs[1].Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, Text: str(""),
				}},
			})
		}, "must not be empty"},
		{"predicate constant too long", func(b *irv1.RuleBundle) {
			op := b.Programs[1].Nodes[1].GetPredicateOperation()
			op.Kind = irv1.PredicateOpKind_PREDICATE_OP_KIND_STARTS_WITH
			op.Length = nil
			op.Text = str(strings.Repeat("x", 5000))
		}, "constant exceeds"},
		{"too many weights", func(b *irv1.RuleBundle) {
			weights := make([]int64, 300)
			for i := range weights {
				weights[i] = 1
			}
			b.Programs[2].Nodes[1] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
					Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM, Weights: weights,
					Alignment: alignPtr(irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT),
					Mapping:   mappingPtr(irv1.CharMapping_CHAR_MAPPING_DIGIT_VALUE),
				}},
			}
		}, "weights exceed the limit"},
		{"cycling weighted sum without a bound", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes[1] = &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
					Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM, Weights: []int64{1, 2},
					Alignment: alignPtr(irv1.WeightAlignment_WEIGHT_ALIGNMENT_CYCLE),
					Mapping:   mappingPtr(irv1.CharMapping_CHAR_MAPPING_DIGIT_VALUE),
				}},
			}
		}, "statically bounded operand"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bundle := baseBundle(t)
			tc.mutate(bundle)
			expectRejection(t, bundle, artifact.ErrInvalid, tc.substring)
		})
	}
}

func sliceNode(start, end uint32) *irv1.Node {
	return &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_STRING,
		InputNodes: []uint32{0},
		Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
			Kind: irv1.StringOpKind_STRING_OP_KIND_SLICE, Start: u32(start), End: u32(end),
		}},
	}
}

func modDigitsNode() *irv1.Node {
	return &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
		InputNodes: []uint32{0},
		Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
			Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS, Modulus: i64(97),
		}},
	}
}
