// Package linker builds the single global symbol table of a compilation unit,
// validates every declaration level invariant of the language and proves that
// the program call graph is acyclic and shallow enough.
package linker

import (
	"sort"
	"strings"

	"github.com/libbusinessid/spec/internal/ast"
	"github.com/libbusinessid/spec/internal/diagnostics"
	"github.com/libbusinessid/spec/internal/limits"
)

// Diagnostic codes emitted by the linker.
const (
	CodeDuplicateSymbol   = "LINK001"
	CodeUnknownSymbol     = "LINK002"
	CodeBadLabel          = "LINK003"
	CodeDispatch          = "LINK004"
	CodeOrphanDefinition  = "LINK005"
	CodeCycle             = "LINK006"
	CodeCallDepth         = "LINK007"
	CodeMissingSource     = "LINK008"
	CodeProfile           = "LINK009"
	CodeChecksumDecl      = "LINK010"
	CodeSourceField       = "LINK011"
	CodePrefixCase        = "LINK012"
	CodeMissingDispatcher = "LINK013"
)

// Table is the resolved global symbol table of a compilation unit.
type Table struct {
	Canonicalizers  map[string]*ast.Canonicalizer
	Formats         map[string]*ast.Format
	Checksums       map[string]*ast.Checksum
	Identifiers     map[string]*ast.Identifier
	Dispatchers     []*ast.Dispatcher
	IdentifierOrder []*ast.Identifier
	// TargetOf maps an identifier symbol to the dispatcher target selecting it.
	TargetOf map[string]*ast.DispatchTarget
	// DispatcherOf maps an identifier symbol to its dispatcher.
	DispatcherOf map[string]*ast.Dispatcher
}

// Link resolves the compilation unit into a symbol table.
func Link(unit *ast.Unit) (*Table, *diagnostics.Bag) {
	bag := diagnostics.New()
	t := &Table{
		Canonicalizers: map[string]*ast.Canonicalizer{},
		Formats:        map[string]*ast.Format{},
		Checksums:      map[string]*ast.Checksum{},
		Identifiers:    map[string]*ast.Identifier{},
		TargetOf:       map[string]*ast.DispatchTarget{},
		DispatcherOf:   map[string]*ast.Dispatcher{},
	}
	l := &linker{table: t, bag: bag}
	l.collect(unit)
	l.checkDeclarations()
	l.checkDispatchers()
	l.checkCallGraph()
	return t, bag
}

type linker struct {
	table *Table
	bag   *diagnostics.Bag
}

func (l *linker) duplicate(pos diagnostics.Position, symbol string) {
	l.bag.Errorf(pos, CodeDuplicateSymbol, "duplicate symbol %q", symbol)
}

func (l *linker) collect(unit *ast.Unit) {
	for _, c := range unit.Canonicalizers() {
		if !l.checkNamespaceName(c.Position, c.Namespace, c.Name) {
			continue
		}
		if _, dup := l.table.Canonicalizers[c.Symbol()]; dup {
			l.duplicate(c.Position, c.Symbol())
			continue
		}
		l.table.Canonicalizers[c.Symbol()] = c
	}
	for _, f := range unit.Formats() {
		if !l.checkNamespaceName(f.Position, f.Namespace, f.Name) {
			continue
		}
		if _, dup := l.table.Formats[f.Symbol()]; dup {
			l.duplicate(f.Position, f.Symbol())
			continue
		}
		l.table.Formats[f.Symbol()] = f
	}
	for _, c := range unit.Checksums() {
		if !l.checkNamespaceName(c.Position, c.Namespace, c.Name) {
			continue
		}
		if _, dup := l.table.Checksums[c.Symbol()]; dup {
			l.duplicate(c.Position, c.Symbol())
			continue
		}
		l.table.Checksums[c.Symbol()] = c
	}
	for _, id := range unit.Identifiers() {
		if !l.checkIdentifierLabels(id) {
			continue
		}
		if _, dup := l.table.Identifiers[id.Symbol()]; dup {
			l.duplicate(id.Position, id.Symbol())
			continue
		}
		l.table.Identifiers[id.Symbol()] = id
		l.table.IdentifierOrder = append(l.table.IdentifierOrder, id)
	}
	seenKind := map[string]bool{}
	for _, d := range unit.Dispatchers() {
		if !validKind(d.Kind) {
			l.bag.Suggestf(d.Position, CodeBadLabel,
				"a canonical kind matches [a-z][a-z0-9_]{0,63} so that identifier.<kind>.<country> stays writable",
				"invalid dispatcher kind %q", d.Kind)
			continue
		}
		if seenKind[d.Kind] {
			l.duplicate(d.Position, d.Symbol())
			continue
		}
		seenKind[d.Kind] = true
		l.table.Dispatchers = append(l.table.Dispatchers, d)
	}
	sort.SliceStable(l.table.Dispatchers, func(i, j int) bool {
		return l.table.Dispatchers[i].Kind < l.table.Dispatchers[j].Kind
	})
}

