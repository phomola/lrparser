# lrparser
An LR parser
```
lrparser.MustBuildRule(`Init ->Expr`, func(args []any) any { return args[0] }),
lrparser.MustBuildRule(`Expr -> "expr" AddExpr`, func(args []any) any { return args[1] }),
lrparser.MustBuildRule(`AddExpr -> AddExpr "+" MulExpr`, func(args []any) any { return args[0].(int) + args[2].(int) }),
lrparser.MustBuildRule(`AddExpr -> AddExpr "-" MulExpr`, func(args []any) any { return args[0].(int) - args[2].(int) }),
lrparser.MustBuildRule(`AddExpr -> MulExpr`, func(args []any) any { return args[0] }),
lrparser.MustBuildRule(`MulExpr -> MulExpr "*" ConstExpr`, func(args []any) any { return args[0].(int) * args[2].(int) }),
lrparser.MustBuildRule(`MulExpr -> MulExpr "/" ConstExpr`, func(args []any) any { return args[0].(int) / args[2].(int) }),
lrparser.MustBuildRule(`MulExpr -> ConstExpr`, func(args []any) any { return args[0] }),
lrparser.MustBuildRule(`ConstExpr -> integer`, func(args []any) any { return args[0] }),
```
