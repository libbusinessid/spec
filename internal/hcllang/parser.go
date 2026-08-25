package hcllang

import (
	"sort"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/entid-org/spec/internal/ast"
	"github.com/entid-org/spec/internal/diagnostics"
)

// Diagnostic codes emitted by the parser.
const (
	CodeSyntax           = "HCL001"
	CodeUnknownBlock     = "HCL002"
	CodeUnknownAttribute = "HCL003"
	CodeBadLabels        = "HCL004"
	CodeMissingAttribute = "HCL005"
	CodeBadExpression    = "HCL006"
	CodeBadValue         = "HCL007"
	CodeDuplicateBlock   = "HCL008"
	CodeEncoding         = "HCL009"
)

// ParseUnit parses every source file of a compilation unit. Files must already
// be sorted by relative path; the returned unit preserves that order.
func ParseUnit(files []SourceFile, read func(SourceFile) ([]byte, error)) (*ast.Unit, *diagnostics.Bag) {
	bag := diagnostics.New()
	unit := &ast.Unit{}
	for _, sf := range files {
		src, err := read(sf)
		if err != nil {
			bag.Errorf(diagnostics.Position{File: sf.RelPath}, CodeEncoding, "cannot read file: %v", err)
			continue
		}
		file, fileBag := ParseFile(sf.RelPath, src)
		bag.Extend(fileBag)
		if file != nil {
			unit.Files = append(unit.Files, file)
		}
	}
	return unit, bag
}

// ParseFile parses one source file into the typed AST.
func ParseFile(relPath string, src []byte) (*ast.File, *diagnostics.Bag) {
	bag := diagnostics.New()
	if !utf8.Valid(src) {
		bag.Errorf(diagnostics.Position{File: relPath}, CodeEncoding, "source is not valid UTF-8")
		return nil, bag
	}
	if len(src) >= 3 && src[0] == 0xEF && src[1] == 0xBB && src[2] == 0xBF {
		bag.Suggestf(diagnostics.Position{File: relPath}, CodeEncoding,
			"remove the byte order mark", "source starts with a UTF-8 byte order mark")
		return nil, bag
	}

	parsed, hclDiags := hclsyntax.ParseConfig(src, relPath, hcl.Pos{Line: 1, Column: 1})
	p := &parser{file: relPath, bag: bag}
	p.addHCLDiags(hclDiags)
	if hclDiags.HasErrors() {
		return nil, bag
	}
	body, ok := parsed.Body.(*hclsyntax.Body)
	if !ok { // unreachable with hclsyntax.ParseConfig
		bag.Errorf(diagnostics.Position{File: relPath}, CodeSyntax, "unsupported HCL body")
		return nil, bag
	}
	out := &ast.File{Path: relPath}
	p.parseTopLevel(body, out)
	return out, bag
}

type parser struct {
	file string
	bag  *diagnostics.Bag
}

func (p *parser) pos(r hcl.Range) diagnostics.Position {
	return diagnostics.Position{File: p.file, Line: r.Start.Line, Column: r.Start.Column}
}

func (p *parser) addHCLDiags(diags hcl.Diagnostics) {
	for _, d := range diags {
		pos := diagnostics.Position{File: p.file}
		if d.Subject != nil {
			pos = p.pos(*d.Subject)
		}
		msg := d.Summary
		if d.Detail != "" {
			msg = d.Summary + ": " + d.Detail
		}
		p.bag.Errorf(pos, CodeSyntax, "%s", msg)
	}
}

func (p *parser) parseTopLevel(body *hclsyntax.Body, out *ast.File) {
	for _, attr := range sortedAttributes(body) {
		p.bag.Suggestf(p.pos(attr.SrcRange), CodeUnknownAttribute,
			"top level declarations are blocks, not attributes",
			"unexpected top level attribute %q", attr.Name)
	}
	for _, block := range body.Blocks {
		switch block.Type {
		case "canonicalizer":
			if d := p.parseCanonicalizer(block); d != nil {
				out.Canonicalizers = append(out.Canonicalizers, d)
			}
		case "format":
			if d := p.parseFormat(block); d != nil {
				out.Formats = append(out.Formats, d)
			}
		case "checksum":
			if d := p.parseChecksum(block); d != nil {
				out.Checksums = append(out.Checksums, d)
			}
		case "dispatcher":
			if d := p.parseDispatcher(block); d != nil {
				out.Dispatchers = append(out.Dispatchers, d)
			}
		case "identifier":
			if d := p.parseIdentifier(block); d != nil {
				out.Identifiers = append(out.Identifiers, d)
			}
		default:
			p.bag.Suggestf(p.pos(block.DefRange()), CodeUnknownBlock,
				"expected canonicalizer, format, checksum, dispatcher or identifier",
				"unknown top level block %q", block.Type)
		}
	}
}