func (l *linker) checkNamespaceName(pos diagnostics.Position, namespace, name string) bool {
	ok := true
	if !validSymbolPart(namespace) {
		l.bag.Errorf(pos, CodeBadLabel, "invalid namespace label %q", namespace)
		ok = false
	}
	if !validSymbolPart(name) {
		l.bag.Errorf(pos, CodeBadLabel, "invalid name label %q", name)
		ok = false
	}
	return ok
}

func (l *linker) checkIdentifierLabels(id *ast.Identifier) bool {
	ok := true
	if !validKind(id.Kind) {
		l.bag.Errorf(id.Position, CodeBadLabel, "invalid identifier kind %q", id.Kind)
		ok = false
	}
	if !id.Global && !validCountry(id.CountryCode) {
		l.bag.Suggestf(id.Position, CodeBadLabel,
			"use an ISO 3166-1 alpha-2 code in upper case, or the literal GLOBAL",
			"invalid identifier country label %q", id.CountryCode)
		ok = false
	}
	return ok
}

// checkDeclarations validates every identifier declaration against the table.
func (l *linker) checkDeclarations() {
	for _, id := range l.table.IdentifierOrder {
		if id.Canonicalizer != nil {
			l.resolveCanonicalizer(id.Canonicalizer)
		}
		var format *ast.Format
		if id.Format != nil {
			format = l.resolveFormat(id.Format)
		}
		if id.Checksum != nil {
			l.resolveChecksum(id.Checksum)
		}
		switch {
		case id.Checksum != nil && id.NoChecksum != nil:
			l.bag.Errorf(id.Position, CodeChecksumDecl,
				"%s declares both a checksum and a no_checksum block", id.Symbol())
		case id.Checksum == nil && id.NoChecksum == nil:
			l.bag.Suggestf(id.Position, CodeChecksumDecl,
				"add `checksum = checksum.<ns>.<name>` or a no_checksum block with its reason code",
				"%s must declare a checksum or an explicit absence", id.Symbol())
		case id.NoChecksum != nil:
			switch id.NoChecksum.ReasonCode {
			case "unsupported_checksum", "checksum_not_published":
			default:
				l.bag.Errorf(id.NoChecksum.Position, CodeChecksumDecl,
					"no_checksum reason_code must be unsupported_checksum or checksum_not_published, got %q",
					id.NoChecksum.ReasonCode)
			}
			if strings.TrimSpace(id.NoChecksum.Notes) == "" {
				l.bag.Errorf(id.NoChecksum.Position, CodeChecksumDecl,
					"no_checksum must document why no algorithm is applied")
			}
		}
		switch id.DefaultProfile {
		case "compatible", "strict_current":
		case "":
			l.bag.Errorf(id.Position, CodeProfile, "%s must declare default_profile", id.Symbol())
		default:
			l.bag.Errorf(id.Position, CodeProfile,
				"invalid default_profile %q, expected compatible or strict_current", id.DefaultProfile)
		}
		l.checkSources(id, format)
	}
}

// canReject reports whether the format rule of a definition is able to reject
// an input, which makes provenance mandatory.
func (l *linker) canReject(format *ast.Format) bool {
	if format == nil {
		return true
	}
	if len(format.Uses) > 0 {
		return true
	}
	for _, check := range format.Checks {
		reject := false
		ast.Walk(check, func(e ast.Expr) {
			if call, ok := e.(*ast.CallExpr); ok && call.Name == "require" {
				reject = true
			}
		})
		if reject {
			return true
		}
	}
	return false
}

