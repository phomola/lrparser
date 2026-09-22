// Copyright 2018-2020 Petr Homola. All rights reserved.
// Use of this source code is governed by the AGPL v3.0
// that can be found in the LICENSE file.

// Package lrparser is an LR-parser.
package lrparser

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/phomola/gomisc/list"
	"github.com/phomola/textkit"
)

// Rule is a context-free rule with a builder function.
type Rule struct {
	LHS     string
	RHS     []string
	Conv    func([]any) any
	rhsList list.List[string]
}

// NewRule creates a new rule.
func NewRule(lhs string, rhs []string, conv func([]any) any) *Rule {
	return &Rule{LHS: lhs, RHS: rhs, Conv: conv}
}

func (r *Rule) rhsAsList() list.List[string] {
	if r.rhsList.IsEmpty() {
		r.rhsList = list.FromSlice(r.RHS)
	}
	return r.rhsList
}

// String returns a string representation of the rule.
func (r *Rule) String() string {
	return fmt.Sprintf("%s -> %v", r.LHS, r.RHS)
}

var (
	ruleTokeniser   *textkit.Tokeniser
	ruleGrammar     *Grammar
	ruleGrammarOnce sync.Once
)

type ruleDef struct {
	LHS string
	RHS []string
}

// MustBuildRule builds a grammar rule from a string. It panics on error.
func MustBuildRule(def string, f func([]any) any) *Rule {
	r, err := BuildRule(def, f)
	if err != nil {
		panic(err)
	}
	return r
}

// BuildRule builds a grammar rule from a string.
func BuildRule(def string, f func([]any) any) (*Rule, error) {
	ruleGrammarOnce.Do(func() {
		ruleTokeniser = &textkit.Tokeniser{
			StringRune: '"',
		}
		ruleGrammar = &Grammar{Rules: []*Rule{
			{LHS: "Init", RHS: []string{"Rule"}, Conv: func(args []any) any {
				return args[0]
			}},
			{LHS: "Rule", RHS: []string{"_ID", "&->", "Symbols"}, Conv: func(args []any) any {
				return &ruleDef{LHS: string(args[0].(*textkit.Token).Form), RHS: args[2].([]string)}
			}},
			{LHS: "Symbols", RHS: []string{"Symbols", "Symbol"}, Conv: func(args []any) any {
				return append(args[0].([]string), args[1].(string))
			}},
			{LHS: "Symbols", RHS: []string{"Symbol"}, Conv: func(args []any) any {
				return []string{args[0].(string)}
			}},
			{LHS: "Symbol", RHS: []string{"&string"}, Conv: func(args []any) any {
				return "_STR"
			}},
			{LHS: "Symbol", RHS: []string{"&integer"}, Conv: func(args []any) any {
				return "_NUM"
			}},
			{LHS: "Symbol", RHS: []string{"&ident"}, Conv: func(args []any) any {
				return "_ID"
			}},
			{LHS: "Symbol", RHS: []string{"&eol"}, Conv: func(args []any) any {
				return "_EOL"
			}},
			{LHS: "Symbol", RHS: []string{"&end"}, Conv: func(args []any) any {
				return "_END"
			}},
			{LHS: "Symbol", RHS: []string{"&comment"}, Conv: func(args []any) any {
				return "_COMM"
			}},
			{LHS: "Symbol", RHS: []string{"_ID"}, Conv: func(args []any) any {
				return string(args[0].(*textkit.Token).Form)
			}},
			{LHS: "Symbol", RHS: []string{"_STR"}, Conv: func(args []any) any {
				return "&" + string(args[0].(*textkit.Token).Form)
			}},
		}}
		ruleGrammar.BuildItems()
	})
	tokens := ruleTokeniser.Tokenise(def, "grammar")
	tokens = CoalesceSymbols(tokens, []string{"->"})
	r, err := ruleGrammar.Parse(tokens)
	if err != nil {
		return nil, err
	}
	rule := r.(*ruleDef)
	return &Rule{
		LHS: rule.LHS,
		RHS: rule.RHS,
		Conv: func(args []any) any {
			r := make([]any, len(args))
			for i, arg := range args {
				switch x := arg.(type) {
				case *textkit.Token:
					switch x.Type {
					case textkit.Word, textkit.Symbol, textkit.String:
						r[i] = string(x.Form)
					case textkit.Number:
						n, _ := strconv.Atoi(string(x.Form))
						r[i] = n
					}
				default:
					r[i] = arg
				}
			}
			return f(r)
		},
	}, nil
}

