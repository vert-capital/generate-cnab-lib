package cnab

import (
	"context"

	"github.com/vert-capital/generate-cnab-lib/internal/parser"
	"github.com/vert-capital/generate-cnab-lib/types"
)

// Re-exporta os tipos do pacote types para conveniência dos usuários.
type (
	// Input representa o payload de entrada para geração do CNAB.
	Input = types.Input

	// CompanyData dados da empresa pagadora.
	CompanyData = types.CompanyData

	// PaymentData dados de um pagamento individual.
	PaymentData = types.PaymentData

	// Result resultado da geração do arquivo CNAB.
	Result = types.Result

	// ReturnRecord representa um registro de detalhe do arquivo de retorno.
	ReturnRecord = types.ReturnRecord

	// ReturnFile representa o arquivo de retorno CNAB parseado.
	ReturnFile = types.ReturnFile
)

// Generate gera um arquivo CNAB de remessa a partir dos dados de entrada.
// O templateName deve ser o nome do template desejado (ex: "cnab240_pix_conta").
//
// Limitação atual: gera um único lote por arquivo. Para múltiplos tipos de
// pagamento no mesmo envio, gere arquivos separados.
func Generate(ctx context.Context, input Input, templateName string) (*Result, error) {
	return generate(ctx, input, templateName)
}

// ParseReturnFile parseia um arquivo de retorno CNAB e retorna os dados estruturados.
// Recebe o conteúdo do arquivo retorno, código do banco e nome do template.
func ParseReturnFile(ctx context.Context, content string, bankCode, templateName string) (*ReturnFile, error) {
	return parser.Parse(ctx, content, bankCode, templateName)
}
