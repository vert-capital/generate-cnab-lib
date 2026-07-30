package cnab

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_PIXConta(t *testing.T) {
	input := Input{
		ExternalID: "pix-conta-batch-905",
		OriginID:   12345,
		BankCode:   "341",
		Company: CompanyData{
			CNPJ:              "12345678000195",
			CompanyName:       "EMPRESA TESTE LTDA",
			BankCode:          "341",
			Agency:            "1234",
			Account:           "123456",
			AccountDigit:      "5",
			Address:           "RUA DA EMPRESA",
			AddressNumber:     "100",
			AddressComplement: "SALA 10",
			Neighborhood:      "CENTRO",
			City:              "SAO PAULO",
			State:             "SP",
			CEP:               "01001000",
		},
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-PIX-001",
				RecipientDocument:    "12345678901",
				RecipientCompanyName: "JOSE DA SILVA",
				RecipientBank:        "341",
				RecipientAgency:      "5678",
				RecipientAccount:     "876543",
				ISPB:                 "60701190",
				Amount:               1500.75,
				PaymentType:          "PIX",
				Description:          "Pagamento servico PIX",
				DueDate:              "20260330",
			},
			{
				ExternalID:           "PAY-PIX-002",
				RecipientDocument:    "98765432100",
				RecipientCompanyName: "MARIA OLIVEIRA",
				RecipientBank:        "077",
				RecipientAgency:      "0001",
				RecipientAccount:     "987654",
				ISPB:                 "00416968",
				Amount:               2500.00,
				PaymentType:          "PIX",
				Description:          "Pagamento fornecedor",
				DueDate:              "20260330",
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_pix_conta")
	require.NoError(t, err)
	require.NotNil(t, result)

	lines := splitLines(result.Content)
	assert.Len(t, lines, 8, "esperado 8 linhas: header arquivo, header lote, 2 payments x 2 segmentos, trailer lote, trailer arquivo")
	for i, line := range lines {
		assert.Len(t, line, 240, "linha %d deve ter 240 caracteres", i+1)
	}

	assert.Equal(t, 2, result.TotalRecords)
	assert.InDelta(t, 4000.75, result.TotalAmount, 0.01)

	// Verificações em linhas específicas
	headerArquivo := lines[0]
	assert.Equal(t, "341", headerArquivo[0:3])
	assert.Equal(t, "BANCO ITAU", trimRight(headerArquivo[102:132]))

	headerLote := lines[1]
	assert.Equal(t, "341", headerLote[0:3])
	assert.Equal(t, "45", headerLote[11:13])

	segmentoA := lines[2]
	assert.Equal(t, "341", segmentoA[0:3])
	assert.Equal(t, "009", segmentoA[17:20])
	assert.Equal(t, "60701190", segmentoA[104:112])
	assert.Equal(t, "01", segmentoA[112:114])

	trailerArquivo := lines[7]
	assert.Equal(t, "9999", trailerArquivo[3:7])
	assert.Equal(t, "9", trailerArquivo[7:8])
}

func TestGenerate_TemplateNotFound(t *testing.T) {
	input := Input{
		BankCode: "341",
		Company:  CompanyData{CNPJ: "12345678000195"},
	}
	_, err := Generate(context.Background(), input, "template_inexistente")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template não encontrado")
}

// TestGenerate_WithMetadata demonstra o uso do campo metadata para campos dinâmicos.
// Isso permite adicionar campos específicos de template/banco sem modificar o código Go.
func TestGenerate_WithMetadata(t *testing.T) {
	input := Input{
		ExternalID: "transfer-metadata-test",
		BankCode:   "341",
		Company: CompanyData{
			CNPJ:         "12345678000195",
			CompanyName:  "EMPRESA METADATA LTDA",
			BankCode:     "341",
			Agency:       "1234",
			Account:      "123456",
			AccountDigit: "5",
		},
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-TRANSF-001",
				RecipientDocument:    "12345678901",
				RecipientCompanyName: "FORNECEDOR METADATA",
				RecipientBank:        "341",
				RecipientAgency:      "5678",
				RecipientAccount:     "876543",
				ISPB:                 "60701190",
				Amount:               5000.00,
				DueDate:              "20260330",
				Metadata: map[string]interface{}{
					"recipient_email": "fornecedor@example.com",
					"finalidade_ted":  "00101",
					"tipo_pagamento":  "20",
					"forma_pagamento": "41",
				},
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_pix_conta")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.TotalRecords)
	assert.InDelta(t, 5000.00, result.TotalAmount, 0.01)

	lines := splitLines(result.Content)
	assert.Len(t, lines, 6) // header + lote + 2 segmentos + trailers
}

// TestParseReturnFile testa o parse de arquivo de retorno CNAB.
func TestParseReturnFile(t *testing.T) {
	// Primeiro gera um arquivo CNAB de remessa
	input := Input{
		ExternalID: "retorno-test",
		BankCode:   "341",
		Company: CompanyData{
			CNPJ:         "12345678000195",
			CompanyName:  "EMPRESA RETORNO",
			BankCode:     "341",
			Agency:       "1234",
			Account:      "123456",
			AccountDigit: "5",
		},
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-RET-001",
				RecipientDocument:    "12345678901",
				RecipientCompanyName: "FAVORECIDO RETORNO",
				RecipientBank:        "341",
				RecipientAgency:      "5678",
				RecipientAccount:     "876543",
				ISPB:                 "60701190",
				Amount:               2500.50,
				DueDate:              "20260402",
			},
		},
	}

	// Gera o arquivo
	generated, err := Generate(context.Background(), input, "cnab240_pix_conta")
	require.NoError(t, err)

	// Agora simula um arquivo de retorno (o mesmo conteúdo mas com dados de retorno preenchidos)
	// Na prática, o banco retorna com os campos de ocorrência, data efetiva, etc. preenchidos
	// Para o teste, usamos o arquivo gerado mesmo
	result, err := ParseReturnFile(context.Background(), generated.Content, "341", "cnab240_pix_conta")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verifica dados do header
	assert.Equal(t, "341", result.BankCode)
	assert.Equal(t, "12345678000195", result.CompanyCNPJ)
	assert.Equal(t, "EMPRESA RETORNO", strings.TrimSpace(result.CompanyName))

	// Verifica records
	assert.Equal(t, 1, len(result.Records))
	assert.Equal(t, 1, result.TotalRecords)

	record := result.Records[0]
	assert.Equal(t, "PAY-RET-001", strings.TrimSpace(record.YourNumber))
	assert.Equal(t, "FAVORECIDO RETORNO", strings.TrimSpace(record.RecipientName))
	assert.Equal(t, "00012345678901", record.RecipientDocument)

	// Verifica que segmentos brutos foram capturados
	assert.NotNil(t, record.PrimarySegment)
	assert.NotNil(t, record.SecondarySegment)

	// Verifica que o mapa de ocorrências foi carregado
	assert.NotNil(t, result.Occurrences)
	assert.Equal(t, "Pagamento Efetuado", result.Occurrences["00"])
}