// Item is an item of the parser.
type Item struct {
	LHS    string
	RHS    list.List[string]
	DotPos int
}

// Less compares the two items.
func (it Item) Less(it2 Item) bool {
	if it.LHS < it2.LHS {
		return true
	}
	if it.LHS > it2.LHS {
		return false
	}
	if it.DotPos < it2.DotPos {
		return true
	}
	if it.DotPos > it2.DotPos {
		return false
	}
	if it.RHS.Len() < it2.RHS.Len() {
		return true
	}
	if it.RHS.Len() > it2.RHS.Len() {
		return false
	}
	for i, x := range it.RHS.EnumIndexed() {
		y := it2.RHS.At(i)
		if x < y {
			return true
		}
		if x > y {
			return false
		}
	}
	return false
}

// func (it *Item) String() string {
// 	var s strings.Builder
// 	s.WriteString(it.LHS + " ->")
// 	for i, el := range it.RHS.EnumIndexed() {
// 		s.WriteString(" ")
// 		if it.DotPos == i {
// 			s.WriteString("*")
// 		}
// 		s.WriteString(el)
// 	}
// 	if it.DotPos == it.RHS.Len() {
// 		s.WriteString("*")
// 	}
// 	return s.String() + ";"
// }

// State is a state of the parser.
type State struct {
	Items list.List[Item]
}

// func (st *State) String() string {
// 	// sort.Slice(st.Items, func(i, j int) bool { return st.Items[i].String() < st.Items[j].String() })
// 	keys := make([]string, st.Items.Len())
// 	for i, it := range st.Items.EnumIndexed() {
// 		keys[i] = it.String()
// 	}
// 	return strings.Join(keys, " ")
// }

type tableKey struct {
	row    list.List[Item]
	column string
}

type action any

type shiftAction struct {
	state list.List[Item]
}

type reduceAction struct {
	rule int
}

type acceptAction struct{}

type gotoAction struct {
	state list.List[Item]
}

// Located specifies methods for AST node location.
type Located interface {
	Location() *textkit.Location
	SetLocation(*textkit.Location)
}

// Grammar is a formal grammar.
type Grammar struct {
	// The rules of the grammar.
	Rules             []*Rule
	ErrorOnNonlocated bool
	// states            map[list.List[Item]]*State
	initialState list.List[Item]
	actionTable  map[tableKey]action
	gotoTable    map[tableKey]action
}

