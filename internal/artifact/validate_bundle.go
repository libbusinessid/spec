package artifact

import (
	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/features"
	"github.com/entid-org/spec/internal/limits"
)

//nolint:gocyclo // one exhaustive validation of the definition table.
func (v *validator) validateIdentifiers() error {
	defs := v.bundle.GetIdentifiers()
	if len(defs) > limits.MaxIdentifiers {
		return invalidf("the bundle holds %d identifiers, the limit is %d", len(defs), limits.MaxIdentifiers)
	}
	v.definitionByID = make(map[uint32]*irv1.IdentifierDefinition, len(defs))
	seenKey := map[string]bool{}
	for i, d := range defs {
		if d.GetId() == 0 {
			return invalidf("identifier ids start at 1")
		}
		if _, dup := v.definitionByID[d.GetId()]; dup {
			return invalidf("duplicate identifier id %d", d.GetId())
		}
		v.definitionByID[d.GetId()] = d
		if !validKind(d.GetKind()) {
			return invalidf("identifier %d has an invalid kind %q", d.GetId(), d.GetKind())
		}
		if d.CountryCode != nil && !validCountry(d.GetCountryCode()) {
			return invalidf("identifier %d has an invalid country %q", d.GetId(), d.GetCountryCode())
		}
		key := d.GetKind() + "/" + globalOr(d.CountryCode)
		if seenKey[key] {
			return invalidf("two definitions share the sort key %s", key)
		}
		seenKey[key] = true
		if i > 0 && !definitionOrderBefore(defs[i-1], d) {
			return invalidf("identifiers are not in the normative serialization order")
		}
		if err := v.checkProgram(d.GetCanonicalizationProgram(),
			irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION, "canonicalization", d.GetId()); err != nil {
			return err
		}
		if err := v.checkProgram(d.GetFormatProgram(),
			irv1.ProgramKind_PROGRAM_KIND_FORMAT, "format", d.GetId()); err != nil {
			return err
		}
		switch {
		case d.ChecksumProgram != nil:
			if err := v.checkProgram(d.GetChecksumProgram(),
				irv1.ProgramKind_PROGRAM_KIND_CHECKSUM, "checksum", d.GetId()); err != nil {
				return err
			}
			if d.AbsentChecksumReason != nil {
				return invalidf("identifier %d declares both a checksum program and an absence reason", d.GetId())
			}
		case d.AbsentChecksumReason == nil:
			return invalidf("identifier %d declares neither a checksum program nor an absence reason", d.GetId())
		default:
			if !unsupportedReasonAllowed(d.GetAbsentChecksumReason()) {
				return invalidf("identifier %d has an invalid absent checksum reason %v",
					d.GetId(), d.GetAbsentChecksumReason())
			}
		}
		if d.GetDefaultProfile() != "compatible" && d.GetDefaultProfile() != "strict_current" {
			return invalidf("identifier %d has an invalid default profile %q", d.GetId(), d.GetDefaultProfile())
		}
		if d.CountryCode == nil && v.prependCountryPrograms[d.GetCanonicalizationProgram()] {
			return invalidf("the GLOBAL identifier %d uses a country aware canonicalization step", d.GetId())
		}
		if err := validateSources(d); err != nil {
			return err
		}
	}
	return nil
}

func validateSources(d *irv1.IdentifierDefinition) error {
	seen := map[string]bool{}
	for i, s := range d.GetSources() {
		if s.GetId() == "" {
			return invalidf("identifier %d holds a source without id", d.GetId())
		}
		if seen[s.GetId()] {
			return invalidf("identifier %d holds the source %q twice", d.GetId(), s.GetId())
		}
		seen[s.GetId()] = true
		if i > 0 && d.GetSources()[i-1].GetId() > s.GetId() {
			return invalidf("identifier %d does not sort its sources by id", d.GetId())
		}
	}
	return nil
}

func (v *validator) checkProgram(id uint32, kind irv1.ProgramKind, label string, defID uint32) error {
	p, ok := v.programByID[id]
	if !ok {
		return invalidf("identifier %d references the unknown %s program %d", defID, label, id)
	}
	if p.GetKind() != kind {
		return invalidf("identifier %d references the %s program %d of kind %v", defID, label, id, p.GetKind())
	}
	return nil
}

