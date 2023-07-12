# lrparser
An LR parser

Example:
```
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
```
See [`cmd/example`](https://github.com/phomola/lrparser/blob/master/cmd/example/main.go)