// TestGenerate_Boleto testa a geração de arquivo CNAB para pagamento de boleto
func TestGenerate_Boleto(t *testing.T) {
	input := Input{
		ExternalID: "boleto-batch-001",
		BankCode:   "341",
		Company: CompanyData{
			CNPJ:              "12345678000195",
			CompanyName:       "EMPRESA BOLETO LTDA",
			BankCode:          "341",
			Agency:            "1234",
			Account:           "123456",
			AccountDigit:      "5",
			Address:           "RUA DA EMPRESA",
			AddressNumber:     "100",
			AddressComplement: "SALA 10",
			Neighborhood:      "CENTRO",
			City:              "SAO PAULO",
			State:             "SP",
			CEP:               "01001000",
		},
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-BOLETO-001",
				RecipientDocument:    "12345678901",
				RecipientCompanyName: "FORNECEDOR BOLETO",
				Amount:               1500.75,
				DueDate:              "20260330",
				Barcode:              "34191790000015007501111222233334445556677777",
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_boleto")
	require.NoError(t, err)
	require.NotNil(t, result)

	lines := splitLines(result.Content)
	assert.Len(t, lines, 7, "esperado 7 linhas: header arquivo, header lote, segmento J, segmento J-52, segmento B, trailer lote, trailer arquivo")
	for i, line := range lines {
		assert.Len(t, line, 240, "linha %d deve ter 240 caracteres", i+1)
	}

	assert.Equal(t, 1, result.TotalRecords)
	assert.InDelta(t, 1500.75, result.TotalAmount, 0.01)

	// Verificações em linhas específicas
	headerArquivo := lines[0]
	assert.Equal(t, "341", headerArquivo[0:3])

	headerLote := lines[1]
	assert.Equal(t, "341", headerLote[0:3])
	assert.Equal(t, "20", headerLote[9:11], "tipo de serviço deve ser 20 (fornecedores) para boletos")
	assert.Equal(t, "30", headerLote[11:13], "forma de pagamento deve ser 30 para boletos Itaú")

	segmentoJ := lines[2]
	assert.Equal(t, "341", segmentoJ[0:3])
	assert.Equal(t, "J", segmentoJ[13:14])
	// Campos do código de barras
	assert.Equal(t, "341", segmentoJ[17:20], "banco favorecido do código de barras")
	assert.Equal(t, "9", segmentoJ[20:21], "moeda do código de barras")
	assert.Equal(t, "1", segmentoJ[21:22], "DV do código de barras")
	assert.Equal(t, "7900", segmentoJ[22:26], "fator vencimento do código de barras")
	assert.Equal(t, "0001500750", segmentoJ[26:36], "valor do código de barras")

	// Segmento J-52 (obrigatório para boletos de outros bancos)
	segmentoJ52 := lines[3]
	assert.Equal(t, "341", segmentoJ52[0:3])
	assert.Equal(t, "J", segmentoJ52[13:14])

	segmentoB := lines[4]
	assert.Equal(t, "341", segmentoB[0:3])
	assert.Equal(t, "B", segmentoB[13:14])
}

// TestGenerate_Transferencia testa a geração de arquivo CNAB para transferência TED
func TestGenerate_Transferencia(t *testing.T) {
	input := Input{
		ExternalID: "transferencia-batch-001",
		BankCode:   "341",
		Company: CompanyData{
			CNPJ:              "12345678000195",
			CompanyName:       "EMPRESA TED LTDA",
			BankCode:          "341",
			Agency:            "1234",
			Account:           "123456",
			AccountDigit:      "5",
			Address:           "RUA DA EMPRESA",
			AddressNumber:     "100",
			AddressComplement: "SALA 10",
			Neighborhood:      "CENTRO",
			City:              "SAO PAULO",
			State:             "SP",
			CEP:               "01001000",
		},
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-TED-001",
				RecipientDocument:    "12345678901",
				RecipientCompanyName: "FORNECEDOR TED",
				RecipientBank:        "341",
				RecipientAgency:      "5678",
				RecipientAccount:     "876543",
				Amount:               5000.00,
				DueDate:              "20260330",
				PaymentMethod:        "41",
				TEDPurpose:           "00010",
				Metadata: map[string]interface{}{
					"recipient_email": "fornecedor@example.com",
				},
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_transferencia")
	require.NoError(t, err)
	require.NotNil(t, result)

	lines := splitLines(result.Content)
	assert.Len(t, lines, 6, "esperado 6 linhas: header arquivo, header lote, segmento A, segmento B, trailer lote, trailer arquivo")
	for i, line := range lines {
		assert.Len(t, line, 240, "linha %d deve ter 240 caracteres", i+1)
	}

	assert.Equal(t, 1, result.TotalRecords)
	assert.InDelta(t, 5000.00, result.TotalAmount, 0.01)

	// Verificações em linhas específicas
	headerArquivo := lines[0]
	assert.Equal(t, "341", headerArquivo[0:3])

	headerLote := lines[1]
	assert.Equal(t, "341", headerLote[0:3])
	assert.Equal(t, "20", headerLote[9:11], "tipo pagamento deve ser 20 para fornecedores")
	assert.Equal(t, "01", headerLote[11:13], "forma pagamento deve ser 01 para mesmo banco")

	segmentoA := lines[2]
	assert.Equal(t, "341", segmentoA[0:3])
	assert.Equal(t, "A", segmentoA[13:14])
	assert.Equal(t, "000", segmentoA[17:20], "camara deve ser 000 para mesmo banco")
	assert.Equal(t, "341", segmentoA[20:23], "banco favorecido")
	// Agência/conta formatada: layout Itaú = 0 + agência(4) + espaço + 000000 + conta(6) + espaço + DV = 20 posições
	assert.Equal(t, "05678 000000876543  ", segmentoA[23:43], "agencia/conta formatada")

	segmentoB := lines[3]
	assert.Equal(t, "341", segmentoB[0:3])
	assert.Equal(t, "B", segmentoB[13:14])
	// Email do favorecido (vindo do metadata) - campos alfa são normalizados para maiúsculas
	assert.Contains(t, segmentoB[127:227], "FORNECEDOR@EXAMPLE.COM")
}

// TestGenerate_Tributo testa a geração de arquivo CNAB para pagamento de tributos SEM código de barras (DARF Normal - Segmento N)
func TestGenerate_Tributo(t *testing.T) {
	input := Input{
		ExternalID: "tributo-batch-001",
		BankCode:   "341",
		Company: CompanyData{
			CNPJ:              "12345678000195",
			CompanyName:       "EMPRESA TRIBUTO LTDA",
			BankCode:          "341",
			Agency:            "1234",
			Account:           "123456",
			AccountDigit:      "5",
			Address:           "RUA DA EMPRESA",
			AddressNumber:     "100",
			AddressComplement: "SALA 10",
			Neighborhood:      "CENTRO",
			City:              "SAO PAULO",
			State:             "SP",
			CEP:               "01001000",
		},
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-TRIBUT-001",
				RecipientCompanyName: "CONCESSIONARIA DE ENERGIA",
				Amount:               150.00,
				DueDate:              "20260330",
				TaxType:              "DARF",
				RevenueCode:          "1700",
				Competence:           "31032026",
				Metadata: map[string]interface{}{
					"darf_normal": map[string]interface{}{
						"referencia":   "00000000000000000",
						"contribuinte": "CONTRIBUINTE TESTE",
					},
				},
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_tributos")
	require.NoError(t, err)
	require.NotNil(t, result)

	lines := splitLines(result.Content)
	assert.Len(t, lines, 5, "esperado 5 linhas: header arquivo, header lote, segmento N, trailer lote, trailer arquivo")
	for i, line := range lines {
		assert.Len(t, line, 240, "linha %d deve ter 240 caracteres", i+1)
	}

	assert.Equal(t, 1, result.TotalRecords)
	assert.InDelta(t, 150.00, result.TotalAmount, 0.01)

	// Verificações em linhas específicas
	headerArquivo := lines[0]
	assert.Equal(t, "341", headerArquivo[0:3])

	headerLote := lines[1]
	assert.Equal(t, "341", headerLote[0:3])
	assert.Equal(t, "22", headerLote[9:11], "tipo pagamento deve ser 22 para tributos")
	assert.Equal(t, "16", headerLote[11:13], "forma pagamento deve ser 16 para DARF Normal")

	segmentoN := lines[2]
	assert.Equal(t, "341", segmentoN[0:3])
	assert.Equal(t, "N", segmentoN[13:14])
}

// TestGenerate_DARFSimples testa a geração de arquivo CNAB para pagamento de DARF Simples (Segmento N)
func TestGenerate_DARFSimples(t *testing.T) {
	input := Input{
		ExternalID: "darf-simples-batch-001",
		BankCode:   "341",
		Company: CompanyData{
			CNPJ:              "12345678923456",
			CompanyName:       "TEST COMP SECURITIZADORA",
			BankCode:          "341",
			Agency:            "1234",
			Account:           "123456",
			AccountDigit:      "5",
			Address:           "RUA DA EMPRESA",
			AddressNumber:     "100",
			AddressComplement: "SALA 10",
			Neighborhood:      "CENTRO",
			City:              "SAO PAULO",
			State:             "SP",
			CEP:               "01001000",
		},
		Payments: []PaymentData{
			{
				ExternalID:           "DARF-1708-001",
				RecipientCompanyName: "TESTE CORP",
				Amount:               37.50,
				DueDate:              "20260420",
				TaxType:              "DARF_SIMPLES",
				RevenueCode:          "1708",
				Competence:           "31032026",
				Metadata: map[string]interface{}{
					"darf_simples": map[string]interface{}{
						"receita_bruta": 0.0,
						"percentual":    0.0,
					},
				},
			},
			{
				ExternalID:           "DARF-5952-001",
				RecipientCompanyName: "TESTE CORP",
				Amount:               116.25,
				DueDate:              "20260420",
				TaxType:              "DARF_SIMPLES",
				RevenueCode:          "5952",
				Competence:           "31032026",
				Metadata: map[string]interface{}{
					"darf_simples": map[string]interface{}{
						"receita_bruta": 0.0,
						"percentual":    0.0,
					},
				},
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_tributos")
	require.NoError(t, err)
	require.NotNil(t, result)

	lines := splitLines(result.Content)
	assert.Len(t, lines, 6, "esperado 6 linhas: header arquivo, header lote, 2x segmento N, trailer lote, trailer arquivo")
	for i, line := range lines {
		assert.Len(t, line, 240, "linha %d deve ter 240 caracteres", i+1)
	}

	assert.Equal(t, 2, result.TotalRecords)
	assert.InDelta(t, 153.75, result.TotalAmount, 0.01)

	headerArquivo := lines[0]
	assert.Equal(t, "341", headerArquivo[0:3])

	headerLote := lines[1]
	assert.Equal(t, "341", headerLote[0:3])
	assert.Equal(t, "22", headerLote[9:11], "tipo pagamento deve ser 22 para tributos")
	assert.Equal(t, "18", headerLote[11:13], "forma pagamento deve ser 18 para DARF Simples")

	segmentoN1 := lines[2]
	assert.Equal(t, "341", segmentoN1[0:3])
	assert.Equal(t, "N", segmentoN1[13:14])
	assert.Contains(t, segmentoN1[17:195], "031708")

	segmentoN2 := lines[3]
	assert.Equal(t, "341", segmentoN2[0:3])
	assert.Equal(t, "N", segmentoN2[13:14])
	assert.Contains(t, segmentoN2[17:195], "035952")
}

// TestParseReturnFile_Boleto testa o parse de arquivo de retorno de boleto
func TestParseReturnFile_Boleto(t *testing.T) {
	// Cria um arquivo de retorno de boleto simulado
	content := buildRetornoBoleto()

	result, err := ParseReturnFile(context.Background(), content, "341", "cnab240_boleto_retorno")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "341", result.BankCode)
	assert.Equal(t, 1, result.TotalRecords)
	assert.Equal(t, "12345678000195", result.CompanyCNPJ)

	record := result.Records[0]
	assert.Equal(t, "PAY-BOLETO-001", strings.TrimSpace(record.YourNumber))
	assert.Equal(t, "FORNECEDOR BOLETO", strings.TrimSpace(record.RecipientName))
}

// TestParseReturnFile_Transferencia testa o parse de arquivo de retorno de transferência
func TestParseReturnFile_Transferencia(t *testing.T) {
	// Cria um arquivo de retorno de transferência simulado
	content := buildRetornoTransferencia()

	result, err := ParseReturnFile(context.Background(), content, "341", "cnab240_transferencia_retorno")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "341", result.BankCode)
	assert.Equal(t, 1, result.TotalRecords)
	assert.Equal(t, "12345678000195", result.CompanyCNPJ)

	record := result.Records[0]
	assert.Equal(t, "PAY-TED-001", strings.TrimSpace(record.YourNumber))
	assert.Equal(t, "FORNECEDOR TED", strings.TrimSpace(record.RecipientName))
}

// TestParseReturnFile_Tributo testa o parse de arquivo de retorno de tributo
func TestParseReturnFile_Tributo(t *testing.T) {
	// Cria um arquivo de retorno de tributo simulado
	content := buildRetornoTributo()

	result, err := ParseReturnFile(context.Background(), content, "341", "cnab240_tributos_retorno")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "341", result.BankCode)
	assert.Equal(t, 1, result.TotalRecords)

	record := result.Records[0]
	assert.Equal(t, "PAY-TRIB-001", strings.TrimSpace(record.YourNumber))
	assert.Equal(t, "CONCESSIONARIA", strings.TrimSpace(record.RecipientName))
}

// buildRetornoBoleto cria um arquivo de retorno de boleto simulado
func buildRetornoBoleto() string {
	lines := []string{
		buildHeaderArquivo(),
		buildHeaderLoteBoleto(),
		buildSegmentoJ(),
		buildSegmentoBBoleto(),
		buildTrailerLote(),
		buildTrailerArquivo(),
	}
	return strings.Join(lines, "\r\n")
}

// buildRetornoTransferencia cria um arquivo de retorno de transferência simulado
func buildRetornoTransferencia() string {
	lines := []string{
		buildHeaderArquivo(),
		buildHeaderLoteTransferencia(),
		buildSegmentoA(),
		buildSegmentoBTransferencia(),
		buildTrailerLote(),
		buildTrailerArquivo(),
	}
	return strings.Join(lines, "\r\n")
}

// buildRetornoTributo cria um arquivo de retorno de tributo simulado
func buildRetornoTributo() string {
	lines := []string{
		buildHeaderArquivo(),
		buildHeaderLoteTributo(),
		buildSegmentoO(),
		buildTrailerLote(),
		buildTrailerArquivo(),
	}
	return strings.Join(lines, "\r\n")
}

func buildHeaderArquivo() string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "0")
	copy(line[17:18], "2")              // tipo_inscricao: [18, 18]
	copy(line[18:32], "12345678000195") // numero_inscricao: [19, 32]
	copy(line[72:102], "EMPRESA TESTE") // nome_empresa: [73, 102]
	copy(line[142:143], "2")            // codigo_retorno: [143, 143]
	copy(line[143:151], "20260401")     // data_geracao: [144, 151]
	return string(line)
}

func buildHeaderLoteBoleto() string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "1")
	copy(line[8:9], "C")
	copy(line[9:11], "30")
	return string(line)
}