func (v *validator) validateDispatchers() error {
	v.dispatcherByKind = map[string]*irv1.IdentifierDispatcher{}
	claimed := map[uint32]bool{}
	tokens := map[string]bool{}
	for i, d := range v.bundle.GetDispatchers() {
		if !validKind(d.GetKind()) {
			return invalidf("dispatcher %q has an invalid kind", d.GetKind())
		}
		if i > 0 && v.bundle.GetDispatchers()[i-1].GetKind() >= d.GetKind() {
			return invalidf("dispatchers are not sorted by kind, or share a kind")
		}
		if tokens[d.GetKind()] {
			return invalidf("the kind token %q is claimed twice", d.GetKind())
		}
		tokens[d.GetKind()] = true
		v.dispatcherByKind[d.GetKind()] = d
		for j, alias := range d.GetKindAliases() {
			if !validKindAlias(alias) {
				return invalidf("dispatcher %q has an invalid kind alias %q", d.GetKind(), alias)
			}
			if j > 0 && d.GetKindAliases()[j-1] >= alias {
				return invalidf("dispatcher %q does not sort its kind aliases", d.GetKind())
			}
			if tokens[alias] {
				return invalidf("the kind token %q is claimed twice", alias)
			}
			tokens[alias] = true
			v.dispatcherByKind[alias] = d
		}
		pre, ok := v.programByID[d.GetPreCanonicalizationProgram()]
		if !ok || pre.GetKind() != irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION {
			return invalidf("dispatcher %q references an invalid pre-canonicalization program", d.GetKind())
		}
		for _, kind := range v.canonicalizationKinds[pre.GetId()] {
			if !preCanonicalizationAllowed(kind) {
				return invalidf("dispatcher %q uses %v in its pre-canonicalization program", d.GetKind(), kind)
			}
		}
		if err := v.validateTargets(d, claimed); err != nil {
			return err
		}
	}
	for _, d := range v.bundle.GetIdentifiers() {
		if !claimed[d.GetId()] {
			return invalidf("identifier %d is not referenced by any dispatch target", d.GetId())
		}
	}
	return nil
}

func (v *validator) validateTargets(d *irv1.IdentifierDispatcher, claimed map[uint32]bool) error {
	if len(d.GetTargets()) == 0 {
		return invalidf("dispatcher %q declares no target", d.GetKind())
	}
	if err := v.validateCountryAliases(d); err != nil {
		return err
	}

	global := false
	implicit := 0
	prefixes := map[string]bool{}
	seenCountry := map[string]bool{}
	for i, t := range d.GetTargets() {
		if i > 0 && !targetOrderBefore(d.GetTargets()[i-1], t) {
			return invalidf("dispatcher %q does not sort its targets", d.GetKind())
		}
		if err := v.validateTarget(d, t, claimed, seenCountry, prefixes); err != nil {
			return err
		}
		if t.CountryCode == nil {
			global = true
		}
		if t.GetAllowUnprefixedWithoutCountry() {
			implicit++
		}
	}
	if global && len(d.GetTargets()) != 1 {
		return invalidf("dispatcher %q mixes a GLOBAL target with country targets", d.GetKind())
	}
	if global && len(d.GetCountryAliases()) > 0 {
		return invalidf("the GLOBAL dispatcher %q declares country aliases", d.GetKind())
	}
	if implicit > 1 {
		return invalidf("dispatcher %q declares %d implicit targets", d.GetKind(), implicit)
	}
	return nil
}

// validateCountryAliases checks the alias table of a dispatcher.
func (v *validator) validateCountryAliases(d *irv1.IdentifierDispatcher) error {
	countries := map[string]bool{}
	for _, t := range d.GetTargets() {
		if t.CountryCode != nil {
			countries[t.GetCountryCode()] = true
		}
	}
	seen := map[string]bool{}
	for i, ca := range d.GetCountryAliases() {
		if !validCountry(ca.GetAlias()) || !validCountry(ca.GetCountryCode()) {
			return invalidf("dispatcher %q has an invalid country alias", d.GetKind())
		}
		if ca.GetAlias() == ca.GetCountryCode() {
			return invalidf("dispatcher %q maps the country alias %q to itself", d.GetKind(), ca.GetAlias())
		}
		if seen[ca.GetAlias()] {
			return invalidf("dispatcher %q declares the country alias %q twice", d.GetKind(), ca.GetAlias())
		}
		if countries[ca.GetAlias()] {
			return invalidf("dispatcher %q has a country alias %q shadowing a target", d.GetKind(), ca.GetAlias())
		}
		seen[ca.GetAlias()] = true
		if i > 0 && d.GetCountryAliases()[i-1].GetAlias() >= ca.GetAlias() {
			return invalidf("dispatcher %q does not sort its country aliases", d.GetKind())
		}
	}
	return nil
}

