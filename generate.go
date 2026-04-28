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

	// Para tributos, escolhe dinamicamente o header e os segmentos conforme o tipo
	headerLoteKey := "header_lote"
	detailSegments := tmpl.DetailSegments
	if templateName == "cnab240_tributos" && len(input.Payments) > 0 {
		if code := resolver.TaxTypeToPaymentCode(strings.ToUpper(input.Payments[0].TaxType)); isTributoSemCodigoBarras(code) {
			headerLoteKey = "header_lote_tributos_sem_codigo_barras"
			detailSegments = []string{"n"}
		}
	}
	if len(detailSegments) == 0 {
		detailSegments = []string{"a", "b"}
	}

	headerLote, err := generateSegment(tmpl, headerLoteKey, ctx, resolv, 0)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar header lote: %w", err)
	}
	lines = append(lines, headerLote)
	ctx.CurrentPayment = nil // Limpa para não afetar os próximos segmentos

	// 3. Detalhes

	recordCount := 0
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
			segment, err := generateSegment(tmpl, segKey, &detailCtx, resolv, i+1)
			if err != nil {
				return nil, fmt.Errorf("erro ao gerar %s para pagamento %d: %w", segKey, i+1, err)
			}
			if segment != "" {
				lines = append(lines, segment)
				recordCount++
			}
		}

		totalCents += int64(math.Round(payment.Amount * 100))
	}

	// 4. Trailer do lote
	totalAmount := float64(totalCents) / 100.0
	ctx.RecordCount = recordCount + 2
	ctx.TotalAmount = totalAmount
	trailerLote, err := generateSegment(tmpl, "trailer_lote", ctx, resolv, 0)
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

// isTributoSemCodigoBarras retorna true para códigos de pagamento de tributos
// que NÃO possuem código de barras (DARF, GPS, DARF Simples, IPTU, DARJ, GARE, IPVA, DPVAT, FGTS).
func isTributoSemCodigoBarras(code string) bool {
	switch code {
	case "16", "17", "18", "19", "21", "22", "25", "27", "35":
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
		return val, nil
	}

	return "", nil
}
