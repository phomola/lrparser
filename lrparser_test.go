package lrparser

import (
	"testing"

	"github.com/phomola/textkit"
	"github.com/stretchr/testify/require"
)

func TestParsing(t *testing.T) {
	req := require.New(t)

	rules := []*Rule{
		NewRule("Init", []string{"List"}, func(r []any) any { return r[0] }),
		NewRule("List", []string{"List", "Elem"}, func(r []any) any { return append(r[0].([]string), r[1].(string)) }),
		NewRule("List", []string{"Elem"}, func(r []any) any { return []string{r[0].(string)} }),
		NewRule("Elem", []string{"_ID"}, func(r []any) any { return string(r[0].(*textkit.Token).Form) }),
		NewRule("Elem", []string{"_STR"}, func(r []any) any { return string(r[0].(*textkit.Token).Form) }),
		NewRule("Elem", []string{"_NUM"}, func(r []any) any { return string(r[0].(*textkit.Token).Form) }),
	}
	grammar := NewGrammar(rules)
	grammar.BuildItems()
	var tok textkit.Tokeniser
	tokens := tok.Tokenise("abcd 1234", "")
	r, err := grammar.Parse(tokens)
	req.Nil(err)
	req.Equal([]string{"abcd", "1234"}, r)
}

func BenchmarkBuildingParser(b *testing.B) {
	rules := []*Rule{
		NewRule("Init", []string{"List"}, func(r []any) any { return r[0] }),
		NewRule("List", []string{"List", "Elem"}, func(r []any) any { return append(r[0].([]string), r[1].(string)) }),
		NewRule("List", []string{"Elem"}, func(r []any) any { return []string{r[0].(string)} }),
		NewRule("Elem", []string{"_ID"}, func(r []any) any { return string(r[0].(*textkit.Token).Form) }),
		NewRule("Elem", []string{"_STR"}, func(r []any) any { return string(r[0].(*textkit.Token).Form) }),
		NewRule("Elem", []string{"_NUM"}, func(r []any) any { return string(r[0].(*textkit.Token).Form) }),
	}
	b.ResetTimer()
	for b.Loop() {
		grammar := NewGrammar(rules)
		grammar.BuildItems()
	}
}

func BenchmarkParsing(b *testing.B) {
	rules := []*Rule{
		NewRule("Init", []string{"List"}, func(r []any) any { return r[0] }),
		NewRule("List", []string{"List", "Elem"}, func(r []any) any { return append(r[0].([]string), r[1].(string)) }),
		NewRule("List", []string{"Elem"}, func(r []any) any { return []string{r[0].(string)} }),
		NewRule("Elem", []string{"_ID"}, func(r []any) any { return string(r[0].(*textkit.Token).Form) }),
		NewRule("Elem", []string{"_STR"}, func(r []any) any { return string(r[0].(*textkit.Token).Form) }),
		NewRule("Elem", []string{"_NUM"}, func(r []any) any { return string(r[0].(*textkit.Token).Form) }),
	}
	grammar := NewGrammar(rules)
	grammar.BuildItems()
	var tok textkit.Tokeniser
	tokens := tok.Tokenise("abcd 1234", "")
	b.ResetTimer()
	for b.Loop() {
		if _, err := grammar.Parse(tokens); err != nil {
			b.Error(err)
		}
	}
}