// validateTarget checks one routing entry against the definition table.
func (v *validator) validateTarget(d *irv1.IdentifierDispatcher, t *irv1.DispatchTarget,
	claimed map[uint32]bool, seenCountry, prefixes map[string]bool,
) error {
	def, ok := v.definitionByID[t.GetIdentifierDefinitionId()]
	if !ok {
		return invalidf("dispatcher %q references the unknown definition %d",
			d.GetKind(), t.GetIdentifierDefinitionId())
	}
	if claimed[def.GetId()] {
		return invalidf("definition %d is referenced by two dispatch targets", def.GetId())
	}
	claimed[def.GetId()] = true
	if def.GetKind() != d.GetKind() {
		return invalidf("dispatcher %q references the definition %d of kind %q",
			d.GetKind(), def.GetId(), def.GetKind())
	}
	switch t.CountryCode {
	case nil:
		if def.CountryCode != nil {
			return invalidf("the GLOBAL target of %q references a country definition", d.GetKind())
		}
		if len(t.GetAcceptedPrefixes()) > 0 || t.CanonicalPrefix != nil {
			return invalidf("the GLOBAL target of %q declares prefixes", d.GetKind())
		}
		if t.GetAllowUnprefixedWithoutCountry() {
			return invalidf("the GLOBAL target of %q declares allow_unprefixed_without_country", d.GetKind())
		}
	default:
		if !validCountry(t.GetCountryCode()) {
			return invalidf("dispatcher %q has an invalid target country %q", d.GetKind(), t.GetCountryCode())
		}
		if seenCountry[t.GetCountryCode()] {
			return invalidf("dispatcher %q declares the country %q twice", d.GetKind(), t.GetCountryCode())
		}
		seenCountry[t.GetCountryCode()] = true
		if def.CountryCode == nil || def.GetCountryCode() != t.GetCountryCode() {
			return invalidf("dispatcher %q target %q references the definition %d",
				d.GetKind(), t.GetCountryCode(), def.GetId())
		}
	}
	for j, p := range t.GetAcceptedPrefixes() {
		if !validPrefix(p) {
			return invalidf("dispatcher %q declares the invalid prefix %q", d.GetKind(), p)
		}
		if j > 0 && t.GetAcceptedPrefixes()[j-1] >= p {
			return invalidf("dispatcher %q does not sort the prefixes of a target", d.GetKind())
		}
		if prefixes[p] {
			return invalidf("dispatcher %q maps the prefix %q to two targets", d.GetKind(), p)
		}
		prefixes[p] = true
	}
	if t.CanonicalPrefix != nil && !containsString(t.GetAcceptedPrefixes(), t.GetCanonicalPrefix()) {
		return invalidf("dispatcher %q has a canonical prefix outside its accepted prefixes", d.GetKind())
	}
	return nil
}