func (l *linker) checkSources(id *ast.Identifier, format *ast.Format) {
	if len(id.Sources) == 0 {
		if l.canReject(format) {
			l.bag.Suggestf(id.Position, CodeMissingSource,
				"every rule able to reject an input must carry at least one source block",
				"%s declares no source", id.Symbol())
		}
		return
	}
	seen := map[string]bool{}
	for _, s := range id.Sources {
		if seen[s.ID] {
			l.bag.Errorf(s.Position, CodeSourceField, "duplicate source id %q", s.ID)
		}
		seen[s.ID] = true
		for name, value := range map[string]string{
			"id": s.ID, "url": s.URL, "authority": s.Authority, "title": s.Title,
			"accessed_at": s.AccessedAt, "jurisdiction": s.Jurisdiction,
			"language": s.Language, "license_or_terms": s.LicenseOrTerms,
		} {
			if strings.TrimSpace(value) == "" {
				l.bag.Errorf(s.Position, CodeSourceField, "source field %q must not be empty", name)
			}
		}
		if !isISODate(s.AccessedAt) {
			l.bag.Errorf(s.Position, CodeSourceField,
				"source accessed_at must be an ISO 8601 date YYYY-MM-DD, got %q", s.AccessedAt)
		}
	}
}

func (l *linker) resolveCanonicalizer(ref *ast.RefExpr) *ast.Canonicalizer {
	if len(ref.Parts) != 3 || ref.Parts[0] != "canonicalizer" {
		l.bag.Errorf(ref.Position, CodeUnknownSymbol,
			"expected a canonicalizer.<namespace>.<name> reference, got %q", ref.String())
		return nil
	}
	d, ok := l.table.Canonicalizers[ref.String()]
	if !ok {
		l.unknown(ref, l.canonicalizerNames())
		return nil
	}
	return d
}

func (l *linker) resolveFormat(ref *ast.RefExpr) *ast.Format {
	if len(ref.Parts) != 3 || ref.Parts[0] != "format" {
		l.bag.Errorf(ref.Position, CodeUnknownSymbol,
			"expected a format.<namespace>.<name> reference, got %q", ref.String())
		return nil
	}
	d, ok := l.table.Formats[ref.String()]
	if !ok {
		l.unknown(ref, l.formatNames())
		return nil
	}
	return d
}

func (l *linker) resolveChecksum(ref *ast.RefExpr) *ast.Checksum {
	if len(ref.Parts) != 3 || ref.Parts[0] != "checksum" {
		l.bag.Errorf(ref.Position, CodeUnknownSymbol,
			"expected a checksum.<namespace>.<name> reference, got %q", ref.String())
		return nil
	}
	d, ok := l.table.Checksums[ref.String()]
	if !ok {
		l.unknown(ref, l.checksumNames())
		return nil
	}
	return d
}

func (l *linker) unknown(ref *ast.RefExpr, candidates []string) {
	if best, ok := nearest(ref.String(), candidates); ok {
		l.bag.Suggestf(ref.Position, CodeUnknownSymbol, "did you mean "+best+"?",
			"unknown symbol %q", ref.String())
		return
	}
	l.bag.Errorf(ref.Position, CodeUnknownSymbol, "unknown symbol %q", ref.String())
}

func (l *linker) canonicalizerNames() []string { return sortedKeys(l.table.Canonicalizers) }
func (l *linker) formatNames() []string        { return sortedKeys(l.table.Formats) }
func (l *linker) checksumNames() []string      { return sortedKeys(l.table.Checksums) }

func sortedKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// checkDispatchers validates the routing tables against the definitions.
func (l *linker) checkDispatchers() {
	claimed := map[string]*ast.Dispatcher{}
	for _, d := range l.table.Dispatchers {
		l.checkDispatcher(d, claimed)
	}
	for _, id := range l.table.IdentifierOrder {
		if _, ok := claimed[id.Symbol()]; !ok {
			l.bag.Suggestf(id.Position, CodeOrphanDefinition,
				"add a dispatcher target referencing this definition",
				"%s is not referenced by any dispatcher target", id.Symbol())
		}
	}
}

