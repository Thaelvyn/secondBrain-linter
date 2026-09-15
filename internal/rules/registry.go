package rules

// Rule is a single ADR 0004 rule implementation.
type Rule struct {
	Num int
	Run func(*Context)
}

// Rules is the registry, applied in rule-number order.
var Rules = []Rule{
	{Num: 1, Run: rule01},
	{Num: 2, Run: rule02},
	{Num: 3, Run: rule03},
	{Num: 4, Run: rule04},
	{Num: 5, Run: rule05},
	{Num: 6, Run: rule06},
	{Num: 7, Run: rule07},
	{Num: 8, Run: rule08},
	{Num: 9, Run: rule09},
	{Num: 10, Run: rule10},
	{Num: 11, Run: rule11},
	{Num: 12, Run: rule12},
	{Num: 13, Run: rule13},
	{Num: 15, Run: rule15},
	{Num: 16, Run: rule16},
	{Num: 17, Run: rule17},
	{Num: 18, Run: rule18},
	{Num: 19, Run: rule19},
	{Num: 20, Run: rule20},
	{Num: 21, Run: rule21},
	{Num: 22, Run: rule22},
	{Num: 23, Run: rule23},
	{Num: 24, Run: rule24},
	{Num: 25, Run: rule25},
}

// Run executes every registered rule against the context.
func Run(c *Context) {
	for _, r := range Rules {
		r.Run(c)
	}
}
