package artifact

import (
	"unicode/utf8"

	"google.golang.org/protobuf/proto"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/features"
	"github.com/libbusinessid/spec/internal/limits"
)

// SupportedFormatVersion is the only IR structural version this runtime loads.
const SupportedFormatVersion uint32 = 1

// unknownLength marks a node whose maximum length is not statically provable.
const unknownLength = -1

// LoadRuleset decodes and defensively validates a rule bundle, in the exact
// order mandated by the engine contract.
func LoadRuleset(raw []byte) (*Ruleset, error) {
	if len(raw) > limits.MaxBundleBytes {
		return nil, invalidf("bundle of %d bytes exceeds the limit of %d", len(raw), limits.MaxBundleBytes)
	}
	bundle := &irv1.RuleBundle{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, bundle); err != nil {
		return nil, invalidf("protobuf decoding failed: %v", err)
	}
	// The version checks come before the unknown field scan, and the order
	// matters. A bundle built against a later version carries fields this
	// runtime has never heard of, and reporting those as unknown fields calls a
	// legitimate version gap a forged bundle. Asking first whether the bundle
	// announces something unsupported gives the accurate answer:
	// incompatible_ruleset, which tells the operator to upgrade rather than to
	// suspect the file.
	//
	// It stays safe: the message is fully decoded by now, and both checks read
	// declared scalars rather than following anything the bundle controls.
	if bundle.GetFormatVersion() != SupportedFormatVersion {
		return nil, incompatiblef("unsupported format_version %d", bundle.GetFormatVersion())
	}
	if err := validateFeatureIDs(bundle.GetRequiredFeatureIds()); err != nil {
		return nil, err
	}
	if path, bad := findUnknownFields(bundle); bad {
		return nil, invalidf("unknown Protobuf field at %s", path)
	}
	if err := validateRulesVersion(bundle.GetRulesVersion()); err != nil {
		return nil, err
	}
	if len(bundle.GetSourceDigest()) != 32 {
		return nil, invalidf("source_digest must hold exactly 32 bytes, got %d", len(bundle.GetSourceDigest()))
	}

	v := &validator{bundle: bundle, used: features.NewSet()}
	if err := v.validatePrograms(); err != nil {
		return nil, err
	}
	if err := v.validateIdentifiers(); err != nil {
		return nil, err
	}
	if err := v.validateDispatchers(); err != nil {
		return nil, err
	}
	if err := v.validateCallGraph(); err != nil {
		return nil, err
	}
	if err := v.validateDeclaredFeatures(); err != nil {
		return nil, err
	}
	return &Ruleset{
		Bundle:           bundle,
		ProgramByID:      v.programByID,
		DefinitionByID:   v.definitionByID,
		DispatcherByKind: v.dispatcherByKind,
	}, nil
}

func validateFeatureIDs(ids []uint32) error {
	for i, id := range ids {
		if i > 0 && ids[i-1] >= id {
			return invalidf("required_feature_ids must be strictly ascending")
		}
		if !features.Known(id) {
			return incompatiblef("unknown capability id %d", id)
		}
	}
	return nil
}

// validateRulesVersion bounds the shape of the version, not only its emptiness.
//
// The value travels far beyond the loader: engines put it in generated sources,
// in manifests and in logs. The Go engine found by fuzzing that a version
// holding a NUL produced source code that would not compile - accepted by the
// loader, fatal three steps later. Restricting it to the characters a version
// actually uses removes that class of divergence for every engine at once.
func validateRulesVersion(v string) error {
	if v == "" {
		return invalidf("rules_version must not be empty")
	}
	if !utf8.ValidString(v) {
		return invalidf("rules_version is not valid UTF-8")
	}
	if len(v) > limits.MaxRulesVersionBytes {
		return invalidf("rules_version exceeds %d bytes", limits.MaxRulesVersionBytes)
	}
	for _, r := range v {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case r == '.' || r == '-' || r == '_':
		default:
			return invalidf("rules_version holds %q, outside the accepted set of "+
				"ASCII letters, digits, dot, dash and underscore", r)
		}
	}
	return nil
}

type validator struct {
	bundle           *irv1.RuleBundle
	programByID      map[uint32]*irv1.Program
	definitionByID   map[uint32]*irv1.IdentifierDefinition
	dispatcherByKind map[string]*irv1.IdentifierDispatcher
	used             *features.Set
	// prependCountryPrograms records the canonicalization programs using the
	// country aware step, so definition level restrictions can be applied.
	prependCountryPrograms map[uint32]bool
	// programOps records the operation kinds of each canonicalization program.
	canonicalizationKinds map[uint32][]irv1.CanonicalizationOpKind
	// calls records the call edges of each program.
	calls map[uint32][]uint32
}