func (l *linker) checkDispatcher(d *ast.Dispatcher, claimed map[string]*ast.Dispatcher) {
	var pre *ast.Canonicalizer
	if d.PreCanonicalizer != nil {
		pre = l.resolveCanonicalizer(d.PreCanonicalizer)
	}
	uppercases := pre != nil && canonicalizerUppercases(pre)

	l.checkKindAliases(d)
	if len(d.Targets) == 0 {
		l.bag.Errorf(d.Position, CodeDispatch, "dispatcher %q declares no target", d.Kind)
		return
	}
	l.checkGlobalShape(d)
	l.checkCountryAliases(d)

	seenTargetCountry := map[string]bool{}
	seenPrefix := map[string]bool{}
	implicit := 0
	for _, t := range d.Targets {
		l.checkTarget(d, t, uppercases, seenTargetCountry, seenPrefix)
		if t.AllowUnprefixedWithoutCountry {
			implicit++
		}
		l.bindTarget(d, t, claimed)
	}
	if implicit > 1 {
		l.bag.Errorf(d.Position, CodeDispatch,
			"dispatcher %q declares %d implicit targets, at most one is allowed", d.Kind, implicit)
	}
}

// checkKindAliases validates the alias space of a dispatcher kind.
func (l *linker) checkKindAliases(d *ast.Dispatcher) {
	seen := map[string]bool{d.Kind: true}
	for i, alias := range d.Aliases {
		pos := d.Position
		if i < len(d.AliasPositions) {
			pos = d.AliasPositions[i]
		}
		if !validKindAlias(alias) {
			l.bag.Errorf(pos, CodeBadLabel, "invalid kind alias %q", alias)
			continue
		}
		if seen[alias] {
			l.bag.Errorf(pos, CodeDispatch, "duplicate kind alias %q", alias)
			continue
		}
		seen[alias] = true
	}
}

// checkGlobalShape enforces that a GLOBAL dispatcher holds exactly one target.
func (l *linker) checkGlobalShape(d *ast.Dispatcher) {
	global := false
	for _, t := range d.Targets {
		if t.Global {
			global = true
		}
	}
	if !global {
		return
	}
	if len(d.Targets) != 1 {
		l.bag.Errorf(d.Position, CodeDispatch,
			"dispatcher %q mixes a GLOBAL target with country targets", d.Kind)
	}
	if len(d.CountryAliases) > 0 {
		l.bag.Errorf(d.Position, CodeDispatch, "a GLOBAL dispatcher must not declare country aliases")
	}
}

// checkCountryAliases validates the country alias table of a dispatcher.
func (l *linker) checkCountryAliases(d *ast.Dispatcher) {
	targetCountries := map[string]bool{}
	for _, t := range d.Targets {
		if !t.Global {
			targetCountries[t.CountryCode] = true
		}
	}
	seen := map[string]bool{}
	for _, ca := range d.CountryAliases {
		switch {
		case !validCountry(ca.Alias):
			l.bag.Errorf(ca.Position, CodeBadLabel, "invalid country alias %q", ca.Alias)
		case !validCountry(ca.CountryCode):
			l.bag.Errorf(ca.Position, CodeBadLabel,
				"country alias %q must map to an ISO 3166-1 alpha-2 code, got %q", ca.Alias, ca.CountryCode)
		case ca.Alias == ca.CountryCode:
			l.bag.Errorf(ca.Position, CodeDispatch, "country alias %q maps to itself", ca.Alias)
		case seen[ca.Alias]:
			l.bag.Errorf(ca.Position, CodeDispatch, "duplicate country alias %q", ca.Alias)
		case targetCountries[ca.Alias]:
			l.bag.Errorf(ca.Position, CodeDispatch,
				"country alias %q shadows a target country of the same dispatcher", ca.Alias)
		default:
			seen[ca.Alias] = true
		}
	}
}

