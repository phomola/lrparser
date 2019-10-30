// Copyright 2018-2019 Petr Homola. All rights reserved.
// Use of this source code is governed by the AGPL v3.0
// that can be found in the LICENSE file.

package lrparser

import (
	"fmt"
	"sort"
	"strings"
)

type OperatorAssociativiy int

const (
	LeftAssociative OperatorAssociativiy = iota
	RightAssociative
	NonAssociative
)

type Operator struct {
	Associativity OperatorAssociativiy
	Priority      int
	Symbols       []string
}

type BinaryOperator struct {
	Op    string
	Left  interface{}
	Right interface{}
}

func (op BinaryOperator) String() string {
	return fmt.Sprintf("(%v)%s(%v)", op.Left, op.Op, op.Right)
}

func BuildOperatorRules(root, leaf string, ops []Operator) []*Rule {
	opMap := make(map[int][]Operator)
	for _, op := range ops {
		opMap[op.Priority] = append(opMap[op.Priority], op)
	}
	var prios []int
	for p, _ := range opMap {
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
					return BinaryOperator{strings.Join(op.Symbols, "")[1:], r[0], r[len(r)-1]}
				}},
				&Rule{sym1, []string{sym2}, func(r []interface{}) interface{} { return r[0] }},
			}
			rules = append(rules, rules2...)
		}
	}
	return rules
}
