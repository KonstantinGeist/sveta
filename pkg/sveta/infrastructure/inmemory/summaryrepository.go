package inmemory

import "sync"

type SummaryRepository struct {
	mutex     sync.Mutex
	summaries map[string]string // where => summary
}

func NewSummaryRepository() *SummaryRepository {
	return &SummaryRepository{
		summaries: make(map[string]string),
	}
}

func (s *SummaryRepository) FindByWhere(where string) (*string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	summary, ok := s.summaries[where]
	if !ok {
		return nil, nil
	}
	return &summary, nil
}

func (s *SummaryRepository) Store(where, summary string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.summaries[where] = summary
	return nil
}