// checkTarget validates one routing entry of a dispatcher.
func (l *linker) checkTarget(d *ast.Dispatcher, t *ast.DispatchTarget, uppercases bool,
	seenTargetCountry, seenPrefix map[string]bool,
) {
	if t.Global {
		if len(t.AcceptedPrefixes) > 0 || t.HasCanonicalPrefix {
			l.bag.Errorf(t.Position, CodeDispatch, "a GLOBAL target must not declare prefixes")
		}
		if t.AllowUnprefixedWithoutCountry {
			l.bag.Errorf(t.Position, CodeDispatch,
				"a GLOBAL target must not declare allow_unprefixed_without_country")
		}
	} else {
		if !validCountry(t.CountryCode) {
			l.bag.Errorf(t.CountryPosition, CodeBadLabel, "invalid target country_code %q", t.CountryCode)
		}
		if seenTargetCountry[t.CountryCode] {
			l.bag.Errorf(t.CountryPosition, CodeDispatch, "two targets declare country %q", t.CountryCode)
		}
		seenTargetCountry[t.CountryCode] = true
	}
	for i, p := range t.AcceptedPrefixes {
		pos := t.Position
		if i < len(t.PrefixPositions) {
			pos = t.PrefixPositions[i]
		}
		switch {
		case !validPrefix(p):
			l.bag.Errorf(pos, CodeBadLabel,
				"invalid prefix %q, expected 1 to %d ASCII alphanumeric characters", p, limits.MaxPrefixLength)
		case seenPrefix[p]:
			l.bag.Errorf(pos, CodeDispatch,
				"prefix %q is claimed by two targets of dispatcher %q", p, d.Kind)
		default:
			if uppercases && strings.ToUpper(p) != p {
				l.bag.Suggestf(pos, CodePrefixCase, "write the prefix in upper case",
					"prefix %q can never match: the pre-canonicalizer upper cases the value", p)
			}
			seenPrefix[p] = true
		}
	}
	if t.HasCanonicalPrefix {
		if !validPrefix(t.CanonicalPrefix) {
			l.bag.Errorf(t.Position, CodeBadLabel, "invalid canonical_prefix %q", t.CanonicalPrefix)
		} else if !containsString(t.AcceptedPrefixes, t.CanonicalPrefix) {
			l.bag.Errorf(t.Position, CodeDispatch,
				"canonical_prefix %q is not part of accepted_prefixes", t.CanonicalPrefix)
		}
	}
}

func (l *linker) bindTarget(d *ast.Dispatcher, t *ast.DispatchTarget, claimed map[string]*ast.Dispatcher) {
	ref := t.Identifier
	if ref == nil {
		return
	}
	if len(ref.Parts) != 3 || ref.Parts[0] != "identifier" {
		l.bag.Errorf(ref.Position, CodeUnknownSymbol,
			"expected an identifier.<kind>.<country> reference, got %q", ref.String())
		return
	}
	id, ok := l.table.Identifiers[ref.String()]
	if !ok {
		l.unknown(ref, sortedKeys(l.table.Identifiers))
		return
	}
	if id.Kind != d.Kind {
		l.bag.Errorf(ref.Position, CodeDispatch,
			"target of dispatcher %q references %s of kind %q", d.Kind, id.Symbol(), id.Kind)
	}
	switch {
	case t.Global && !id.Global:
		l.bag.Errorf(ref.Position, CodeDispatch,
			"GLOBAL target references the country definition %s", id.Symbol())
	case !t.Global && id.Global:
		l.bag.Errorf(ref.Position, CodeDispatch,
			"target of country %q references the GLOBAL definition %s", t.CountryCode, id.Symbol())
	case !t.Global && !id.Global && t.CountryCode != id.CountryCode:
		l.bag.Errorf(ref.Position, CodeDispatch,
			"target of country %q references %s", t.CountryCode, id.Symbol())
	}
	if prev, dup := claimed[id.Symbol()]; dup {
		l.bag.Errorf(ref.Position, CodeDispatch,
			"%s is already referenced by dispatcher %q", id.Symbol(), prev.Kind)
		return
	}
	claimed[id.Symbol()] = d
	l.table.TargetOf[id.Symbol()] = t
	l.table.DispatcherOf[id.Symbol()] = d
}

