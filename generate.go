package cnab

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/vert-capital/generate-cnab-lib/internal/formatter"
	"github.com/vert-capital/generate-cnab-lib/internal/resolver"
	"github.com/vert-capital/generate-cnab-lib/internal/template"
	"github.com/vert-capital/generate-cnab-lib/internal/validation"
)

// generate é a implementação interna de geração de CNAB.
// Cada pagamento recebe um contexto isolado para resolução de campos.
func generate(goCtx context.Context, input Input, templateName string) (*Result, error) {
	// Verifica se o template existe antes de validar campos
	tmpl, err := template.Get(input.BankCode, templateName)
	if err != nil {
		return nil, err
	}

	// Valida os dados de entrada
	if validationErrors := validation.ValidateInput(input, templateName); len(validationErrors) > 0 {
		var errMsgs []string
		for _, verr := range validationErrors {
			errMsgs = append(errMsgs, verr.Error())
		}
		return nil, fmt.Errorf("erros de validação:\n%s", strings.Join(errMsgs, "\n"))
	}

	lineLength := tmpl.LineLength

	var lines []string
	var warnings []string
	now := time.Now().In(resolver.SaoPauloLocation())
	var totalCents int64

	ctx := &resolver.Context{
		Company:      input.Company,
		Payments:     input.Payments,
		Now:          now,
		LineLength:   lineLength,
		TemplateName: templateName,
		Warnings:     &warnings,
	}

	resolv := resolver.New()

	// 1. Header do arquivo
	headerArquivo, err := generateSegment(tmpl, "header_arquivo", ctx, resolv, 0)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar header arquivo: %w", err)
	}
	lines = append(lines, headerArquivo)

	// 2. Header do lote
	// Para campos do header_lote que precisam de dados do payment, usamos o primeiro payment se disponível
	if len(input.Payments) > 0 {
		ctx.CurrentPayment = &input.Payments[0]
	}

	// Para tributos, escolhe dinamicamente o header, os segmentos e o trailer
	// conforme o tipo (com código de barras -> Segmento O; sem -> Segmento N).
	// O trailer diverge: COM barras usa TOTAL VALOR PAGTOS (24-41) + TOTAL QTDE
	// MOEDA (42-56); SEM barras usa principal/outras/acréscimos/arrecadado.
	headerLoteKey := "header_lote"
	trailerLoteKey := "trailer_lote"
	detailSegments := tmpl.DetailSegments
	if templateName == "cnab240_tributos" && len(input.Payments) > 0 {
		if code := resolver.TaxTypeToPaymentCode(strings.ToUpper(input.Payments[0].TaxType)); isTributoSemCodigoBarras(code) {
			// Só troca o header dinamicamente se o template tiver a variante específica
			// (layout do Itaú). Templates que separam o tributo em arquivos próprios
			// (ex: Santander) mantêm o header_lote padrão.
			if _, ok := tmpl.Segments["header_lote_tributos_sem_codigo_barras"]; ok {
				headerLoteKey = "header_lote_tributos_sem_codigo_barras"
			}
			if _, ok := tmpl.Segments["trailer_lote_tributos_sem_codigo_barras"]; ok {
				trailerLoteKey = "trailer_lote_tributos_sem_codigo_barras"
			}
			detailSegments = []string{"n"}
		} else if len(input.Payments) > 1 {
			// Lote COM código de barras: tipo e forma do header saem de payments[0],
			// mas o Itaú valida CADA guia contra esse par. Guia de prefeitura,
			// de concessionária e de órgão governamental não cabem no mesmo par
			// (ver resolver.FormaPagamentoTributoItau) — as divergentes voltam
			// recusadas. Enquanto o gerador produz um lote por chamada, quem monta
			// a remessa precisa separá-las.
			parLote := parTipoFormaTributo(input.Payments[0].Barcode)
			for i := 1; i < len(input.Payments); i++ {
				parGuia := parTipoFormaTributo(input.Payments[i].Barcode)
				if parGuia == parLote {
					continue
				}
				warnings = append(warnings, fmt.Sprintf(
					"lote de tributos com guias incompatíveis: o header sai com tipo x forma %s (guia do pagamento %s) e o pagamento %s exigiria %s — separe as guias em remessas distintas",
					parLote, input.Payments[0].ExternalID, input.Payments[i].ExternalID, parGuia,
				))
				break
			}
		}
	}
	if len(detailSegments) == 0 {
		detailSegments = []string{"a", "b"}
	}

	// Chave PIX informada num template que não tem onde gravá-la (ex.: o
	// cnab240_pix_conta do Bradesco traz o Segmento B genérico da FEBRABAN, com
	// endereço/vencimento e nenhum campo de chave). O arquivo sai válido, mas pago
	// por agência/conta — o oposto do que quem mandou a chave pediu. Sem aviso isso
	// só apareceria na conciliação.
	if !templateGravaChavePix(tmpl, detailSegments) {
		for i := range input.Payments {
			if input.Payments[i].RecipientPixKey == "" {
				continue
			}
			warnings = append(warnings, fmt.Sprintf(
				"pagamento %s informou chave PIX, mas o template %s do banco %s não possui campo de chave: o pagamento sairá por agência/conta",
				input.Payments[i].ExternalID, templateName, input.BankCode,
			))
			break
		}
	}

	headerLote, err := generateSegment(tmpl, headerLoteKey, ctx, resolv, 0)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar header lote: %w", err)
	}
	lines = append(lines, headerLote)
	ctx.CurrentPayment = nil // Limpa para não afetar os próximos segmentos

	// 3. Detalhes

	recordCount := 0
	// Número Sequencial do Registro no Lote (Nota G004): inicia em 1 e incrementa
	// a cada registro de detalhe efetivamente gerado, independente do pagamento.
	seqInLote := 0
	for i := range input.Payments {
		if err := goCtx.Err(); err != nil {
			return nil, err
		}

		payment := &input.Payments[i]

		// Cria contexto isolado por iteração para evitar mutação compartilhada
		detailCtx := *ctx
		detailCtx.CurrentPayment = payment
		detailCtx.PaymentIndex = i + 1

		for _, segCode := range detailSegments {
			// Remove hífen para encontrar o segmento (ex: J-52 -> j52)
			segKey := "segmento_" + strings.ReplaceAll(strings.ToLower(segCode), "-", "")
			seqInLote++
			segment, err := generateSegment(tmpl, segKey, &detailCtx, resolv, seqInLote)
			if err != nil {
				return nil, fmt.Errorf("erro ao gerar %s para pagamento %d: %w", segKey, i+1, err)
			}
			if segment != "" {
				lines = append(lines, segment)
				recordCount++
			} else {
				// Segmento opcional não gerado: devolve o número sequencial.
				seqInLote--
			}
		}

		totalCents += int64(math.Round(payment.Amount * 100))
	}

	// 4. Trailer do lote
	totalAmount := float64(totalCents) / 100.0
	ctx.RecordCount = recordCount + 2
	ctx.TotalAmount = totalAmount
	trailerLote, err := generateSegment(tmpl, trailerLoteKey, ctx, resolv, 0)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar trailer lote: %w", err)
	}
	lines = append(lines, trailerLote)

	// 5. Trailer do arquivo
	// Nota: O Itaú espera que a contagem de registros no trailer de arquivo
	// EXCLUA o próprio trailer de arquivo (diferente do padrão CNAB 240)
	ctx.TotalFileRecords = len(lines)
	trailerArquivo, err := generateSegment(tmpl, "trailer_arquivo", ctx, resolv, 0)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar trailer arquivo: %w", err)
	}
	lines = append(lines, trailerArquivo)

	lineSeparator := tmpl.LineSeparator
	if lineSeparator == "" {
		lineSeparator = "\r\n"
	}
	// Adiciona CRLF no final do arquivo (padrão CNAB 240 do Itaú)
	content := strings.Join(lines, lineSeparator) + lineSeparator

	return &Result{
		Content:      content,
		TotalRecords: len(input.Payments),
		TotalAmount:  totalAmount,
		Warnings:     warnings,
	}, nil
}

