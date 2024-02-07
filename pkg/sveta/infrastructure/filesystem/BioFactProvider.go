package filesystem

import "kgeyst.com/sveta/pkg/common"

type BioFactProvider struct {
	config           *common.Config
	bioFactsFilePath string
}

func NewBioFactProvider(config *common.Config) *BioFactProvider {
	return &BioFactProvider{
		config:           config,
		bioFactsFilePath: config.GetString("bioFactsFilePath"),
	}
}

func (b *BioFactProvider) GetBioFacts() ([]string, error) {
	if b.bioFactsFilePath == "" {
		return nil, nil
	}
	return common.ReadAllLines(b.bioFactsFilePath)
}
