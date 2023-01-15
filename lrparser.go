// Copyright 2018-2020 Petr Homola. All rights reserved.
// Use of this source code is governed by the AGPL v3.0
// that can be found in the LICENSE file.

// An LR-parser.
package lrparser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/phomola/textkit"
)

// A context-free rule with a builder function.
type Rule struct {
	Lhs  string
	Rhs  []string
	Conv func([]interface{}) interface{}
}

// Returns a string representation of the rule.
func (r *Rule) String() string {
	return fmt.Sprintf("%s -> %v", r.Lhs, r.Rhs)
}

// An item of the parser.
type Item struct {
	Lhs    string
	Rhs    []string
	DotPos int
}

func (it *Item) String() string {
	s := it.Lhs + " ->"
	for i, el := range it.Rhs {
		s += " "
		if it.DotPos == i {
			s += "*"
		}
		s += el
	}
	if it.DotPos == len(it.Rhs) {
		s += "*"
	}
	return s + ";"
}

// A state of the parser.
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

// A formal grammar.
type Grammar struct {
	// The rules of the grammar.
	Rules        []*Rule
	states       map[string]*State
	initialState string
	actionTable  map[tableKey]action
	gotoTable    map[tableKey]action
}

// Builds the items of the automaton.
func (gr *Grammar) BuildItems() {
	gr.states = make(map[string]*State)
	gr.actionTable = make(map[tableKey]action)
	gr.gotoTable = make(map[tableKey]action)
	rule := gr.Rules[0]
	acceptingItem := &Item{rule.Lhs, rule.Rhs, len(rule.Rhs)}
	items := gr.closeItems([]*Item{&Item{rule.Lhs, rule.Rhs, 0}})
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
				if it.DotPos < len(it.Rhs) {
					tr[it.Rhs[it.DotPos]] = struct{}{}
				}
			}
			for symb, _ := range tr {
				var items []*Item
				for _, it := range state.Items {
					if it.DotPos < len(it.Rhs) && it.Rhs[it.DotPos] == symb {
						items = append(items, &Item{it.Lhs, it.Rhs, it.DotPos + 1})
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
	for key, _ := range gr.actionTable {
		terminals[key.column] = struct{}{}
	}
	for _, state := range gr.states {
		for i, rule := range gr.Rules {
			if i > 0 {
				it := &Item{rule.Lhs, rule.Rhs, len(rule.Rhs)}
				for _, it2 := range state.Items {
					if it.String() == it2.String() {
						for terminal, _ := range terminals {
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
		if it.DotPos < len(it.Rhs) {
			symb := it.Rhs[it.DotPos]
			for _, rule := range gr.Rules {
				if rule.Lhs == symb {
					it2 := &Item{rule.Lhs, rule.Rhs, 0}
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

// Parses a list of tokens.
func (gr *Grammar) Parse(tokens []*textkit.Token) (interface{}, error) {
	terminals := make(map[string]struct{})
	for key, _ := range gr.actionTable {
		terminals[key.column] = struct{}{}
	}
	keywords := make(map[string]struct{})
	for key, _ := range gr.actionTable {
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
			results := resultStack[len(resultStack)-len(rule.Rhs):]
			resultStack = resultStack[: len(resultStack)-len(rule.Rhs) : len(resultStack)-len(rule.Rhs)]
			stateStack = stateStack[:len(stateStack)-len(rule.Rhs)]
			resultStack = append(resultStack, rule.Conv(results))
			if nextState, ok := gr.gotoTable[tableKey{stateStack[len(stateStack)-1], rule.Lhs}]; ok {
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
			for terminal, _ := range terminals {
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
					expected = append(expected, symbol)
				}
			}
			if len(expected) > 1 {
				return nil, fmt.Errorf("expected one of %s at line %d", strings.Join(expected, ", "), token.Line)
			} else if len(expected) > 0 {
				return nil, fmt.Errorf("expected %s at line %d", expected[0], token.Line)
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

// Returns a new grammar.
func NewGrammar(rules ...[]*Rule) *Grammar {
	var allRules []*Rule
	for _, r := range rules {
		allRules = append(allRules, r...)
	}
	gr := &Grammar{Rules: allRules}
	gr.BuildItems()
	return gr
}
