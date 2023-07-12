package main

import (
	"fmt"

	"github.com/phomola/lrparser"
	"github.com/phomola/textkit"
)

func main() {
	gr := lrparser.NewGrammar(lrparser.MustBuildRules([]*lrparser.SynSem{
		{Syn: `Init -> Expr`, Sem: func(args []any) any { return args[0] }},
		{Syn: `Expr -> "expr" AddExpr`, Sem: func(args []any) any { return args[1] }},
		{Syn: `AddExpr -> AddExpr "+" MulExpr`, Sem: func(args []any) any { return args[0].(int) + args[2].(int) }},
		{Syn: `AddExpr -> AddExpr "-" MulExpr`, Sem: func(args []any) any { return args[0].(int) - args[2].(int) }},
		{Syn: `AddExpr -> MulExpr`, Sem: func(args []any) any { return args[0] }},
		{Syn: `MulExpr -> MulExpr "*" ConstExpr`, Sem: func(args []any) any { return args[0].(int) * args[2].(int) }},
		{Syn: `MulExpr -> MulExpr "/" ConstExpr`, Sem: func(args []any) any { return args[0].(int) / args[2].(int) }},
		{Syn: `MulExpr -> ConstExpr`, Sem: func(args []any) any { return args[0] }},
		{Syn: `ConstExpr -> integer`, Sem: func(args []any) any { return args[0] }},
	}))
	tok := new(textkit.Tokeniser)
	tokens := tok.Tokenise(`expr 2+3*4`, "")
	r, err := gr.Parse(tokens)
	if err != nil {
		panic(err)
	}
	fmt.Println(r)
}
