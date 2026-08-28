package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vert-capital/generate-cnab-lib/internal/template"
)

func TestParseEmptyFile(t *testing.T) {
	result, err := Parse(context.Background(), "", "341", "cnab240_pix_conta_retorno")
	require.NoError(t, err)
	assert.Equal(t, "341", result.BankCode)
	assert.Equal(t, 0, result.TotalRecords)
	assert.Empty(t, result.Records)
}

func TestParseHeaderArquivo(t *testing.T) {
	// Cria uma linha de header de arquivo simulada
	line := buildLine(240, map[int]string{
		1:   "341",                // código banco
		8:   "0",                  // tipo registro
		18:  "2",                  // tipo inscrição
		19:  "12345678000195",     // CNPJ
		73:  "EMPRESA TESTE LTDA", // nome empresa
		144: "20260401",           // data geração
	})

	content := line + "\r\n"
	result, err := Parse(context.Background(), content, "341", "cnab240_pix_conta_retorno")
	require.NoError(t, err)

	assert.Equal(t, "12345678000195", result.CompanyCNPJ)
	assert.Equal(t, "EMPRESA TESTE LTDA", result.CompanyName)
	assert.Equal(t, "20260401", result.GenerationDate)
}

func TestParseSegmentoA(t *testing.T) {
	lines := []string{
		buildLine(240, map[int]string{
			1:  "341",
			8:  "0",
			18: "2",
			19: "12345678000195",
			73: "EMPRESA TESTE",
		}),
		buildLine(240, map[int]string{
			1: "341",
			8: "1",
		}),
		buildLine(240, map[int]string{
			1:   "341",
			8:   "3",
			14:  "A",
			74:  "PAY-001",
			44:  "FORNECEDOR A",
			135: "NOSSO001",
			155: "20260401",
			163: "000000000150075",
			204: "12345678901",
			231: "00",
		}),
		buildLine(240, map[int]string{
			1: "341",
			8: "5",
		}),
		buildLine(240, map[int]string{
			1: "341",
			8: "9",
		}),
	}

	content := joinLines(lines)
	result, err := Parse(context.Background(), content, "341", "cnab240_transferencia_retorno")
	require.NoError(t, err)

	assert.Equal(t, 1, result.TotalRecords)
	require.Len(t, result.Records, 1)

	record := result.Records[0]
	assert.Equal(t, "PAY-001", record.YourNumber)
	assert.Equal(t, "FORNECEDOR A", record.RecipientName)
	assert.Equal(t, "NOSSO001", record.OurNumber)
	assert.Equal(t, "00", record.OccurrenceCode)
	assert.Equal(t, "20260401", record.PaymentDate)
	assert.Equal(t, 1500.75, record.PaidAmount)
	assert.Equal(t, "12345678901", record.RecipientDocument)
}

func TestParseSegmentoJ(t *testing.T) {
	lines := []string{
		buildHeaderArquivo(),
		buildHeaderLote(),
		buildLine(240, map[int]string{
			1:   "341",
			8:   "3",
			14:  "J",
			62:  "FORNECEDOR BOLETO",
			92:  "20260330",
			183: "PAY-BOLETO-001",
			203: "NOSSO123",
			// Ocorrências do retorno: posições 231-240, como em todo registro CNAB240.
			231: "00",
		}),
		buildTrailerLote(),
		buildTrailerArquivo(),
	}

	content := joinLines(lines)
	result, err := Parse(context.Background(), content, "341", "cnab240_boleto_retorno")
	require.NoError(t, err)

	assert.Equal(t, 1, result.TotalRecords)
	require.Len(t, result.Records, 1)

	record := result.Records[0]
	assert.Equal(t, "PAY-BOLETO-001", record.YourNumber)
	assert.Equal(t, "FORNECEDOR BOLETO", record.RecipientName)
	assert.Equal(t, "NOSSO123", record.OurNumber)
	assert.Equal(t, "00", record.OccurrenceCode)
}

