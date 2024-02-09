package domain

type SummaryRepository interface {
	FindByWhere(where string) (*string, error)
	Store(where, summary string) error
	RemoveAll() error
}