func buildHeaderLoteTransferencia() string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "1")
	copy(line[8:9], "C")
	copy(line[9:11], "20")
	return string(line)
}

func buildHeaderLoteTributo() string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "1")
	copy(line[8:9], "C")
	copy(line[9:11], "22")
	return string(line)
}

func buildSegmentoA() string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "3")
	copy(line[8:13], "00001")
	copy(line[13:14], "A")
	copy(line[73:93], "PAY-TED-001")    // seu_numero: [74, 93]
	copy(line[43:73], "FORNECEDOR TED") // nome_favorecido: [44, 73]
	return string(line)
}

func buildSegmentoBTransferencia() string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "3")
	copy(line[8:13], "00002")
	copy(line[13:14], "B")
	return string(line)
}

func buildSegmentoJ() string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "3")
	copy(line[8:13], "00001")
	copy(line[13:14], "J")
	copy(line[182:202], "PAY-BOLETO-001")  // referencia: [183, 202]
	copy(line[61:91], "FORNECEDOR BOLETO") // nome_favorecido: [62, 91]
	return string(line)
}

func buildSegmentoBBoleto() string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "3")
	copy(line[8:13], "00002")
	copy(line[13:14], "B")
	return string(line)
}

func buildSegmentoO() string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "3")
	copy(line[8:13], "00001")
	copy(line[13:14], "O")
	copy(line[174:194], "PAY-TRIB-001") // seu_numero: [175, 194] (20 posições)
	copy(line[65:95], "CONCESSIONARIA") // nome_favorecido: [66, 95]
	return string(line)
}