// BuildItems builds the items of the automaton.
func (gr *Grammar) BuildItems() {
	statesMap := make(map[list.List[Item]]*State)
	gr.actionTable = make(map[tableKey]action)
	gr.gotoTable = make(map[tableKey]action)
	rule := gr.Rules[0]
	rhsList := rule.rhsAsList()
	acceptingItem := Item{rule.LHS, rhsList, len(rule.RHS)}
	items := gr.closeItems([]Item{{rule.LHS, rhsList, 0}})
	state := &State{list.FromSlice(items).Sorted(Item.Less)}
	gr.initialState = state.Items
	states := []*State{state}
	for len(states) > 0 {
		state := states[0]
		states = states[1:]
		if _, ok := statesMap[state.Items]; !ok {
			statesMap[state.Items] = state
			tr := make(map[string]struct{})
			for it := range state.Items.Enum() {
				if it.DotPos < it.RHS.Len() {
					tr[it.RHS.At(it.DotPos)] = struct{}{}
				}
			}
			for symb := range tr {
				var items []Item
				for it := range state.Items.Enum() {
					if it.DotPos < it.RHS.Len() && it.RHS.At(it.DotPos) == symb {
						items = append(items, Item{it.LHS, it.RHS, it.DotPos + 1})
					}
				}
				items = gr.closeItems(items)
				state2 := &State{list.FromSlice(items).Sorted(Item.Less)}
				if symb[0] == '_' || symb[0] == '&' {
					gr.actionTable[tableKey{state.Items, symb}] = &shiftAction{state2.Items}
				} else {
					gr.gotoTable[tableKey{state.Items, symb}] = &gotoAction{state2.Items}
				}
				if _, ok := statesMap[state2.Items]; !ok {
					for _, it := range items {
						if it == acceptingItem {
							gr.actionTable[tableKey{state2.Items, "_EOF"}] = &acceptAction{}
						}
					}
					states = append(states, state2)
				}
			}
		}
	}
	terminals := make(map[string]struct{})
	for key := range gr.actionTable {
		terminals[key.column] = struct{}{}
	}
	for _, state := range statesMap {
		for i, rule := range gr.Rules {
			if i > 0 {
				it := Item{rule.LHS, rule.rhsAsList(), len(rule.RHS)}
				for it2 := range state.Items.Enum() {
					if it == it2 {
						for terminal := range terminals {
							if prevAction, ok := gr.actionTable[tableKey{state.Items, terminal}]; ok {
								if _, ok := prevAction.(*shiftAction); !ok {
									panic(fmt.Sprintf("conflict: %s %T %s", terminal, prevAction, prevAction))
								}
							} else {
								gr.actionTable[tableKey{state.Items, terminal}] = &reduceAction{i}
							}
						}
					}
				}
			}
		}
	}
	//fmt.Println("# states:", len(gr.states))
}

func (gr *Grammar) closeItems(items []Item) []Item {
	m := make(map[Item]struct{}, len(items))
	for _, it := range items {
		m[it] = struct{}{}
	}
	for len(items) > 0 {
		it := items[0]
		items = items[1:]
		if it.DotPos < it.RHS.Len() {
			symb := it.RHS.At(it.DotPos)
			for _, rule := range gr.Rules {
				if rule.LHS == symb {
					it2 := Item{rule.LHS, rule.rhsAsList(), 0}
					if _, ok := m[it2]; !ok {
						m[it2] = struct{}{}
						items = append(items, it2)
					}
				}
			}
		}
	}
	for it := range m {
		items = append(items, it)
	}
	return items
}