// checkCallGraph proves that the program call graph is acyclic and that its
// static depth stays within the V1 limit.
func (l *linker) checkCallGraph() {
	edges := map[string][]edge{}
	positions := map[string]diagnostics.Position{}
	for _, f := range l.table.Formats {
		positions[f.Symbol()] = f.Position
		for _, use := range f.Uses {
			if use.Rule == nil {
				continue
			}
			if _, ok := l.table.Formats[use.Rule.String()]; !ok {
				l.unknown(use.Rule, l.formatNames())
				continue
			}
			edges[f.Symbol()] = append(edges[f.Symbol()], edge{to: use.Rule.String(), pos: use.Position})
		}
	}
	for _, c := range l.table.Checksums {
		positions[c.Symbol()] = c.Position
		ast.Walk(c.Rule, func(e ast.Expr) {
			call, ok := e.(*ast.CallExpr)
			if !ok || call.Name != "apply_checksum" || len(call.Args) == 0 {
				return
			}
			ref, ok := call.Args[0].(*ast.RefExpr)
			if !ok {
				return
			}
			if _, ok := l.table.Checksums[ref.String()]; !ok {
				l.unknown(ref, l.checksumNames())
				return
			}
			edges[c.Symbol()] = append(edges[c.Symbol()], edge{to: ref.String(), pos: call.Position})
		})
	}
	for k := range edges {
		sort.SliceStable(edges[k], func(i, j int) bool { return edges[k][i].to < edges[k][j].to })
	}

	roots := make([]string, 0, len(positions))
	for k := range positions {
		roots = append(roots, k)
	}
	sort.Strings(roots)

	state := map[string]int{} // 0 unvisited, 1 in progress, 2 done
	depth := map[string]int{}
	var visit func(symbol string, stack []string) int
	visit = func(symbol string, stack []string) int {
		switch state[symbol] {
		case 1:
			l.bag.Errorf(positions[symbol], CodeCycle,
				"reference cycle: %s", strings.Join(append(stack, symbol), " -> "))
			return 1
		case 2:
			return depth[symbol]
		}
		state[symbol] = 1
		best := 1
		for _, e := range edges[symbol] {
			d := visit(e.to, append(stack, symbol))
			if d+1 > best {
				best = d + 1
			}
		}
		state[symbol] = 2
		depth[symbol] = best
		if best > limits.MaxCallDepth {
			l.bag.Errorf(positions[symbol], CodeCallDepth,
				"static call depth %d exceeds the limit of %d", best, limits.MaxCallDepth)
		}
		return best
	}
	for _, r := range roots {
		visit(r, nil)
	}
}

type edge struct {
	to  string
	pos diagnostics.Position
}

func canonicalizerUppercases(c *ast.Canonicalizer) bool {
	for _, s := range c.Steps {
		found := false
		ast.Walk(s, func(e ast.Expr) {
			if call, ok := e.(*ast.CallExpr); ok && call.Name == "uppercase_ascii" {
				found = true
			}
		})
		if found {
			return true
		}
	}
	return false
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func validSymbolPart(s string) bool {
	if s == "" || len(s) > limits.MaxKindLength+1 {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case (r >= '0' && r <= '9') || r == '_':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func validKind(s string) bool {
	if s == "" || len(s) > limits.MaxKindLength {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case (r >= '0' && r <= '9') || r == '_':
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
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case (r >= '0' && r <= '9') || r == '_' || r == '-':
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
	if len(s) != 2 {
		return false
	}
	return s[0] >= 'A' && s[0] <= 'Z' && s[1] >= 'A' && s[1] <= 'Z'
}

func validPrefix(s string) bool {
	if len(s) < limits.MinPrefixLength || len(s) > limits.MaxPrefixLength {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		default:
			return false
		}
	}
	return true
}

func isISODate(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	for i, r := range s {
		if i == 4 || i == 7 {
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	month := (int(s[5]-'0') * 10) + int(s[6]-'0')
	day := (int(s[8]-'0') * 10) + int(s[9]-'0')
	return month >= 1 && month <= 12 && day >= 1 && day <= 31
}

// nearest returns the closest candidate within a small edit distance.
func nearest(want string, candidates []string) (string, bool) {
	best, bestDistance := "", 1<<30
	for _, c := range candidates {
		d := editDistance(want, c)
		if d < bestDistance {
			best, bestDistance = c, d
		}
	}
	limit := len(want) / 3
	if limit < 1 {
		limit = 1
	}
	if best == "" || bestDistance > limit {
		return "", false
	}
	return best, true
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = minInt(minInt(cur[j-1]+1, prev[j]+1), prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