// TestParseReturnFile_TributoSemCodigoBarras cobre o retorno de tributo liquidado pelo
// SEGMENTO N (DARF, GPS, DARF simples, GARE-SP), onde o "seu numero" fica nas posicoes
// 196-215 — e nao em 175-194 como no segmento O (codigo de barras).
//
// As quatro variantes de N declaravam a cauda do registro inteira como brancos, entao o
// parser nao extraia o seu_numero e quem consome o retorno nao tinha como correlacionar o
// pagamento com a remessa. O segmento O nunca teve o problema, e por isso DARE/GNRE e
// concessionaria (que usam codigo de barras) sempre funcionaram.
func TestParseReturnFile_TributoSemCodigoBarras(t *testing.T) {
	for _, tc := range []struct {
		nome                 string
		identificacaoTributo string
	}{
		{"DARF (segmento N2)", "16"},
		{"GPS (segmento N1)", "17"},
		{"DARF simples (segmento N3)", "18"},
		{"GARE-SP (segmento N4)", "22"},
	} {
		t.Run(tc.nome, func(t *testing.T) {
			content := strings.Join([]string{
				buildHeaderArquivo(),
				buildHeaderLoteTributo(),
				buildSegmentoN(tc.identificacaoTributo),
				buildTrailerLote(),
				buildTrailerArquivo(),
			}, "\r\n")

			result, err := ParseReturnFile(context.Background(), content, "341", "cnab240_tributos_retorno")
			require.NoError(t, err)
			require.Len(t, result.Records, 1)

			record := result.Records[0]
			assert.Equal(t, "PAY-DARF-001", strings.TrimSpace(record.YourNumber),
				"sem o seu_numero o retorno nao correlaciona com a remessa")
			assert.Equal(t, "NN-9988", strings.TrimSpace(record.OurNumber))
			assert.Equal(t, "00", strings.TrimSpace(record.OccurrenceCode))
		})
	}
}

