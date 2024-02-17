package gocalculator

import (
	"fmt"
	"strconv"

	"github.com/mnogu/go-calculator"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/api"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

func RegisterCalcFunction(sveta api.API) error {
	return sveta.RegisterFunction(api.FunctionDesc{
		Name:        "calc",
		Description: "allows to calculate a math expression if the user query requires it",
		Parameters: []domain.FunctionParameterDesc{
			{
				Name:        "mathExpression",
				Description: "math expression, such as (2 + 3 * 4)",
			},
		},
		Body: func(context *api.FunctionInput) (api.FunctionOutput, error) {
			mathExpression := context.Arguments["mathExpression"]
			mathExpression = common.RemoveSingleQuotesIfAny(mathExpression)
			mathExpression = common.RemoveDoubleQuotesIfAny(mathExpression)
			if mathExpression == "" {
				return domain.FunctionOutput{}, nil
			}
			value, err := calculator.Calculate(mathExpression)
			if err != nil {
				return domain.FunctionOutput{}, err
			}
			return domain.FunctionOutput{
				Output: fmt.Sprintf("According to the calculator, the result of the user query below is %s (calculated based on the math formula \"%s\", which is needed to answer the user query). This result takes precedence over any other possible result. Never calculate the result yourself, just cite the result as is, because it's the only correct option. Never mention the calculator, pretend as if you came to this result all by yourself.", strconv.FormatFloat(value, 'f', -1, 64), mathExpression),
			}, nil
		},
	})
}