func TestParseSegmentoO(t *testing.T) {
	lines := []string{
		buildHeaderArquivo(),
		buildHeaderLote(),
		buildLine(240, map[int]string{
			1:   "341",
			8:   "3",
			14:  "O",
			66:  "CONCESSIONARIA ENERGIA",
			137: "30032026",
			175: "PAY-TRIB-001",
			231: "BD",
		}),
		buildTrailerLote(),
		buildTrailerArquivo(),
	}

	content := joinLines(lines)
	result, err := Parse(context.Background(), content, "341", "cnab240_tributos_retorno")
	require.NoError(t, err)

	assert.Equal(t, 1, result.TotalRecords)
	require.Len(t, result.Records, 1)

	record := result.Records[0]
	assert.Equal(t, "PAY-TRIB-001", record.YourNumber)
	assert.Equal(t, "CONCESSIONARIA ENERGIA", record.RecipientName)
	assert.Equal(t, "BD", record.OccurrenceCode)
}

func TestParseMultipleRecords(t *testing.T) {
	lines := []string{
		buildHeaderArquivo(),
		buildHeaderLote(),
		// Primeiro pagamento
		buildLine(240, map[int]string{
			1:   "341",
			8:   "3",
			14:  "A",
			74:  "PAY-001",
			44:  "FORNECEDOR 1",
			231: "00",
		}),
		// Segundo pagamento
		buildLine(240, map[int]string{
			1:   "341",
			8:   "3",
			14:  "A",
			74:  "PAY-002",
			44:  "FORNECEDOR 2",
			231: "BD",
		}),
		buildTrailerLote(),
		buildTrailerArquivo(),
	}

	content := joinLines(lines)
	result, err := Parse(context.Background(), content, "341", "cnab240_transferencia_retorno")
	require.NoError(t, err)

	assert.Equal(t, 2, result.TotalRecords)
	require.Len(t, result.Records, 2)

	assert.Equal(t, "PAY-001", result.Records[0].YourNumber)
	assert.Equal(t, "PAY-002", result.Records[1].YourNumber)
}

func TestParseWithSegmentoB(t *testing.T) {
	lines := []string{
		buildHeaderArquivo(),
		buildHeaderLote(),
		buildLine(240, map[int]string{
			1:  "341",
			8:  "3",
			14: "A",
			74: "PAY-001",
			44: "FORNECEDOR A",
		}),
		// Segmento B complementa o registro anterior
		buildLine(240, map[int]string{
			1:   "341",
			8:   "3",
			14:  "B",
			128: "fornecedor@example.com",
		}),
		buildTrailerLote(),
		buildTrailerArquivo(),
	}

	content := joinLines(lines)
	result, err := Parse(context.Background(), content, "341", "cnab240_transferencia_retorno")
	require.NoError(t, err)

	require.Len(t, result.Records, 1)
	assert.NotNil(t, result.Records[0].SecondarySegment)
	assert.Contains(t, result.Records[0].SecondarySegment["email"], "fornecedor@example.com")
}

func TestParseCurrency(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"zero", "000000000000000", 0},
		{"integer", "000000000150000", 1500},
		{"with cents", "000000000150075", 1500.75},
		{"small value", "000000000000001", 0.01},
		{"empty", "", 0},
		{"spaces", "   ", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCurrency(tt.input)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestExtractFields(t *testing.T) {
	seg := template.Segment{
		Fields: map[string]template.Field{
			"codigo_banco":  {Pos: [2]int{1, 3}},
			"lote_servico":  {Pos: [2]int{4, 7}},
			"tipo_registro": {Pos: [2]int{8, 8}},
		},
	}

	line := "34100010" + string(make([]byte, 232))
	fields := extractFields(line, seg)

	assert.Equal(t, "341", fields["codigo_banco"])
	assert.Equal(t, "0001", fields["lote_servico"])
	assert.Equal(t, "0", fields["tipo_registro"])
}

func TestGetFieldValue(t *testing.T) {
	fields := map[string]string{
		"existing": "value",
	}

	assert.Equal(t, "value", getFieldValue(fields, "existing"))
	assert.Equal(t, "", getFieldValue(fields, "nonexistent"))
}

func TestParseOrphanSegmentoB(t *testing.T) {
	// Segmento B sem segmento A precedente não deve causar panic nem associar ao record errado
	lines := []string{
		buildHeaderArquivo(),
		buildHeaderLote(),
		buildLine(240, map[int]string{
			1:  "341",
			8:  "3",
			14: "B",
		}),
		buildTrailerLote(),
		buildTrailerArquivo(),
	}

	content := joinLines(lines)
	result, err := Parse(context.Background(), content, "341", "cnab240_transferencia_retorno")
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalRecords, "segmento B órfão não deve criar record")
}