// templateGravaChavePix informa se algum segmento de detalhe do template tem
// campo alimentado pela chave PIX do favorecido. Serve para avisar quando a chave
// chega num layout que não a suporta, em vez de descartá-la em silêncio.
func templateGravaChavePix(tmpl template.Config, detailSegments []string) bool {
	for _, segCode := range detailSegments {
		segKey := "segmento_" + strings.ReplaceAll(strings.ToLower(segCode), "-", "")
		seg, ok := tmpl.Segments[segKey]
		if !ok {
			continue
		}
		for _, field := range seg.SortedFields {
			switch field.Config.Source {
			case "payment.recipient_pix_key", "payment.pix_key":
				return true
			}
		}
	}
	return false
}

// parTipoFormaTributo descreve o par tipo x forma que a guia exige no header do
// lote, no formato "22x19". É o par que o Itaú confere contra cada guia do lote.
func parTipoFormaTributo(barcode string) string {
	return resolver.TipoPagamentoTributoItau(barcode) + "x" + resolver.FormaPagamentoTributoItau(barcode)
}

// isTributoSemCodigoBarras retorna true para códigos de pagamento de tributos
// que NÃO possuem código de barras (DARF, GPS, DARF Simples, IPTU, DARJ, GARE, IPVA, DPVAT, FGTS).
func isTributoSemCodigoBarras(code string) bool {
	switch code {
	case "16", "17", "18", "19", "21", "22", "25", "26", "27", "35":
		return true
	}
	return false
}