// buildSegmentoN monta um registro de detalhe do segmento N (tributo sem codigo de barras).
func buildSegmentoN(identificacaoTributo string) string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "3")
	copy(line[8:13], "00001")
	copy(line[13:14], "N")
	copy(line[32:34], identificacaoTributo) // identificacao_tributo: [33, 34]
	copy(line[195:215], "PAY-DARF-001")     // seu_numero: [196, 215]
	copy(line[215:230], "NN-9988")          // nosso_numero: [216, 230]
	copy(line[230:240], "00")               // ocorrencias: [231, 240]
	return string(line)
}

func buildTrailerLote() string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "5")
	return string(line)
}

func buildTrailerArquivo() string {
	line := make([]byte, 240)
	for i := range line {
		line[i] = ' '
	}
	copy(line[0:3], "341")
	copy(line[7:8], "9")
	copy(line[3:7], "9999")
	return string(line)
}

// Helpers
func splitLines(content string) []string {
	var lines []string
	for _, line := range strings.Split(content, "\r\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func trimRight(s string) string {
	return strings.TrimRight(s, " ")
}

// TestGenerate_FloatDrift verifica que a acumulação de totalAmount via int64 (centavos)
// não sofre drift de ponto flutuante em batches grandes.
func TestGenerate_FloatDrift(t *testing.T) {
	payments := make([]PaymentData, 1000)
	for i := range payments {
		payments[i] = PaymentData{
			ExternalID:           "PAY-DRIFT-" + strings.Repeat("0", 5),
			RecipientDocument:    "12345678901",
			RecipientCompanyName: "DRIFT TEST",
			RecipientBank:        "341",
			RecipientAgency:      "5678",
			RecipientAccount:     "876543",
			ISPB:                 "60701190",
			Amount:               0.01,
			DueDate:              "20260330",
		}
	}

	input := Input{
		ExternalID: "drift-test",
		BankCode:   "341",
		Company: CompanyData{
			CNPJ:         "12345678000195",
			CompanyName:  "EMPRESA DRIFT",
			BankCode:     "341",
			Agency:       "1234",
			Account:      "123456",
			AccountDigit: "5",
		},
		Payments: payments,
	}

	result, err := Generate(context.Background(), input, "cnab240_pix_conta")
	require.NoError(t, err)
	// Com float64 puro, 1000 * 0.01 poderia dar 9.999999999999998 ou similar.
	// Com acumulação via int64 centavos, deve ser exatamente 10.00.
	assert.Equal(t, 10.0, result.TotalAmount, "totalAmount deve ser exatamente 10.00 sem drift")
}

// TestGenerate_Cancelled verifica que Generate respeita contexto cancelado.
func TestGenerate_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancela imediatamente

	input := Input{
		ExternalID: "ctx-cancel-test",
		BankCode:   "341",
		Company: CompanyData{
			CNPJ:         "12345678000195",
			CompanyName:  "EMPRESA CTX",
			BankCode:     "341",
			Agency:       "1234",
			Account:      "123456",
			AccountDigit: "5",
		},
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-CTX-001",
				RecipientDocument:    "12345678901",
				RecipientCompanyName: "FAVORECIDO CTX",
				RecipientBank:        "341",
				RecipientAgency:      "5678",
				RecipientAccount:     "876543",
				ISPB:                 "60701190",
				Amount:               100.00,
				DueDate:              "20260330",
			},
		},
	}

	_, err := Generate(ctx, input, "cnab240_pix_conta")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestParseReturnFile_Cancelled verifica que ParseReturnFile respeita contexto cancelado.