func TestParseSegmentoNTributos(t *testing.T) {
	// Valida que o parser dinâmico resolve segmento_n1_gps (a chave real no template de
	// tributos) sem nenhum código específico por segmento. O template antigo hardcodava
	// "segmento_n" que não existe no JSON de tributos, causando extração vazia.
	lines := []string{
		buildHeaderArquivo(),
		buildHeaderLote(),
		buildLine(240, map[int]string{
			1:   "341",
			8:   "3",
			14:  "N",
			231: "BD",
		}),
		buildTrailerLote(),
		buildTrailerArquivo(),
	}

	content := joinLines(lines)
	result, err := Parse(context.Background(), content, "341", "cnab240_tributos_retorno")
	require.NoError(t, err)

	assert.Equal(t, 1, result.TotalRecords)
	require.Len(t, result.Records, 1)

	record := result.Records[0]
	assert.Equal(t, "N", record.SegmentType)
	assert.Equal(t, "BD", record.OccurrenceCode, "campos do segmento_n1_gps devem ser extraídos")
	assert.NotEmpty(t, record.PrimarySegment, "PrimarySegment deve ter os campos brutos")
}

// Helpers

func buildLine(length int, values map[int]string) string {
	line := make([]byte, length)
	for i := range line {
		line[i] = ' '
	}
	for pos, value := range values {
		if pos > 0 && pos <= length {
			copy(line[pos-1:], value)
		}
	}
	return string(line)
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\r\n"
		}
		result += line
	}
	return result
}

func buildHeaderArquivo() string {
	return buildLine(240, map[int]string{
		1:  "341",
		8:  "0",
		18: "2",
		19: "12345678000195",
		73: "EMPRESA TESTE",
	})
}

func buildHeaderLote() string {
	return buildLine(240, map[int]string{
		1: "341",
		8: "1",
	})
}

func buildTrailerLote() string {
	return buildLine(240, map[int]string{
		1: "341",
		8: "5",
	})
}

func buildTrailerArquivo() string {
	return buildLine(240, map[int]string{
		1: "341",
		8: "9",
	})
}

// Muro de regressão: todo template de retorno precisa conseguir extrair a autenticação
// bancária, senão o comprovante de pagamento sai sem a linha "Autenticação Bancária" e
// ninguém percebe — foi assim que o retorno de boleto do Itaú ficou meses sem hash.
// Banco ou modalidade nova que entre sem segmento Z, ou com o campo de autenticação
// batizado de um jeito que a tabela canônica não conhece, quebra aqui.
func TestTodoTemplateDeRetornoExtraiAutenticacao(t *testing.T) {
	templates, err := template.Load()
	require.NoError(t, err)

	nomesConhecidos := make(map[string]bool, len(AuthenticationFieldNames))
	for _, nome := range AuthenticationFieldNames {
		nomesConhecidos[nome] = true
	}

	verificados := 0
	for chave, tmpl := range templates {
		if tmpl.FileType != "RETORNO" {
			continue
		}
		verificados++

		t.Run(chave, func(t *testing.T) {
			segKey, ok := buildSegmentIndex(tmpl)["Z"]
			require.True(t, ok, "template de retorno sem segmento Z: a autenticação bancária não tem de onde sair")

			var encontrados []string
			for campo := range tmpl.Segments[segKey].Fields {
				if nomesConhecidos[campo] {
					encontrados = append(encontrados, campo)
				}
			}
			require.NotEmpty(t, encontrados,
				"segmento %q não tem nenhum campo de autenticação conhecido (%v); "+
					"acrescente o nome usado por este banco em AuthenticationFieldNames",
				segKey, AuthenticationFieldNames)
		})
	}

	require.NotZero(t, verificados, "nenhum template de retorno foi verificado")
}