// validateCallGraph proves the call graph is acyclic, typed and shallow.
func (v *validator) validateCallGraph() error {
	state := map[uint32]int{}
	depth := map[uint32]int{}
	ids := make([]uint32, 0, len(v.programByID))
	for _, p := range v.bundle.GetPrograms() {
		ids = append(ids, p.GetId())
	}
	var visit func(id uint32) (int, error)
	visit = func(id uint32) (int, error) {
		switch state[id] {
		case 1:
			return 0, invalidf("the call graph holds a cycle through program %d", id)
		case 2:
			return depth[id], nil
		}
		state[id] = 1
		best := 1
		caller := v.programByID[id]
		for _, callee := range v.calls[id] {
			target, ok := v.programByID[callee]
			if !ok {
				return 0, invalidf("program %d calls the unknown program %d", id, callee)
			}
			if err := checkCallTypes(caller, target); err != nil {
				return 0, err
			}
			d, err := visit(callee)
			if err != nil {
				return 0, err
			}
			if d+1 > best {
				best = d + 1
			}
		}
		state[id] = 2
		depth[id] = best
		if best > limits.MaxCallDepth {
			return 0, invalidf("program %d has a static call depth of %d, the limit is %d",
				id, best, limits.MaxCallDepth)
		}
		return best, nil
	}
	for _, id := range ids {
		if _, err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func checkCallTypes(caller, callee *irv1.Program) error {
	if caller.GetKind() != callee.GetKind() {
		return invalidf("program %d of kind %v calls program %d of kind %v",
			caller.GetId(), caller.GetKind(), callee.GetId(), callee.GetKind())
	}
	return nil
}

// validateDeclaredFeatures rejects a bundle using a capability it does not
// declare.
func (v *validator) validateDeclaredFeatures() error {
	if len(v.bundle.GetDispatchers()) > 0 {
		v.used.Add(features.IdentifierDispatchV1)
	}
	if len(v.bundle.GetIdentifiers()) > 0 {
		v.used.Add(features.CoreGraphV1, features.ProfilesV1)
	}
	for _, d := range v.bundle.GetIdentifiers() {
		if len(d.GetSources()) > 0 {
			v.used.Add(features.ProvenanceV1)
			for _, src := range d.GetSources() {
				// The tier is a closed set. A value outside it would leave two
				// engines free to read the same source differently, and the
				// capability that carries it is what announces a new one.
				if _, ok := irv1.SourceTier_name[int32(src.GetTier())]; !ok {
					return invalidf("identifier %d source %q declares an unknown tier",
						d.GetId(), src.GetId())
				}
				// tier ships under its own capability, PROVENANCE_V1 being
				// frozen, and the point of a separate id is that the two stay
				// independent. tier is not optional in the schema, so an absent
				// field and an explicit UNSPECIFIED are the same bytes:
				// refusing UNSPECIFIED would make 41 mandatory the moment 40
				// is, which is the opposite of independent. UNSPECIFIED means
				// the source states no tier, and only a stated one needs the
				// capability.
				if src.GetTier() != irv1.SourceTier_SOURCE_TIER_UNSPECIFIED {
					v.used.Add(features.ProvenanceTierV1)
				}
			}
		}
		if d.AbsentChecksumReason != nil {
			v.used.Add(features.ChecksumTristateV1)
		}
	}
	for _, p := range v.bundle.GetPrograms() {
		// features.md section 11 freezes Program.subject_node into this
		// capability alongside the captures. Deriving it from the captures
		// alone let a bundle declare a subject node without declaring the
		// capability that owns the field: three engines refused a fixture here
		// that this loader accepted, and they were reading features.md.
		if len(p.GetCaptures()) > 0 || p.SubjectNode != nil {
			v.used.Add(features.CapturesAndCallsV1)
		}
	}
	declared := map[uint32]bool{}
	for _, id := range v.bundle.GetRequiredFeatureIds() {
		declared[id] = true
	}
	for _, id := range v.used.Sorted() {
		if !declared[id] {
			capability, _ := features.Lookup(id)
			return invalidf("the bundle uses the capability %s (%d) without declaring it", capability.Name, id)
		}
	}
	return nil
}

func preCanonicalizationAllowed(kind irv1.CanonicalizationOpKind) bool {
	switch kind {
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_TRIM_WHITESPACE,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_WHITESPACE,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_UPPERCASE_ASCII,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_CHARS,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_SEQUENCE:
		return true
	default:
		return false
	}
}

func definitionOrderBefore(a, b *irv1.IdentifierDefinition) bool {
	if a.GetKind() != b.GetKind() {
		return a.GetKind() < b.GetKind()
	}
	switch {
	case a.CountryCode == nil && b.CountryCode == nil:
		return false
	case a.CountryCode == nil:
		return true
	case b.CountryCode == nil:
		return false
	default:
		return a.GetCountryCode() < b.GetCountryCode()
	}
}

func targetOrderBefore(a, b *irv1.DispatchTarget) bool {
	switch {
	case a.CountryCode == nil && b.CountryCode == nil:
		return false
	case a.CountryCode == nil:
		return true
	case b.CountryCode == nil:
		return false
	default:
		return a.GetCountryCode() < b.GetCountryCode()
	}
}

// globalCountry is the sort key of a definition or target without country.
const globalCountry = "GLOBAL"

func globalOr(country *string) string {
	if country == nil {
		return globalCountry
	}
	return *country
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func validKind(s string) bool {
	if s == "" || len(s) > limits.MaxKindLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
		case (c >= '0' && c <= '9') || c == '_':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func validKindAlias(s string) bool {
	if s == "" || len(s) > limits.MaxKindLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
		case (c >= '0' && c <= '9') || c == '_' || c == '-':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func validCountry(s string) bool {
	return len(s) == 2 && s[0] >= 'A' && s[0] <= 'Z' && s[1] >= 'A' && s[1] <= 'Z'
}

func validPrefix(s string) bool {
	if len(s) < limits.MinPrefixLength || len(s) > limits.MaxPrefixLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9', c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z':
		default:
			return false
		}
	}
	return true
}
