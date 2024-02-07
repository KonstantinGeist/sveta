package bio

type Provider interface {
	GetBioFacts() ([]string, error)
}
