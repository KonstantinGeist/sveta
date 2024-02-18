package mask

import "kgeyst.com/sveta/pkg/sveta/domain"

const maskCapability = "mask"

type pass struct{}

func NewPass() domain.Pass {
	return &pass{}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        maskCapability,
			Description: "determines which passes need to run",
			IsMaskable:  false,
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	maskableCapabilities := make([]*domain.Capability, 0, len(context.EnabledCapabilities))
	for _, capability := range context.EnabledCapabilities {
		if capability.IsMaskable {
			maskableCapabilities = append(maskableCapabilities, capability)
		}
	}
	return nextPassFunc(context)
}
