package artifact

import (
	"fmt"
	"strings"
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/limits"
)

// A bundle does not have to come from this compiler, so the loader repeats what
// the type checker already refused. CUSTOM_ALPHABET is where two conformant
// engines could quietly disagree: an alphabet listing a code point twice gives
// it two values, and which one wins depends on how an implementation searches.
func TestCheckAlphabet(t *testing.T) {
	const uscc = "0123456789ABCDEFGHJKLMNPQRTUWXY"
	long := strings.Builder{}
	for i := range limits.MaxAlphabetRunes + 1 {
		long.WriteRune(rune('Ā' + i))
	}

	for name, tc := range map[string]struct {
		mapping  irv1.CharMapping
		alphabet string
		want     string
	}{
		"custom with an alphabet":      {irv1.CharMapping_CHAR_MAPPING_CUSTOM_ALPHABET, uscc, ""},
		"digits with no alphabet":      {irv1.CharMapping_CHAR_MAPPING_DIGIT_VALUE, "", ""},
		"custom without an alphabet":   {irv1.CharMapping_CHAR_MAPPING_CUSTOM_ALPHABET, "", "without an alphabet"},
		"an alphabet nothing reads":    {irv1.CharMapping_CHAR_MAPPING_DIGIT_VALUE, uscc, "only custom_alphabet reads"},
		"a repeated code point":        {irv1.CharMapping_CHAR_MAPPING_CUSTOM_ALPHABET, "01230", "twice"},
		"more code points than bound":  {irv1.CharMapping_CHAR_MAPPING_CUSTOM_ALPHABET, long.String(), "above the accepted"},
		"an alphabet that is not UTF8": {irv1.CharMapping_CHAR_MAPPING_CUSTOM_ALPHABET, "\xff\xfe", "not valid UTF-8"},
	} {
		t.Run(name, func(t *testing.T) {
			op := &irv1.IntegerOperation{
				Kind:    irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM,
				Mapping: tc.mapping.Enum(),
			}
			if tc.alphabet != "" {
				op.Alphabet = &tc.alphabet
			}
			err := checkAlphabet(op, fmt.Errorf)
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("expected acceptance, got %v", err)
			case tc.want == "":
				return
			case err == nil:
				t.Fatal("expected a refusal")
			case !strings.Contains(err.Error(), tc.want):
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}
