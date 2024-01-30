package domain

type NextFilterFunc func(who, what, where string) (string, error)

type AIFilter interface {
	Apply(who, what, where string, nextFilterFunc NextFilterFunc) (string, error)
}
