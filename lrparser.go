// Copyright 2018-2020 Petr Homola. All rights reserved.
// Use of this source code is governed by the AGPL v3.0
// that can be found in the LICENSE file.

// Package lrparser is an LR-parser.
package lrparser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/phomola/textkit"
)

// Rule is a context-free rule with a builder function.
type Rule struct {
	LHS  string
	RHS  []string
	Conv func([]interface{}) interface{}
}

// String returns a string representation of the rule.
func (r *Rule) String() string {
	return fmt.Sprintf("%s -> %v", r.LHS, r.RHS)
}

// Item is an item of the parser.
type Item struct {
	LHS    string
	RHS    []string
	DotPos int
}

func (it *Item) String() string {
	s := it.LHS + " ->"
	for i, el := range it.RHS {
		s += " "
		if it.DotPos == i {
			s += "*"
		}
		s += el
	}
	if it.DotPos == len(it.RHS) {
		s += "*"
	}
	return s + ";"
}

// State is a state of the parser.
type State struct {
	Items []*Item
}

func (st *State) String() string {
	sort.Slice(st.Items, func(i, j int) bool { return st.Items[i].String() < st.Items[j].String() })
	keys := make([]string, len(st.Items))
	for i, it := range st.Items {
		keys[i] = it.String()
	}
	return strings.Join(keys, " ")
}

type tableKey struct {
	row, column string
}

type action interface{}

type shiftAction struct {
	state string
}

type reduceAction struct {
	rule int
}

type acceptAction struct{}

type gotoAction struct {
	state string
}

// Located specifies methods for AST node location.
type Located interface {
	Location() textkit.Location
	SetLocation(textkit.Location)
}

// Grammar is a formal grammar.
type Grammar struct {
	// The rules of the grammar.
	Rules        []*Rule
	states       map[string]*State
	initialState string
	actionTable  map[tableKey]action
	gotoTable    map[tableKey]action
}

// BuildItems builds the items of the automaton.
func (gr *Grammar) BuildItems() {
	gr.states = make(map[string]*State)
	gr.actionTable = make(map[tableKey]action)
	gr.gotoTable = make(map[tableKey]action)
	rule := gr.Rules[0]
	acceptingItem := &Item{rule.LHS, rule.RHS, len(rule.RHS)}
	items := gr.closeItems([]*Item{&Item{rule.LHS, rule.RHS, 0}})
	state := &State{items}
	gr.initialState = state.String()
	states := []*State{state}
	for len(states) > 0 {
		state := states[0]
		states = states[1:]
		if _, ok := gr.states[state.String()]; !ok {
			gr.states[state.String()] = state
			tr := make(map[string]struct{})
			for _, it := range state.Items {
				if it.DotPos < len(it.RHS) {
					tr[it.RHS[it.DotPos]] = struct{}{}
				}
			}
			for symb := range tr {
				var items []*Item
				for _, it := range state.Items {
					if it.DotPos < len(it.RHS) && it.RHS[it.DotPos] == symb {
						items = append(items, &Item{it.LHS, it.RHS, it.DotPos + 1})
					}
				}
				items = gr.closeItems(items)
				state2 := &State{items}
				if symb[0] == '_' || symb[0] == '&' {
					gr.actionTable[tableKey{state.String(), symb}] = &shiftAction{state2.String()}
				} else {
					gr.gotoTable[tableKey{state.String(), symb}] = &gotoAction{state2.String()}
				}
				if _, ok := gr.states[state2.String()]; !ok {
					for _, it := range items {
						if it.String() == acceptingItem.String() {
							gr.actionTable[tableKey{state2.String(), "_EOF"}] = &acceptAction{}
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
	for _, state := range gr.states {
		for i, rule := range gr.Rules {
			if i > 0 {
				it := &Item{rule.LHS, rule.RHS, len(rule.RHS)}
				for _, it2 := range state.Items {
					if it.String() == it2.String() {
						for terminal := range terminals {
							if prevAction, ok := gr.actionTable[tableKey{state.String(), terminal}]; ok {
								if _, ok := prevAction.(*shiftAction); !ok {
									panic(fmt.Sprintf("conflict: %s %T %s", terminal, prevAction, prevAction))
								}
							} else {
								gr.actionTable[tableKey{state.String(), terminal}] = &reduceAction{i}
							}
						}
					}
				}
			}
		}
	}
	//fmt.Println("# states:", len(gr.states))
}

func (gr *Grammar) closeItems(items []*Item) []*Item {
	m := make(map[string]*Item, len(items))
	for _, it := range items {
		m[it.String()] = it
	}
	for len(items) > 0 {
		it := items[0]
		items = items[1:]
		if it.DotPos < len(it.RHS) {
			symb := it.RHS[it.DotPos]
			for _, rule := range gr.Rules {
				if rule.LHS == symb {
					it2 := &Item{rule.LHS, rule.RHS, 0}
					if _, ok := m[it2.String()]; !ok {
						m[it2.String()] = it2
						items = append(items, it2)
					}
				}
			}
		}
	}
	for _, it := range m {
		items = append(items, it)
	}
	return items
}

// Parse parses a list of tokens.
func (gr *Grammar) Parse(tokens []*textkit.Token) (interface{}, error) {
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
	stateStack := []string{gr.initialState}
	resultStack := []interface{}{}
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
			if r, ok := r.(Located); ok {
				var (
					loc textkit.Location
					set bool
				)
				for _, el := range results {
					switch x := el.(type) {
					case *textkit.Token:
						loc = x.Loc
						set = true
						goto setloc
					case Located:
						loc = x.Location()
						set = true
						goto setloc
					}
				}
			setloc:
				if set {
					r.SetLocation(loc)
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
					expected = append(expected, symbol)
				}
			}
			if len(expected) > 1 {
				return nil, fmt.Errorf("expected one of %s at line %s", strings.Join(expected, ", "), token.Loc)
			} else if len(expected) > 0 {
				return nil, fmt.Errorf("expected %s at line %s", expected[0], token.Loc)
			} else {
				return nil, fmt.Errorf("no expected symbol")
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