// Parse parses a list of tokens.
func (gr *Grammar) Parse(tokens []*textkit.Token) (any, error) {
	terminals := make(map[string]struct{})
	for key := range gr.actionTable {
		terminals[key.column] = struct{}{}
	}
	keywords := make(map[string]struct{})
	for key := range gr.actionTable {
		if key.column[0] == '&' {
			keywords[key.column[1:]] = struct{}{}
		}
	}
	stateStack := []list.List[Item]{gr.initialState}
	resultStack := []any{}
	for {
		token := tokens[0]
		var symb string
		switch token.Type {
		case textkit.Symbol:
			symb = "&" + string(token.Form)
		case textkit.Number:
			symb = "_NUM"
		case textkit.String:
			symb = "_STR"
		case textkit.EOF:
			symb = "_EOF"
		case textkit.EOL:
			symb = "_EOL"
		case textkit.EndIndent:
			symb = "_END"
		case textkit.Comment:
			symb = "_COMM"
		case textkit.Word:
			if _, ok := keywords[string(token.Form)]; ok {
				symb = "&" + string(token.Form)
			} else {
				symb = "_ID"
			}
		}
		currentState := stateStack[len(stateStack)-1]
		action := gr.actionTable[tableKey{currentState, symb}]
		switch action := action.(type) {
		case *shiftAction:
			resultStack = append(resultStack, token)
			stateStack = append(stateStack, action.state)
			tokens = tokens[1:]
			//fmt.Println("SHIFT", currentState, "/", symb, "=>", action.state)
		case *reduceAction:
			rule := gr.Rules[action.rule]
			results := resultStack[len(resultStack)-len(rule.RHS):]
			resultStack = resultStack[: len(resultStack)-len(rule.RHS) : len(resultStack)-len(rule.RHS)]
			stateStack = stateStack[:len(stateStack)-len(rule.RHS)]
			r := rule.Conv(results)
			if r2, ok := r.(Located); ok {
				var loc *textkit.Location
				for _, el := range results {
					switch x := el.(type) {
					case *textkit.Token:
						loc = x.Loc
						goto setloc
					case Located:
						loc = x.Location()
						goto setloc
					}
				}
			setloc:
				if loc != nil {
					r2.SetLocation(loc)
				}
			} else {
				if gr.ErrorOnNonlocated {
					return nil, fmt.Errorf("'%T' doesn't conform to lrparser.Located", r)
				}
			}
			resultStack = append(resultStack, r)
			if nextState, ok := gr.gotoTable[tableKey{stateStack[len(stateStack)-1], rule.LHS}]; ok {
				//fmt.Println("REDUCE", len(stateStack), len(results), currentState, "/", symb, "=>", nextState)
				stateStack = append(stateStack, nextState.(*gotoAction).state)
			} else {
				panic("can't reduce")
			}
		case *acceptAction:
			//fmt.Println("ACCEPT", len(stateStack), len(resultStack))
			return resultStack[0], nil
		default:
			var expected []string
			for terminal := range terminals {
				if _, ok := gr.actionTable[tableKey{currentState, terminal}]; ok {
					symbol := terminal
					if terminal[0] == '&' {
						symbol = "'" + terminal[1:] + "'"
					}
					if terminal == "_ID" {
						symbol = "identifier"
					}
					if terminal == "_STR" {
						symbol = "string"
					}
					if terminal == "_NUM" {
						symbol = "number"
					}
					if terminal == "_EOF" {
						symbol = "EOF"
					}
					if terminal == "_EOL" {
						symbol = "EOL"
					}
					if terminal == "_END" {
						symbol = "END"
					}
					if terminal == "_COMM" {
						symbol = "comment"
					}
					expected = append(expected, symbol)
				}
			}
			if len(expected) > 1 {
				return nil, &ParseError{Message: fmt.Sprintf("expected one of %s", strings.Join(expected, ", ")), Loc: token.Loc}
			} else if len(expected) > 0 {
				return nil, &ParseError{Message: fmt.Sprintf("expected %s", expected[0]), Loc: token.Loc}
			} else {
				return nil, &ParseError{Message: "no expected symbol"}
			}
			/*for terminal, _ := range terminals {
				if _, ok := gr.actionTable[tableKey{currentState, terminal}]; ok {
					expected = append(expected, terminal)
				}
			}
			return nil, fmt.Errorf("expected '%s' at line %d", strings.Join(expected, "|"), token.Line)*/
		}
	}
}

// ParseError is a parse error.
type ParseError struct {
	Message string
	Loc     *textkit.Location
}

func (err *ParseError) Error() string {
	if err.Loc != nil {
		return err.Message + " at " + err.Loc.String()
	}
	return err.Message
}

// NewGrammar returns a new grammar.
func NewGrammar(rules ...[]*Rule) *Grammar {
	var allRules []*Rule
	for _, r := range rules {
		allRules = append(allRules, r...)
	}
	gr := &Grammar{Rules: allRules}
	gr.BuildItems()
	return gr
}
