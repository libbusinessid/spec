package reference

import (
	"strings"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/artifact"
	"github.com/libbusinessid/spec/internal/limits"
)

// Engine executes a validated rule bundle. It is immutable after construction
// and safe for concurrent use.
type Engine struct {
	rules *artifact.Ruleset
}

// NewEngine validates the bundle bytes and returns a ready engine.
func NewEngine(raw []byte) (*Engine, error) {
	rules, err := artifact.LoadRuleset(raw)
	if err != nil {
		return nil, err
	}
	return &Engine{rules: rules}, nil
}

// NewEngineFromRuleset wraps an already validated ruleset.
func NewEngineFromRuleset(rules *artifact.Ruleset) *Engine { return &Engine{rules: rules} }

// Ruleset exposes the validated bundle for inspection.
func (e *Engine) Ruleset() *artifact.Ruleset { return e.rules }

// dispatchOutcome is the intermediate state produced by the dispatch algorithm.
type dispatchOutcome struct {
	// kind is the reported kind: canonical after a dispatcher is resolved,
	// otherwise the trimmed and lower cased requested token.
	kind string
	// canonicalValue is the reported value at the point the dispatch stopped.
	canonicalValue string
	// countryCode is the reported country context.
	countryCode *string
	// definition is the selected definition, nil when the dispatch failed.
	definition *irv1.IdentifierDefinition
	target     *irv1.DispatchTarget
	// reason is the failure reason, REASON_CODE_OK on success.
	reason irv1.ReasonCode
}

// dispatch implements the nine normative steps of the dispatch algorithm.
//
//nolint:gocyclo,funlen // the normative algorithm is one linear sequence.
func (e *Engine) dispatch(m *machine, in Input, profile Profile) (dispatchOutcome, error) {
	requested := lowerASCII(trimASCII(in.Kind))
	out := dispatchOutcome{
		kind:           requested,
		canonicalValue: in.Value,
		countryCode:    in.CountryCode,
		reason:         irv1.ReasonCode_REASON_CODE_OK,
	}

	// 1 and 2: resolve the dispatcher through the kind alias table.
	dispatcher, ok := e.rules.DispatcherByKind[requested]
	if !ok || !validKindToken(requested) {
		out.reason = irv1.ReasonCode_REASON_CODE_UNSUPPORTED_KIND
		return out, nil
	}
	out.kind = dispatcher.GetKind()

	// 4: the routing pre-canonicalizer runs exactly once. It cannot fail, so
	// running it before the country checks is observationally identical and
	// lets every post dispatcher result report the pre-canonical value.
	pre, err := m.program(dispatcher.GetPreCanonicalizationProgram())
	if err != nil {
		return out, err
	}
	base := &frame{profile: profile}
	preValue, err := m.RunCanonicalization(pre, base, in.Value)
	if err != nil {
		return out, err
	}
	out.canonicalValue = preValue

	// 3: normalize the explicit country context.
	var country *string
	if in.CountryCode != nil {
		token := upperASCII(trimASCII(*in.CountryCode))
		if token != "" {
			if !validCountryToken(token) {
				out.reason = irv1.ReasonCode_REASON_CODE_UNSUPPORTED_COUNTRY
				return out, nil
			}
			for _, alias := range dispatcher.GetCountryAliases() {
				if alias.GetAlias() == token {
					token = alias.GetCountryCode()
					break
				}
			}
			country = &token
			out.countryCode = &token
		} else {
			out.countryCode = nil
		}
	}

	globalTarget := globalTargetOf(dispatcher)
	var countryTarget *irv1.DispatchTarget
	if country != nil {
		countryTarget = countryTargetOf(dispatcher, *country)
		if countryTarget == nil && globalTarget == nil {
			out.reason = irv1.ReasonCode_REASON_CODE_UNSUPPORTED_COUNTRY
			return out, nil
		}
	}

	// 5: longest exact accepted prefix.
	prefixTarget := longestPrefixTarget(dispatcher, preValue)

	// 6: an explicit country contradicting a recognized prefix.
	if countryTarget != nil && prefixTarget != nil && countryTarget != prefixTarget {
		out.reason = irv1.ReasonCode_REASON_CODE_COUNTRY_MISMATCH
		return out, nil
	}

	// 7: selection priority.
	target := countryTarget
	if target == nil {
		target = prefixTarget
	}
	if target == nil {
		target = globalTarget
	}
	if target == nil {
		target = implicitTargetOf(dispatcher)
	}
	// 8: nothing selectable.
	if target == nil {
		out.reason = irv1.ReasonCode_REASON_CODE_MISSING_COUNTRY_CODE
		return out, nil
	}

	definition, ok := e.rules.DefinitionByID[target.GetIdentifierDefinitionId()]
	if !ok {
		return out, enginef("dispatch target references the unknown definition %d",
			target.GetIdentifierDefinitionId())
	}
	if definition.GetKind() != dispatcher.GetKind() {
		return out, enginef("definition %d does not belong to kind %q", definition.GetId(), dispatcher.GetKind())
	}

	// 9: the definition canonicalizer runs exactly once on the pre-canonical
	// value.
	targetCountry := definition.CountryCode
	canonProgram, err := m.program(definition.GetCanonicalizationProgram())
	if err != nil {
		return out, err
	}
	canonFrame := &frame{profile: profile, country: targetCountry, target: target}
	canonical, err := m.RunCanonicalization(canonProgram, canonFrame, preValue)
	if err != nil {
		return out, err
	}
	out.canonicalValue = canonical
	out.definition = definition
	out.target = target
	if targetCountry != nil {
		out.countryCode = targetCountry
	}
	return out, nil
}

func globalTargetOf(d *irv1.IdentifierDispatcher) *irv1.DispatchTarget {
	for _, t := range d.GetTargets() {
		if t.CountryCode == nil {
			return t
		}
	}
	return nil
}

func countryTargetOf(d *irv1.IdentifierDispatcher, country string) *irv1.DispatchTarget {
	for _, t := range d.GetTargets() {
		if t.CountryCode != nil && t.GetCountryCode() == country {
			return t
		}
	}
	return nil
}

func implicitTargetOf(d *irv1.IdentifierDispatcher) *irv1.DispatchTarget {
	for _, t := range d.GetTargets() {
		if t.GetAllowUnprefixedWithoutCountry() {
			return t
		}
	}
	return nil
}

func longestPrefixTarget(d *irv1.IdentifierDispatcher, value string) *irv1.DispatchTarget {
	var best *irv1.DispatchTarget
	bestLen := 0
	for _, t := range d.GetTargets() {
		for _, prefix := range t.GetAcceptedPrefixes() {
			if len(prefix) > bestLen && strings.HasPrefix(value, prefix) {
				best, bestLen = t, len(prefix)
			}
		}
	}
	return best
}

func validKindToken(s string) bool {
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

func validCountryToken(s string) bool {
	return len(s) == 2 && s[0] >= 'A' && s[0] <= 'Z' && s[1] >= 'A' && s[1] <= 'Z'
}
