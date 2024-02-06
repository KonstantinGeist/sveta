package domain

type AIFilterContext struct {
	Who   string
	What  string
	Where string
}

func NewAIFilterContext(who, what, where string) AIFilterContext {
	return AIFilterContext{
		Who:   who,
		What:  what,
		Where: where,
	}
}

func (a AIFilterContext) WithWhat(what string) AIFilterContext {
	a.What = what
	return a
}