func sortedAttributes(body *hclsyntax.Body) []*hclsyntax.Attribute {
	out := make([]*hclsyntax.Attribute, 0, len(body.Attributes))
	for _, a := range body.Attributes {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SrcRange.Start.Line != out[j].SrcRange.Start.Line {
			return out[i].SrcRange.Start.Line < out[j].SrcRange.Start.Line
		}
		if out[i].SrcRange.Start.Column != out[j].SrcRange.Start.Column {
			return out[i].SrcRange.Start.Column < out[j].SrcRange.Start.Column
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// bodyReader gives strict, position aware access to a block body.
type bodyReader struct {
	p    *parser
	body *hclsyntax.Body
	seen map[string]bool
}

func (p *parser) reader(body *hclsyntax.Body) *bodyReader {
	return &bodyReader{p: p, body: body, seen: map[string]bool{}}
}

func (r *bodyReader) attr(name string) (*hclsyntax.Attribute, bool) {
	r.seen[name] = true
	a, ok := r.body.Attributes[name]
	return a, ok
}

func (r *bodyReader) requireAttr(name string, blockRange hcl.Range) (*hclsyntax.Attribute, bool) {
	a, ok := r.attr(name)
	if !ok {
		r.p.bag.Errorf(r.p.pos(blockRange), CodeMissingAttribute, "missing required attribute %q", name)
		return nil, false
	}
	return a, true
}

// requireString reads a mandatory string attribute. A missing or malformed
// attribute already produced a diagnostic, so the caller keeps the empty value.
func (r *bodyReader) requireString(name string, blockRange hcl.Range) string {
	a, ok := r.requireAttr(name, blockRange)
	if !ok {
		return ""
	}
	value, _ := r.p.stringValue(a.Expr)
	return value
}

func (r *bodyReader) optionalString(name string) (string, diagnostics.Position, bool) {
	a, ok := r.attr(name)
	if !ok {
		return "", diagnostics.Position{}, false
	}
	s, ok := r.p.stringValue(a.Expr)
	return s, r.p.pos(a.SrcRange), ok
}

func (r *bodyReader) optionalBool(name string) (bool, bool) {
	a, ok := r.attr(name)
	if !ok {
		return false, false
	}
	e := r.p.convert(a.Expr)
	lit, ok := e.(*ast.BoolLit)
	if !ok {
		if e != nil {
			r.p.bag.Errorf(e.Pos(), CodeBadValue, "attribute %q expects a boolean literal", name)
		}
		return false, false
	}
	return lit.Value, true
}

func (r *bodyReader) optionalRef(name string) *ast.RefExpr {
	a, ok := r.attr(name)
	if !ok {
		return nil
	}
	return r.p.refValue(a.Expr, name)
}

func (r *bodyReader) requireRef(name string, blockRange hcl.Range) *ast.RefExpr {
	a, ok := r.requireAttr(name, blockRange)
	if !ok {
		return nil
	}
	return r.p.refValue(a.Expr, name)
}

func (r *bodyReader) optionalStringList(name string) ([]string, []diagnostics.Position, bool) {
	a, ok := r.attr(name)
	if !ok {
		return nil, nil, false
	}
	e := r.p.convert(a.Expr)
	list, ok := e.(*ast.ListExpr)
	if !ok {
		if e != nil {
			r.p.bag.Errorf(e.Pos(), CodeBadValue, "attribute %q expects a list of string literals", name)
		}
		return nil, nil, false
	}
	values := make([]string, 0, len(list.Items))
	positions := make([]diagnostics.Position, 0, len(list.Items))
	for _, item := range list.Items {
		lit, ok := item.(*ast.StringLit)
		if !ok {
			r.p.bag.Errorf(item.Pos(), CodeBadValue, "attribute %q expects string literals only", name)
			continue
		}
		values = append(values, lit.Value)
		positions = append(positions, lit.Position)
	}
	return values, positions, true
}

func (r *bodyReader) exprList(name string, blockRange hcl.Range, required bool) []ast.Expr {
	a, ok := r.attr(name)
	if !ok {
		if required {
			r.p.bag.Errorf(r.p.pos(blockRange), CodeMissingAttribute, "missing required attribute %q", name)
		}
		return nil
	}
	e := r.p.convert(a.Expr)
	list, ok := e.(*ast.ListExpr)
	if !ok {
		if e != nil {
			r.p.bag.Errorf(e.Pos(), CodeBadValue, "attribute %q expects a list", name)
		}
		return nil
	}
	return list.Items
}

// finish reports every attribute of the body that was never requested.
func (r *bodyReader) finish(context string) {
	for _, a := range sortedAttributes(r.body) {
		if !r.seen[a.Name] {
			r.p.bag.Errorf(r.p.pos(a.SrcRange), CodeUnknownAttribute,
				"unknown attribute %q in %s", a.Name, context)
		}
	}
}

// blocks returns the blocks of a body, reporting any unexpected type.
func (p *parser) blocks(body *hclsyntax.Body, context string, allowed ...string) map[string][]*hclsyntax.Block {
	set := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		set[a] = true
	}
	out := map[string][]*hclsyntax.Block{}
	for _, b := range body.Blocks {
		if !set[b.Type] {
			p.bag.Errorf(p.pos(b.DefRange()), CodeUnknownBlock, "unknown block %q in %s", b.Type, context)
			continue
		}
		out[b.Type] = append(out[b.Type], b)
	}
	return out
}

func (p *parser) labels(block *hclsyntax.Block, want int) ([]string, bool) {
	if len(block.Labels) != want {
		p.bag.Errorf(p.pos(block.DefRange()), CodeBadLabels,
			"block %q expects exactly %d label(s), got %d", block.Type, want, len(block.Labels))
		return nil, false
	}
	return block.Labels, true
}

func (p *parser) parseCanonicalizer(block *hclsyntax.Block) *ast.Canonicalizer {
	labels, ok := p.labels(block, 2)
	if !ok {
		return nil
	}
	r := p.reader(block.Body)
	steps := r.exprList("steps", block.DefRange(), true)
	r.finish("canonicalizer")
	p.blocks(block.Body, "canonicalizer")
	return &ast.Canonicalizer{
		Namespace: labels[0],
		Name:      labels[1],
		Steps:     steps,
		Position:  p.pos(block.DefRange()),
	}
}

func (p *parser) parseFormat(block *hclsyntax.Block) *ast.Format {
	labels, ok := p.labels(block, 2)
	if !ok {
		return nil
	}
	out := &ast.Format{Namespace: labels[0], Name: labels[1], Position: p.pos(block.DefRange())}
	r := p.reader(block.Body)
	if a, ok := r.attr("subject"); ok {
		out.Subject = p.convert(a.Expr)
	}
	out.Checks = r.exprList("checks", block.DefRange(), true)
	r.finish("format")

	byType := p.blocks(block.Body, "format", "capture", "use_format")
	for _, cb := range byType["capture"] {
		labels, ok := p.labels(cb, 1)
		if !ok {
			continue
		}
		cr := p.reader(cb.Body)
		a, found := cr.requireAttr("value", cb.DefRange())
		cr.finish("capture")
		p.blocks(cb.Body, "capture")
		if !found {
			continue
		}
		out.Captures = append(out.Captures, &ast.Capture{
			Name:     labels[0],
			Value:    p.convert(a.Expr),
			Position: p.pos(cb.DefRange()),
		})
	}
	for _, ub := range byType["use_format"] {
		if _, ok := p.labels(ub, 0); !ok {
			continue
		}
		ur := p.reader(ub.Body)
		rule := ur.requireRef("rule", ub.DefRange())
		var input ast.Expr
		if a, ok := ur.requireAttr("input", ub.DefRange()); ok {
			input = p.convert(a.Expr)
		}
		ur.finish("use_format")
		p.blocks(ub.Body, "use_format")
		if rule == nil || input == nil {
			continue
		}
		out.Uses = append(out.Uses, &ast.UseFormat{Rule: rule, Input: input, Position: p.pos(ub.DefRange())})
	}
	return out
}

func (p *parser) parseChecksum(block *hclsyntax.Block) *ast.Checksum {
	labels, ok := p.labels(block, 2)
	if !ok {
		return nil
	}
	out := &ast.Checksum{Namespace: labels[0], Name: labels[1], Position: p.pos(block.DefRange())}
	r := p.reader(block.Body)
	if a, ok := r.attr("subject"); ok {
		out.Subject = p.convert(a.Expr)
	}
	if a, ok := r.requireAttr("rule", block.DefRange()); ok {
		out.Rule = p.convert(a.Expr)
	}
	r.finish("checksum")
	p.blocks(block.Body, "checksum")
	if out.Rule == nil {
		return nil
	}
	return out
}

func (p *parser) parseDispatcher(block *hclsyntax.Block) *ast.Dispatcher {
	labels, ok := p.labels(block, 1)
	if !ok {
		return nil
	}
	out := &ast.Dispatcher{Kind: labels[0], Position: p.pos(block.DefRange())}
	r := p.reader(block.Body)
	if values, positions, ok := r.optionalStringList("aliases"); ok {
		out.Aliases = values
		out.AliasPositions = positions
	}
	out.PreCanonicalizer = r.requireRef("pre_canonicalizer", block.DefRange())
	if a, ok := r.attr("country_aliases"); ok {
		out.CountryAliases = p.countryAliases(a.Expr)
	}
	r.finish("dispatcher")

	byType := p.blocks(block.Body, "dispatcher", "target")
	for _, tb := range byType["target"] {
		if _, ok := p.labels(tb, 0); !ok {
			continue
		}
		t := &ast.DispatchTarget{Position: p.pos(tb.DefRange())}
		tr := p.reader(tb.Body)
		if code, pos, ok := tr.optionalString("country_code"); ok {
			t.CountryCode = code
			t.CountryPosition = pos
			t.Global = code == "GLOBAL"
		} else {
			t.Global = true
			t.CountryPosition = p.pos(tb.DefRange())
		}
		if values, positions, ok := tr.optionalStringList("accepted_prefixes"); ok {
			t.AcceptedPrefixes = values
			t.PrefixPositions = positions
		}
		if v, _, ok := tr.optionalString("canonical_prefix"); ok {
			t.HasCanonicalPrefix = true
			t.CanonicalPrefix = v
		}
		t.Identifier = tr.requireRef("identifier", tb.DefRange())
		if v, ok := tr.optionalBool("allow_unprefixed_without_country"); ok {
			t.AllowUnprefixedWithoutCountry = v
		}
		tr.finish("target")
		p.blocks(tb.Body, "target")
		if t.Identifier == nil {
			continue
		}
		out.Targets = append(out.Targets, t)
	}
	return out
}

func (p *parser) countryAliases(expr hclsyntax.Expression) []*ast.CountryAlias {
	obj, ok := expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		p.bag.Errorf(p.pos(expr.Range()), CodeBadValue,
			"country_aliases expects an object of string literals")
		return nil
	}
	var out []*ast.CountryAlias
	for _, item := range obj.Items {
		keyExpr := item.KeyExpr
		if wrapped, ok := keyExpr.(*hclsyntax.ObjectConsKeyExpr); ok {
			keyExpr = wrapped.Wrapped
		}
		var key string
		switch k := keyExpr.(type) {
		case *hclsyntax.ScopeTraversalExpr:
			if len(k.Traversal) != 1 {
				p.bag.Errorf(p.pos(k.Range()), CodeBadValue, "country_aliases keys must be string literals")
				continue
			}
			root, ok := k.Traversal[0].(hcl.TraverseRoot)
			if !ok {
				p.bag.Errorf(p.pos(k.Range()), CodeBadValue, "country_aliases keys must be string literals")
				continue
			}
			key = root.Name
		default:
			s, ok := p.stringValue(keyExpr)
			if !ok {
				continue
			}
			key = s
		}
		value, ok := p.stringValue(item.ValueExpr)
		if !ok {
			continue
		}
		out = append(out, &ast.CountryAlias{
			Alias:       key,
			CountryCode: value,
			Position:    p.pos(item.ValueExpr.Range()),
		})
	}
	return out
}

func (p *parser) parseIdentifier(block *hclsyntax.Block) *ast.Identifier {
	labels, ok := p.labels(block, 2)
	if !ok {
		return nil
	}
	out := &ast.Identifier{Kind: labels[0], Position: p.pos(block.DefRange())}
	if labels[1] == "GLOBAL" {
		out.Global = true
	} else {
		out.CountryCode = labels[1]
	}
	r := p.reader(block.Body)
	out.Canonicalizer = r.requireRef("canonicalizer", block.DefRange())
	out.Format = r.requireRef("format", block.DefRange())
	out.Checksum = r.optionalRef("checksum")
	if v, pos, ok := r.optionalString("default_profile"); ok {
		out.DefaultProfile = v
		_ = pos
	}
	r.finish("identifier")

	byType := p.blocks(block.Body, "identifier", "source", "no_checksum")
	for _, sb := range byType["source"] {
		if _, ok := p.labels(sb, 0); !ok {
			continue
		}
		s := &ast.Source{Position: p.pos(sb.DefRange())}
		sr := p.reader(sb.Body)
		s.ID = sr.requireString("id", sb.DefRange())
		s.URL = sr.requireString("url", sb.DefRange())
		s.Authority = sr.requireString("authority", sb.DefRange())
		s.Title = sr.requireString("title", sb.DefRange())
		s.AccessedAt = sr.requireString("accessed_at", sb.DefRange())
		s.Jurisdiction = sr.requireString("jurisdiction", sb.DefRange())
		s.Language = sr.requireString("language", sb.DefRange())
		s.Notes = sr.requireString("notes", sb.DefRange())
		s.LicenseOrTerms = sr.requireString("license_or_terms", sb.DefRange())
		s.Tier = sr.requireString("tier", sb.DefRange())
		if v, _, ok := sr.optionalString("archive_url"); ok {
			s.HasArchiveURL = true
			s.ArchiveURL = v
		}
		sr.finish("source")
		p.blocks(sb.Body, "source")
		out.Sources = append(out.Sources, s)
	}
	for _, nb := range byType["no_checksum"] {
		if _, ok := p.labels(nb, 0); !ok {
			continue
		}
		nr := p.reader(nb.Body)
		reason := nr.requireString("reason_code", nb.DefRange())
		notes := nr.requireString("notes", nb.DefRange())
		nr.finish("no_checksum")
		p.blocks(nb.Body, "no_checksum")
		if out.NoChecksum != nil {
			p.bag.Errorf(p.pos(nb.DefRange()), CodeDuplicateBlock, "duplicate no_checksum block")
			continue
		}
		out.NoChecksum = &ast.NoChecksum{ReasonCode: reason, Notes: notes, Position: p.pos(nb.DefRange())}
	}
	return out
}

func (p *parser) stringValue(expr hclsyntax.Expression) (string, bool) {
	e := p.convert(expr)
	lit, ok := e.(*ast.StringLit)
	if !ok {
		if e != nil {
			p.bag.Errorf(e.Pos(), CodeBadValue, "expected a quoted string literal")
		}
		return "", false
	}
	return lit.Value, true
}

func (p *parser) refValue(expr hclsyntax.Expression, name string) *ast.RefExpr {
	e := p.convert(expr)
	ref, ok := e.(*ast.RefExpr)
	if !ok {
		if e != nil {
			p.bag.Suggestf(e.Pos(), CodeBadValue,
				"write a structural reference such as format.fr.siren",
				"attribute %q expects a symbolic reference", name)
		}
		return nil
	}
	return ref
}

// convert translates one HCL expression into the typed AST. It never evaluates
// a variable, a function or a template.
func (p *parser) convert(expr hclsyntax.Expression) ast.Expr {
	switch e := expr.(type) {
	case *hclsyntax.ParenthesesExpr:
		return p.convert(e.Expression)
	case *hclsyntax.FunctionCallExpr:
		return p.convertCall(e)
	case *hclsyntax.ScopeTraversalExpr:
		return p.convertTraversal(e)
	case *hclsyntax.TemplateExpr:
		return p.convertTemplate(e)
	case *hclsyntax.TemplateWrapExpr:
		return p.convert(e.Wrapped)
	case *hclsyntax.LiteralValueExpr:
		return p.literal(e.Val, p.pos(e.Range()))
	case *hclsyntax.TupleConsExpr:
		return p.convertTuple(e)
	case *hclsyntax.UnaryOpExpr:
		return p.convertUnary(e)
	default:
		p.bag.Suggestf(p.pos(expr.Range()), CodeBadExpression,
			"only literals, lists, dotted references and calls of the LibEntID language are accepted",
			"unsupported expression")
		return nil
	}
}

func (p *parser) convertCall(e *hclsyntax.FunctionCallExpr) ast.Expr {
	args := make([]ast.Expr, 0, len(e.Args))
	for _, a := range e.Args {
		converted := p.convert(a)
		if converted == nil {
			return nil
		}
		args = append(args, converted)
	}
	if e.ExpandFinal {
		p.bag.Errorf(p.pos(e.Range()), CodeBadExpression, "argument expansion is not supported")
		return nil
	}
	return &ast.CallExpr{Name: e.Name, Args: args, Position: p.pos(e.NameRange)}
}

func (p *parser) convertTraversal(e *hclsyntax.ScopeTraversalExpr) ast.Expr {
	parts := make([]string, 0, len(e.Traversal))
	for _, step := range e.Traversal {
		switch t := step.(type) {
		case hcl.TraverseRoot:
			parts = append(parts, t.Name)
		case hcl.TraverseAttr:
			parts = append(parts, t.Name)
		default:
			p.bag.Errorf(p.pos(e.Range()), CodeBadExpression,
				"only dotted symbolic references are supported")
			return nil
		}
	}
	return &ast.RefExpr{Parts: parts, Position: p.pos(e.Range())}
}

func (p *parser) convertTemplate(e *hclsyntax.TemplateExpr) ast.Expr {
	if !e.IsStringLiteral() {
		p.bag.Suggestf(p.pos(e.Range()), CodeBadExpression,
			"references are structural and typed; text interpolation cannot compose rules",
			"string interpolation is not supported")
		return nil
	}
	val, diags := e.Value(nil)
	if diags.HasErrors() {
		p.addHCLDiags(diags)
		return nil
	}
	return &ast.StringLit{Value: val.AsString(), Position: p.pos(e.Range())}
}

func (p *parser) convertTuple(e *hclsyntax.TupleConsExpr) ast.Expr {
	items := make([]ast.Expr, 0, len(e.Exprs))
	for _, item := range e.Exprs {
		converted := p.convert(item)
		if converted == nil {
			return nil
		}
		items = append(items, converted)
	}
	return &ast.ListExpr{Items: items, Position: p.pos(e.Range())}
}

func (p *parser) convertUnary(e *hclsyntax.UnaryOpExpr) ast.Expr {
	if e.Op == hclsyntax.OpNegate {
		inner := p.convert(e.Val)
		if lit, ok := inner.(*ast.IntLit); ok {
			return &ast.IntLit{Value: -lit.Value, Position: p.pos(e.Range())}
		}
	}
	p.bag.Errorf(p.pos(e.Range()), CodeBadExpression, "unsupported operator")
	return nil
}

func (p *parser) literal(val cty.Value, pos diagnostics.Position) ast.Expr {
	if val.IsNull() {
		p.bag.Errorf(pos, CodeBadValue, "null is not a value of the language")
		return nil
	}
	switch val.Type() {
	case cty.String:
		return &ast.StringLit{Value: val.AsString(), Position: pos}
	case cty.Bool:
		return &ast.BoolLit{Value: val.True(), Position: pos}
	case cty.Number:
		bf := val.AsBigFloat()
		if !bf.IsInt() {
			p.bag.Errorf(pos, CodeBadValue, "only integer numbers are supported")
			return nil
		}
		i, acc := bf.Int64()
		if acc != 0 {
			p.bag.Errorf(pos, CodeBadValue, "integer literal does not fit in a signed 64 bit integer")
			return nil
		}
		return &ast.IntLit{Value: i, Position: pos}
	default:
		p.bag.Errorf(pos, CodeBadValue, "unsupported literal of type %s", val.Type().FriendlyName())
		return nil
	}
}