func generateSegment(
	tmpl template.Config,
	segmentKey string,
	ctx *resolver.Context,
	resolv *resolver.Resolver,
	seqNum int,
) (string, error) {
	segmentConfig, ok := tmpl.Segments[segmentKey]
	if !ok {
		// Segmento opcional
		return "", nil
	}

	ctx.SeqNum = seqNum

	line := make([]byte, ctx.LineLength)
	for i := range line {
		line[i] = ' '
	}

	for _, f := range segmentConfig.SortedFields {
		value, err := resolveFieldValue(f.Name, f.Config, ctx, resolv, seqNum)
		if err != nil {
			return "", fmt.Errorf("erro ao resolver campo '%s' no segmento '%s': %w", f.Name, segmentKey, err)
		}
		if value != "" {
			start := f.Config.Pos[0]
			end := f.Config.Pos[1]
			if start < 1 || end > len(line) || start > end {
				return "", fmt.Errorf("posição inválida [%d, %d] para campo '%s' no segmento '%s'", start, end, f.Name, segmentKey)
			}
			formatted, err := formatter.FormatValue(value, end-start+1, f.Config.Type)
			if err != nil {
				return "", fmt.Errorf("campo '%s' no segmento '%s': %w", f.Name, segmentKey, err)
			}
			copy(line[start-1:end], formatted)
		}
	}

	return string(line), nil
}

func resolveFieldValue(
	_ string,
	field template.Field,
	ctx *resolver.Context,
	resolv *resolver.Resolver,
	_ int,
) (string, error) {
	// 1. Valor fixo do template
	if field.Value != "" {
		return field.Value, nil
	}

	// 2. Source dinâmico
	if field.Source != "" {
		val, err := resolv.Resolve(field.Source, ctx, field.Format)
		if err != nil && field.Required {
			return "", fmt.Errorf("source '%s': %w", field.Source, err)
		}
		if val == "" {
			return field.Default, nil
		}
		return val, nil
	}

	return field.Default, nil
}
