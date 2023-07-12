// Copyright 2018-2020 Petr Homola. All rights reserved.
// Use of this source code is governed by the AGPL v3.0
// that can be found in the LICENSE file.

package lrparser

import (
	"fmt"
	"sort"

	"github.com/phomola/textkit"
)

// CoalesceSymbols joins symbols together.
func CoalesceSymbols(tokens []*textkit.Token, clusters []string) []*textkit.Token {
	m := make(map[string][]string)
	for _, c := range clusters {
		m[c[:1]] = append(m[c[:1]], c)
	}
	var tokens2 []*textkit.Token
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if t.Type == textkit.Symbol {
			if cs, ok := m[string(t.Form)]; ok {
				for _, c := range cs {
					if len(c) <= len(tokens)-i {
						for j := 1; j < len(c); j++ {
							t2 := tokens[i+j]
							if (t2.Type != textkit.Symbol && t2.Type != textkit.Word) || string(t2.Form) != c[j:j+1] {
								break
							}
							if j+1 == len(c) {
								i += len(c) - 1
								tokens2 = append(tokens2, &textkit.Token{
									Type: textkit.Symbol,
									Form: []rune(c),
									Loc:  t.Loc,
									Tag:  "",
								})
								goto cont
							}
						}
					}
				}
			}
		}
		tokens2 = append(tokens2, t)
	cont:
	}
	return tokens2
}

// BuildOptSeq builds an optional sequence.
func BuildOptSeq(root string, head, tail []string, builder func([]interface{}, []interface{}) interface{}) []*Rule {
	var rules []*Rule
	rules = append(rules, &Rule{root, head, func(r []interface{}) interface{} { return builder(r, nil) }})
	rules = append(rules, &Rule{root, append(head, tail...), func(r []interface{}) interface{} { return builder(r[:len(head)], r[len(head):]) }})
	return rules
}

// BuildListRules builds list rules.
func BuildListRules(root, leaf string, canBeEmpty bool, leftBracket, sep, rightBracket string, builder func([]interface{}) interface{}) []*Rule {
	var rules []*Rule
	var symbols []string
	index := 0
	if leftBracket != "" {
		symbols = append(symbols, leftBracket)
		index++
	}
	symbols = append(symbols, root+"Els")
	if rightBracket != "" {
		symbols = append(symbols, rightBracket)
	}
	rules = append(rules, &Rule{root, symbols, func(r []interface{}) interface{} { return builder(r[index].([]interface{})) }})
	if canBeEmpty {
		rules = append(rules, &Rule{root, []string{leftBracket, rightBracket}, func(r []interface{}) interface{} { return builder(nil) }})
	}
	rules = append(rules, &Rule{root + "Els", []string{leaf}, func(r []interface{}) interface{} { return []interface{}{r[0]} }})
	if sep != "" {
		rules = append(rules, &Rule{root + "Els", []string{root + "Els", sep, leaf}, func(r []interface{}) interface{} { return append(r[0].([]interface{}), r[2]) }})
	} else {
		rules = append(rules, &Rule{root + "Els", []string{root + "Els", leaf}, func(r []interface{}) interface{} { return append(r[0].([]interface{}), r[1]) }})
	}
	return rules
}

// OperatorAssociativity ...
type OperatorAssociativity int

// constants for OperatorAssociativity
const (
	LeftAssociative OperatorAssociativity = iota
	RightAssociative
	NonAssociative
)

// Operator ...
type Operator struct {
	Associativity OperatorAssociativity
	Priority      int
	Symbols       []string
}

// Name returns the operator's name.
func (op Operator) Name() string {
	var str string
	for _, sym := range op.Symbols {
		str += sym[1:]
	}
	return str
}

// BuildOperatorRules builds operator rules.
func BuildOperatorRules(root, leaf string, ops []Operator, builder func(string, interface{}, interface{}) interface{}) []*Rule {
	opMap := make(map[int][]Operator)
	for _, op := range ops {
		opMap[op.Priority] = append(opMap[op.Priority], op)
	}
	var prios []int
	for p := range opMap {
		prios = append(prios, p)
	}
	sort.Slice(prios, func(i, j int) bool { return i < j })
	rules := []*Rule{&Rule{root, []string{fmt.Sprintf("%sOp%d", root, prios[0])}, func(r []interface{}) interface{} { return r[0] }}}
	for i, prio := range prios {
		sym1 := fmt.Sprintf("%sOp%d", root, prio)
		var sym2 string
		if i+1 == len(prios) {
			sym2 = leaf
		} else {
			sym2 = fmt.Sprintf("%sOp%d", root, prios[i+1])
		}
		for _, op := range opMap[prio] {
			var symbols []string
			if op.Associativity == LeftAssociative {
				symbols = append(symbols, sym1)
			} else {
				symbols = append(symbols, sym2)
			}
			symbols = append(symbols, op.Symbols...)
			if op.Associativity == RightAssociative {
				symbols = append(symbols, sym1)
			} else {
				symbols = append(symbols, sym2)
			}
			rules2 := []*Rule{
				&Rule{sym1, symbols, func(r []interface{}) interface{} {
					return builder(op.Name(), r[0], r[len(r)-1])
				}},
				&Rule{sym1, []string{sym2}, func(r []interface{}) interface{} { return r[0] }},
			}
			rules = append(rules, rules2...)
		}
	}
	return rules
}

// SynSem is a syntactic rule with its semantic counterpart (a lambda function).
type SynSem struct {
	Syn string
	Sem func([]any) any
}

// BuildRules creates rules from a slice of `SynSem`s.
func BuildRules(list []*SynSem) ([]*Rule, error) {
	rules := make([]*Rule, len(list))
	for i, el := range list {
		r, err := BuildRule(el.Syn, el.Sem)
		if err != nil {
			return nil, err
		}
		rules[i] = r
	}
	return rules, nil
}

// MustBuildRules creates rules from a slice of `SynSem`s. It panics on error.
func MustBuildRules(list []*SynSem) []*Rule {
	r, err := BuildRules(list)
	if err != nil {
		panic(err)
	}
	return r
}