func (v *validator) validatePrograms() error {
	v.programByID = make(map[uint32]*irv1.Program, len(v.bundle.GetPrograms()))
	v.prependCountryPrograms = map[uint32]bool{}
	v.canonicalizationKinds = map[uint32][]irv1.CanonicalizationOpKind{}
	v.calls = map[uint32][]uint32{}

	total := 0
	for _, p := range v.bundle.GetPrograms() {
		if p.GetId() == 0 {
			return invalidf("program ids start at 1")
		}
		if _, dup := v.programByID[p.GetId()]; dup {
			return invalidf("duplicate program id %d", p.GetId())
		}
		v.programByID[p.GetId()] = p
		if p.GetKind() == irv1.ProgramKind_PROGRAM_KIND_UNSPECIFIED {
			return invalidf("program %d has an unspecified kind", p.GetId())
		}
		nodes := p.GetNodes()
		if len(nodes) == 0 {
			return invalidf("program %d holds no node", p.GetId())
		}
		if len(nodes) > limits.MaxNodesPerProgram {
			return invalidf("program %d holds %d nodes, the limit is %d",
				p.GetId(), len(nodes), limits.MaxNodesPerProgram)
		}
		total += len(nodes)
		if total > limits.MaxTotalNodes {
			return invalidf("the bundle holds more than %d nodes", limits.MaxTotalNodes)
		}
		maxLen := make([]int, len(nodes))
		for i, n := range nodes {
			if err := v.validateNode(p, i, n, maxLen); err != nil {
				return err
			}
		}
		if int(p.GetRootNode()) >= len(nodes) {
			return invalidf("program %d has an out of range root node", p.GetId())
		}
		// A generator inlining repeated operands emits one instance per path,
		// not one per node, and a bounded node count does not bound that.
		if err := checkExpansion(p); err != nil {
			return err
		}
		if err := v.validateProgramShape(p, nodes); err != nil {
			return err
		}
	}
	for _, p := range v.bundle.GetPrograms() {
		for _, id := range v.calls[p.GetId()] {
			if _, ok := v.programByID[id]; !ok {
				return invalidf("program %d calls the unknown program %d", p.GetId(), id)
			}
		}
	}
	return nil
}

func (v *validator) validateProgramShape(p *irv1.Program, nodes []*irv1.Node) error {
	if err := v.validateProgramRoot(p, nodes); err != nil {
		return err
	}
	if p.SubjectNode != nil {
		if int(p.GetSubjectNode()) >= len(nodes) {
			return invalidf("program %d has an out of range subject node", p.GetId())
		}
		if nodes[p.GetSubjectNode()].GetOutputType() != irv1.ValueType_VALUE_TYPE_STRING {
			return invalidf("program %d has a non string subject node", p.GetId())
		}
		if err := checkSubjectNode(p); err != nil {
			return err
		}
	}
	return validateCaptures(p, nodes)
}

// validateProgramRoot checks the root shape imposed by the program kind.
func (v *validator) validateProgramRoot(p *irv1.Program, nodes []*irv1.Node) error {
	root := nodes[p.GetRootNode()]
	switch p.GetKind() {
	case irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION:
		op := root.GetCanonicalizationOperation()
		if op == nil || op.GetKind() != irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_SEQUENCE {
			return invalidf("canonicalization program %d must be rooted in a SEQUENCE", p.GetId())
		}
		if p.SubjectNode != nil {
			return invalidf("canonicalization program %d must not declare a subject", p.GetId())
		}
		if len(p.GetCaptures()) > 0 {
			return invalidf("canonicalization program %d must not declare captures", p.GetId())
		}
	case irv1.ProgramKind_PROGRAM_KIND_FORMAT:
		op := root.GetAssertionOperation()
		if op == nil || op.GetKind() != irv1.AssertionOpKind_ASSERTION_OP_KIND_SEQUENCE {
			return invalidf("format program %d must be rooted in an assertion SEQUENCE", p.GetId())
		}
	case irv1.ProgramKind_PROGRAM_KIND_CHECKSUM:
		if root.GetOutputType() != irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME {
			return invalidf("checksum program %d must produce a checksum outcome", p.GetId())
		}
		if op := root.GetChecksumOperation(); op != nil &&
			op.GetKind() == irv1.ChecksumOpKind_CHECKSUM_OP_KIND_WHEN {
			return invalidf("checksum program %d must not be rooted in a WHEN branch", p.GetId())
		}
		if len(p.GetCaptures()) > 0 {
			return invalidf("checksum program %d must not declare captures", p.GetId())
		}
	default:
		return invalidf("program %d has an unsupported kind", p.GetId())
	}
	return nil
}

// validateCaptures checks the named views of a format program.
func validateCaptures(p *irv1.Program, nodes []*irv1.Node) error {
	if len(p.GetCaptures()) > limits.MaxCapturesPerFormat {
		return invalidf("program %d declares %d captures, the limit is %d",
			p.GetId(), len(p.GetCaptures()), limits.MaxCapturesPerFormat)
	}
	seen := map[string]bool{}
	for _, c := range p.GetCaptures() {
		if c.GetName() == "" {
			return invalidf("program %d declares an unnamed capture", p.GetId())
		}
		if seen[c.GetName()] {
			return invalidf("program %d declares the capture %q twice", p.GetId(), c.GetName())
		}
		seen[c.GetName()] = true
		if int(c.GetNode()) >= len(nodes) {
			return invalidf("program %d has an out of range capture node", p.GetId())
		}
		if nodes[c.GetNode()].GetOutputType() != irv1.ValueType_VALUE_TYPE_STRING {
			return invalidf("program %d has a non string capture %q", p.GetId(), c.GetName())
		}
	}
	return nil
}