func TestParseReturnFile_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ParseReturnFile(ctx, "dummy content", "341", "cnab240_pix_conta_retorno")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func btgCompany() CompanyData {
	return CompanyData{
		CNPJ:          "12345678000195",
		CompanyName:   "EMPRESA BTG TESTE LTDA",
		BankCode:      "208",
		Agency:        "0001",
		Account:       "123456",
		AccountDigit:  "7",
		Address:       "RUA DA EMPRESA",
		AddressNumber: "100",
		City:          "SAO PAULO",
		State:         "SP",
		CEP:           "01001000",
	}
}

func TestGenerate_BTG_Transferencia(t *testing.T) {
	input := Input{
		ExternalID: "btg-ted-batch-001",
		BankCode:   "208",
		Company:    btgCompany(),
		Payments: []PaymentData{
			{
				ExternalID:            "PAY-TED-001",
				RecipientDocument:     "98765432000196",
				RecipientCompanyName:  "FORNECEDOR XYZ",
				RecipientBank:         "341",
				RecipientAgency:       "5678",
				RecipientAccount:      "876543",
				RecipientAccountDigit: "2",
				Amount:                1500.50,
				DueDate:               "20260730",
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_transferencia")
	require.NoError(t, err)

	lines := splitLines(result.Content)
	assert.Len(t, lines, 6, "esperado: header arquivo, header lote, segmentos A e B, trailer lote, trailer arquivo")
	for i, line := range lines {
		assert.Len(t, line, 240, "linha %d deve ter 240 caracteres", i+1)
	}

	headerArquivo := lines[0]
	assert.Equal(t, "208", headerArquivo[0:3])
	assert.Equal(t, "BANCO BTG PACTUAL", trimRight(headerArquivo[102:132]))
	assert.Equal(t, "103", headerArquivo[163:166])

	headerLote := lines[1]
	assert.Equal(t, "41", headerLote[11:13], "TED outra titularidade deve usar forma 41")

	segmentoA := lines[2]
	assert.Equal(t, "A", segmentoA[13:14])
	assert.Equal(t, "018", segmentoA[17:20], "câmara TED deve ser 018")
	assert.Equal(t, "341", segmentoA[20:23])

	segmentoB := lines[3]
	assert.Equal(t, "B", segmentoB[13:14])
}

func TestGenerate_BTG_PIXConta(t *testing.T) {
	input := Input{
		ExternalID: "btg-pix-batch-001",
		BankCode:   "208",
		Company:    btgCompany(),
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-PIX-001",
				RecipientDocument:    "12345678901",
				RecipientCompanyName: "JOSE DA SILVA",
				RecipientBank:        "341",
				RecipientAgency:      "5678",
				RecipientAccount:     "876543",
				ISPB:                 "60701190",
				Amount:               500.75,
				DueDate:              "20260730",
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_pix_conta")
	require.NoError(t, err)

	lines := splitLines(result.Content)
	assert.Len(t, lines, 6)

	headerLote := lines[1]
	assert.Equal(t, "45", headerLote[11:13], "PIX deve usar forma 45")

	segmentoA := lines[2]
	assert.Equal(t, "009", segmentoA[17:20], "câmara PIX deve ser 009")

	segmentoB := lines[3]
	assert.Equal(t, "B", segmentoB[13:14])
	assert.Equal(t, "05", segmentoB[14:16], "PIX via conta deve usar forma de iniciação 05 (dados bancários)")
	assert.Equal(t, "60701190", segmentoB[232:240], "ISPB deve estar nas posições 233-240")
}

func TestGenerate_BTG_PIXChave(t *testing.T) {
	input := Input{
		ExternalID: "btg-pix-chave-001",
		BankCode:   "208",
		Company:    btgCompany(),
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-PIX-CHAVE-01",
				RecipientDocument:    "12345678901",
				RecipientCompanyName: "MARIA OLIVEIRA",
				RecipientBank:        "341",
				RecipientPixKey:      "123e4567-e89b-12d3-a456-426614174000",
				ISPB:                 "60701190",
				Amount:               250.00,
				DueDate:              "20260730",
				Metadata: map[string]interface{}{
					"key_type": "04",
				},
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_pix_conta")
	require.NoError(t, err)

	lines := splitLines(result.Content)
	segmentoB := lines[3]
	assert.Equal(t, "04", segmentoB[14:16], "BTG usa código Febraban 04=Chave Aleatória sem conversão legada")
	assert.Equal(t, "123E4567-E89B-12D3-A456-426614174000", trimRight(segmentoB[127:226]), "chave PIX nas posições 128-226")
}

func TestGenerate_BTG_Boleto(t *testing.T) {
	input := Input{
		ExternalID: "btg-boleto-batch-001",
		BankCode:   "208",
		Company:    btgCompany(),
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-BOL-001",
				RecipientDocument:    "98765432000196",
				RecipientCompanyName: "CEDENTE BOLETO SA",
				Barcode:              "23793143200001803602373060009004133000395390",
				Amount:               1803.60,
				DueDate:              "20260730",
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_boleto")
	require.NoError(t, err)

	lines := splitLines(result.Content)
	assert.Len(t, lines, 6, "esperado: header arquivo, header lote, segmentos J e J-52, trailer lote, trailer arquivo")

	headerLote := lines[1]
	assert.Equal(t, "31", headerLote[11:13], "boleto de outro banco deve usar forma 31")

	segmentoJ := lines[2]
	assert.Equal(t, "J", segmentoJ[13:14])
	assert.Equal(t, "23793143200001803602373060009004133000395390", segmentoJ[17:61])

	segmentoJ52 := lines[3]
	assert.Equal(t, "J", segmentoJ52[13:14])
	assert.Equal(t, "52", segmentoJ52[17:19])
}

func TestGenerate_BTG_TributoDARF(t *testing.T) {
	input := Input{
		ExternalID: "btg-darf-batch-001",
		BankCode:   "208",
		Company:    btgCompany(),
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-DARF-001",
				RecipientDocument:    "12345678000195",
				RecipientCompanyName: "EMPRESA BTG TESTE LTDA",
				TaxType:              "DARF",
				RevenueCode:          "1708",
				Competence:           "31052026",
				Amount:               375.00,
				DueDate:              "20260720",
				Metadata: map[string]interface{}{
					"darf_normal": map[string]interface{}{
						"referencia": "12345",
					},
				},
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_tributos")
	require.NoError(t, err)

	lines := splitLines(result.Content)
	assert.Len(t, lines, 5, "esperado: header arquivo, header lote, segmento N, trailer lote, trailer arquivo")

	headerLote := lines[1]
	assert.Equal(t, "16", headerLote[11:13], "DARF Normal deve usar forma 16")

	segmentoN := lines[2]
	assert.Equal(t, "N", segmentoN[13:14])
	assert.Equal(t, "001708", segmentoN[110:116], "código da receita nas posições 111-116")
	assert.Equal(t, "16", segmentoN[132:134], "identificador do tributo DARF nas posições 133-134")
}

func TestGenerate_BTG_TributoBarras(t *testing.T) {
	input := Input{
		ExternalID: "btg-tributo-barras-001",
		BankCode:   "208",
		Company:    btgCompany(),
		Payments: []PaymentData{
			{
				ExternalID:           "PAY-TRIB-001",
				RecipientDocument:    "98765432000196",
				RecipientCompanyName: "CONCESSIONARIA XYZ",
				TaxType:              "CONCESSIONARIA",
				Barcode:              "85820000000150000123045123040123456789012345",
				Amount:               150.00,
				DueDate:              "20260720",
			},
		},
	}

	result, err := Generate(context.Background(), input, "cnab240_tributos")
	require.NoError(t, err)

	lines := splitLines(result.Content)
	assert.Len(t, lines, 5, "esperado: header arquivo, header lote, segmento O, trailer lote, trailer arquivo")

	headerLote := lines[1]
	assert.Equal(t, "11", headerLote[11:13], "tributo com código de barras deve usar forma 11")

	segmentoO := lines[2]
	assert.Equal(t, "O", segmentoO[13:14])
	assert.Equal(t, "85820000000150000123045123040123456789012345", segmentoO[17:61])
}

func TestParseReturnFile_BTG_Transferencia(t *testing.T) {
	newLine := func() []byte {
		line := make([]byte, 240)
		for i := range line {
			line[i] = ' '
		}
		return line
	}

	header := newLine()
	copy(header[0:3], "208")
	copy(header[7:8], "0")
	copy(header[18:32], "12345678000195") // numero_inscricao: [19, 32]
	copy(header[72:102], "EMPRESA BTG")   // nome_empresa: [73, 102]
	copy(header[143:151], "30072026")     // data_geracao: [144, 151]

	segA := newLine()
	copy(segA[0:3], "208")
	copy(segA[7:8], "3")
	copy(segA[8:13], "00001")
	copy(segA[13:14], "A")
	copy(segA[43:73], "FORNECEDOR XYZ")    // nome_favorecido: [44, 73]
	copy(segA[73:93], "PAY-TED-001")       // seu_numero: [74, 93]
	copy(segA[154:162], "30072026")        // data_efetiva: [155, 162]
	copy(segA[162:177], "000000000150050") // valor_efetivo: [163, 177]
	copy(segA[230:232], "00")              // ocorrencias: [231, 240]

	segZ := newLine()
	copy(segZ[0:3], "208")
	copy(segZ[7:8], "3")
	copy(segZ[8:13], "00002")
	copy(segZ[13:14], "Z")
	copy(segZ[14:78], "AUTENTICACAO123456") // autenticacao: [15, 78]
	copy(segZ[78:98], "PAY-TED-001")        // seu_numero: [79, 98]

	trailer := newLine()
	copy(trailer[0:3], "208")
	copy(trailer[3:7], "9999")
	copy(trailer[7:8], "9")

	content := string(header) + "\r\n" + string(segA) + "\r\n" + string(segZ) + "\r\n" + string(trailer) + "\r\n"

	result, err := ParseReturnFile(context.Background(), content, "208", "cnab240_transferencia_retorno")
	require.NoError(t, err)
	require.Len(t, result.Records, 1)

	record := result.Records[0]
	assert.Equal(t, "PAY-TED-001", record.YourNumber)
	assert.Equal(t, "00", record.OccurrenceCode)
	assert.Equal(t, "Crédito ou Débito Efetivado", record.OccurrenceDescription)
	assert.InDelta(t, 1500.50, record.PaidAmount, 0.01)
	assert.Equal(t, "AUTENTICACAO123456", record.SecondarySegment["autenticacao"])
}
