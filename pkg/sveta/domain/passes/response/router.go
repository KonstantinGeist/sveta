package response

import "kgeyst.com/sveta/pkg/sveta/domain"

// some models are good at roleplay, others at reasoning, so we choose them dynamically
func (p *pass) getResponseServiceWithRoutedLanguageModel(context *domain.PassContext, inputMemory *domain.Memory) *domain.ResponseService {
	if !context.IsCapabilityEnabled(routerCapability) {
		return p.defaultResponseService
	}
	var output struct {
		ShortReasoning         string `json:"shortReasoning"`
		IsRoleplayOrCreativity bool   `json:"isRoleplayOrCreativity"`
	}
	err := p.getRouterResponseService().RespondToQueryWithJSON(
		"Does the following user query explicitly requires roleplay or creativity: \""+inputMemory.What+"\".",
		&output,
	)
	if err != nil {
		p.logger.Log("failed to route")
		return p.defaultResponseService
	}
	if output.IsRoleplayOrCreativity {
		p.logger.Log("\n\nROLEPLAY model selected\n\n")
		return p.roleplayResponseService
	}
	p.logger.Log("\n\nDEFAULT model selected\n\n")
	return p.defaultResponseService
}

func (p *pass) getRouterResponseService() *domain.ResponseService {
	routerAIContext := domain.NewAIContext("RouterLLM", "You're RouterLLM, an intelligent assistant which tells if a given user query requires roleplay or creativity. Only two existing scenarios are allowed: user asks to imagine a story, or explicitly asks to roleplay.", "")
	return p.defaultResponseService.WithAIContext(routerAIContext)
}
