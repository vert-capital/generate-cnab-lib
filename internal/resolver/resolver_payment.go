package resolver

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/vert-capital/generate-cnab-lib/internal/formatter"
	"github.com/vert-capital/generate-cnab-lib/types"
)

func (r *Resolver) registerPaymentResolvers() {
	r.resolvers["payment.amount"] = paymentResolver(func(p *types.PaymentData) string {
		return formatter.FormatCurrency(p.Amount)
	})
	r.resolvers["payment.external_id"] = paymentResolver(func(p *types.PaymentData) string {
		return p.ExternalID
	})
	r.resolvers["payment.recipient_bank"] = paymentResolver(func(p *types.PaymentData) string {
		return p.RecipientBank
	})
	r.resolvers["payment.recipient_agency"] = paymentResolver(func(p *types.PaymentData) string {
		return p.RecipientAgency
	})
	r.resolvers["payment.recipient_account"] = paymentResolver(func(p *types.PaymentData) string {
		return p.RecipientAccount
	})
	r.resolvers["payment.agencia_conta"] = func(ctx *Context, format string) string {
		if ctx.CurrentPayment == nil {
			return ""
		}
		p := ctx.CurrentPayment

		// Layout específico do Itaú (341) conforme manual SISPAG pág. 58
		if p.RecipientBank == "341" || p.RecipientBank == "409" {
			agencia := formatter.PadLeftZeros(p.RecipientAgency, 4)
			conta := formatter.PadLeftZeros(p.RecipientAccount, 6)
			dac := p.RecipientAccountDigit
			if dac == "" {
				dac = " "
			}
			return "0" + agencia + " " + "000000" + conta + " " + dac
		}

		// Layout genérico para outros bancos
		agencia := formatter.PadLeftZeros(p.RecipientAgency, 5)
		dvAgencia := p.RecipientAgencyDigit
		if dvAgencia == "" {
			dvAgencia = " "
		}
		conta := formatter.PadLeftZeros(p.RecipientAccount, 13)
		dvConta := p.RecipientAccountDigit
		if dvConta == "" {
			dvConta = " "
		}
		return agencia + dvAgencia + conta + dvConta
	}
	r.resolvers["payment.ispb"] = paymentResolver(func(p *types.PaymentData) string {
		return p.ISPB
	})
	r.resolvers["payment.tipo_conta"] = func(ctx *Context, format string) string {
		if ctx.CurrentPayment != nil {
			// Se informado explicitamente, usa o valor fornecido
			if ctx.CurrentPayment.AccountType != "" {
				return ctx.CurrentPayment.AccountType
			}
			if ctx.CurrentPayment.Metadata != nil {
				if val, ok := ctx.CurrentPayment.Metadata["account_type"]; ok {
					return interfaceToString(val)
				}
			}
			// PIX via chave de enderecamento deve usar 04 (Nota 36 Itaú)
			if ctx.CurrentPayment.RecipientPixKey != "" {
				return "04"
			}
		}
		return "01"
	}
	r.resolvers["payment.payment_method"] = func(ctx *Context, format string) string {
		var code string
		if ctx.CurrentPayment != nil {
			method := ctx.CurrentPayment.PaymentMethod

			// Converte valores textuais para códigos numéricos
			if c := TaxTypeToPaymentCode(strings.ToUpper(method)); c != "" {
				code = c
			} else if method != "" {
				code = method
			} else {
				// Default baseado no template
				switch ctx.TemplateName {
				case "cnab240_transferencia":
					if ctx.CurrentPayment.RecipientDocument == ctx.Company.CNPJ {
						code = "43"
					} else {
						code = "41"
					}
				case "cnab240_boleto":
					if len(ctx.CurrentPayment.Barcode) >= 3 && ctx.CurrentPayment.Barcode[0:3] == "341" {
						code = "30"
					} else {
						code = "31"
					}
				case "cnab240_tributos":
					if c := TaxTypeToPaymentCode(strings.ToUpper(ctx.CurrentPayment.TaxType)); c != "" {
						code = c
					} else {
						code = "16"
					}
				case "cnab240_pix_conta":
					code = "45"
				default:
					code = "41"
				}
			}

			// Força regras de titularidade/banco para transferências
			if ctx.TemplateName == "cnab240_transferencia" {
				if ctx.CurrentPayment.RecipientBank == ctx.Company.BankCode {
					return "01"
				}
				if ctx.CurrentPayment.RecipientDocument == ctx.Company.CNPJ {
					return "43"
				}
			}

			// Força regra do banco do código de barras para boletos
			if ctx.TemplateName == "cnab240_boleto" {
				if len(ctx.CurrentPayment.Barcode) >= 3 && ctx.CurrentPayment.Barcode[0:3] == "341" {
					return "30"
				}
				return "31"
			}
		}

		if code == "" {
			code = "41"
		}
		return code
	}
	r.resolvers["payment.camara"] = func(ctx *Context, format string) string {
		if ctx.CurrentPayment != nil && ctx.CurrentPayment.RecipientBank == ctx.Company.BankCode {
			return "000"
		}
		return "018"
	}
	r.resolvers["payment.finalidade_ted"] = func(ctx *Context, format string) string {
		if ctx.CurrentPayment != nil && ctx.CurrentPayment.TEDPurpose != "" {
			return ctx.CurrentPayment.TEDPurpose
		}
		return "00005"
	}
	r.resolvers["payment.recipient_name"] = paymentResolver(func(p *types.PaymentData) string {
		return p.RecipientCompanyName
	})
	r.resolvers["payment.recipient_document"] = paymentResolver(func(p *types.PaymentData) string {
		return p.RecipientDocument
	})
	r.resolvers["payment.recipient_document_type"] = func(ctx *Context, format string) string {
		if ctx.CurrentPayment == nil || ctx.CurrentPayment.RecipientDocument == "" {
			return ""
		}
		if len(ctx.CurrentPayment.RecipientDocument) == 11 {
			return "1"
		}
		return "2"
	}
	r.resolvers["payment.recipient_pix_key"] = paymentResolver(func(p *types.PaymentData) string {
		return p.RecipientPixKey
	})
	r.resolvers["payment.due_date_ddmmyyyy"] = func(ctx *Context, format string) string {
		if ctx.CurrentPayment != nil {
			dueDate := ctx.CurrentPayment.DueDate
			if len(dueDate) == 8 {
				return dueDate[6:8] + dueDate[4:6] + dueDate[0:4]
			}
		}
		return ""
	}
	r.resolvers["payment.due_date"] = paymentResolver(func(p *types.PaymentData) string {
		return p.DueDate
	})
	r.resolvers["payment.txid"] = paymentResolver(func(p *types.PaymentData) string {
		return p.TXID
	})
	r.resolvers["payment.our_number"] = paymentResolver(func(p *types.PaymentData) string {
		return p.OurNumber
	})
	r.resolvers["payment.discount"] = paymentResolver(func(p *types.PaymentData) string {
		return formatter.FormatCurrency(p.Discount)
	})
	r.resolvers["payment.interest"] = paymentResolver(func(p *types.PaymentData) string {
		return formatter.FormatCurrency(p.Interest)
	})
	r.resolvers["payment.pix_key_type"] = func(ctx *Context, format string) string {
		if ctx.CurrentPayment == nil {
			return ""
		}
		var raw string
		if ctx.CurrentPayment.Metadata != nil {
			if val, ok := ctx.CurrentPayment.Metadata["key_type"]; ok {
				raw = interfaceToString(val)
			}
		}
		// Converte códigos do README para códigos oficiais do Itaú (Nota 37)
		// README: 01=Telefone, 02=Email, 03=CPF/CNPJ, 04=Celular, 05=Chave Aleatória
		// Itaú:   01=Telefone, 02=Email, 03=CPF/CNPJ, 04=Chave Aleatória
		switch raw {
		case "05":
			return "04" // Chave Aleatória (EVP)
		case "04":
			return "01" // Celular → Telefone
		case "01", "02", "03":
			return raw
		}
		// Inferência: se a chave for um UUID (EVP), assume 04
		if ctx.CurrentPayment.RecipientPixKey != "" && strings.Contains(ctx.CurrentPayment.RecipientPixKey, "-") {
			return "04"
		}
		return raw
	}
	r.resolvers["payment.tax_type"] = paymentResolver(func(p *types.PaymentData) string {
		return p.TaxType
	})
	r.resolvers["payment.metadata.dados_tributo"] = func(ctx *Context, format string) string {
		if ctx.CurrentPayment == nil || ctx.CurrentPayment.Metadata == nil {
			return ""
		}
		md := ctx.CurrentPayment.Metadata

		// Backwards compatibility: string pronta
		if val, ok := md["dados_tributo"]; ok {
			s := interfaceToString(val)
			if s != "" {
				return s
			}
		}

		// Auto-build baseado no tax_type e nos dados estruturados no metadata
		switch strings.ToUpper(ctx.CurrentPayment.TaxType) {
		case "DARF":
			return buildDARFNormalFromMetadata(md, ctx.CurrentPayment)
		case "DARF_SIMPLES":
			return buildDARFSimplesFromMetadata(md, ctx.CurrentPayment)
		case "GPS":
			return buildGPSFromMetadata(md, ctx.CurrentPayment)
		}
		return ""
	}
}

func formatNum(v float64, length int) string {
	cents := int64(math.Round(v * 100))
	return fmt.Sprintf("%0*d", length, cents)
}

func metaMap(md map[string]interface{}, key string) map[string]interface{} {
	if v, ok := md[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

func metaString(md map[string]interface{}, key string) string {
	if v, ok := md[key]; ok {
		return interfaceToString(v)
	}
	return ""
}

func metaFloat(md map[string]interface{}, key string) (float64, bool) {
	if v, ok := md[key]; ok {
		switch val := v.(type) {
		case float64:
			return val, true
		case int:
			return float64(val), true
		case int64:
			return float64(val), true
		case string:
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

func defaultIfEmpty(val, fallback string) string {
	if strings.TrimSpace(val) != "" {
		return val
	}
	return fallback
}

func toDDMMYYYY(date string) string {
	if len(date) == 8 {
		return date[6:8] + date[4:6] + date[0:4]
	}
	return date
}

func buildDARFNormalFromMetadata(md map[string]interface{}, p *types.PaymentData) string {
	m := metaMap(md, "darf_normal")
	if m == nil {
		return ""
	}
	tributo := "02"
	receita := defaultIfEmpty(metaString(m, "receita"), p.RevenueCode)
	tipoInscricao := "2"
	if s := metaString(m, "tipo_inscricao"); s != "" {
		tipoInscricao = s
	}
	numeroInscricao := defaultIfEmpty(metaString(m, "numero_inscricao"), p.RecipientDocument)
	periodo := defaultIfEmpty(metaString(m, "periodo"), p.Competence)
	referencia := metaString(m, "referencia")
	valorPrincipal := p.Amount
	multa := 0.0
	jurosEncargos := 0.0
	valorTotal := p.Amount
	dataVencimento := toDDMMYYYY(p.DueDate)
	dataPagamento := toDDMMYYYY(p.DueDate)
	contribuinte := p.RecipientCompanyName

	if v, ok := metaFloat(m, "valor_principal"); ok {
		valorPrincipal = v
	}
	if v, ok := metaFloat(m, "multa"); ok {
		multa = v
	}
	if v, ok := metaFloat(m, "juros_encargos"); ok {
		jurosEncargos = v
	}
	if v, ok := metaFloat(m, "valor_total"); ok {
		valorTotal = v
	}
	if s := metaString(m, "data_vencimento"); s != "" {
		dataVencimento = toDDMMYYYY(s)
	}
	if s := metaString(m, "data_pagamento"); s != "" {
		dataPagamento = toDDMMYYYY(s)
	}
	if s := metaString(m, "contribuinte"); s != "" {
		contribuinte = s
	}

	var b strings.Builder
	b.Grow(178)
	b.WriteString(formatter.PadLeftZeros(tributo, 2))
	b.WriteString(formatter.PadLeftZeros(receita, 4))
	b.WriteString(formatter.PadLeftZeros(tipoInscricao, 1))
	b.WriteString(formatter.PadLeftZeros(numeroInscricao, 14))
	b.WriteString(formatter.PadLeftZeros(periodo, 8))
	b.WriteString(formatter.PadLeftZeros(referencia, 17))
	b.WriteString(formatNum(valorPrincipal, 14))
	b.WriteString(formatNum(multa, 14))
	b.WriteString(formatNum(jurosEncargos, 14))
	b.WriteString(formatNum(valorTotal, 14))
	b.WriteString(formatter.PadLeftZeros(dataVencimento, 8))
	b.WriteString(formatter.PadLeftZeros(dataPagamento, 8))
	b.WriteString(strings.Repeat(" ", 30)) // brancos
	b.WriteString(formatter.PadRight(contribuinte, 30))
	return b.String()
}

func buildDARFSimplesFromMetadata(md map[string]interface{}, p *types.PaymentData) string {
	m := metaMap(md, "darf_simples")
	if m == nil {
		return ""
	}
	tributo := "03"
	receita := defaultIfEmpty(metaString(m, "receita"), p.RevenueCode)
	tipoInscricao := "2"
	if s := metaString(m, "tipo_inscricao"); s != "" {
		tipoInscricao = s
	}
	numeroInscricao := defaultIfEmpty(metaString(m, "numero_inscricao"), p.RecipientDocument)
	periodo := defaultIfEmpty(metaString(m, "periodo"), p.Competence)
	receitaBruta := 0.0
	percentual := 0.0
	valorPrincipal := p.Amount
	multa := 0.0
	jurosEncargos := 0.0
	valorTotal := p.Amount
	dataVencimento := toDDMMYYYY(p.DueDate)
	dataPagamento := toDDMMYYYY(p.DueDate)
	contribuinte := p.RecipientCompanyName

	if v, ok := metaFloat(m, "receita_bruta"); ok {
		receitaBruta = v
	}
	if v, ok := metaFloat(m, "percentual"); ok {
		percentual = v
	}
	if v, ok := metaFloat(m, "valor_principal"); ok {
		valorPrincipal = v
	}
	if v, ok := metaFloat(m, "multa"); ok {
		multa = v
	}
	if v, ok := metaFloat(m, "juros_encargos"); ok {
		jurosEncargos = v
	}
	if v, ok := metaFloat(m, "valor_total"); ok {
		valorTotal = v
	}
	if s := metaString(m, "data_vencimento"); s != "" {
		dataVencimento = toDDMMYYYY(s)
	}
	if s := metaString(m, "data_pagamento"); s != "" {
		dataPagamento = toDDMMYYYY(s)
	}
	if s := metaString(m, "contribuinte"); s != "" {
		contribuinte = s
	}

	var b strings.Builder
	b.Grow(178)
	b.WriteString(formatter.PadLeftZeros(tributo, 2))
	b.WriteString(formatter.PadLeftZeros(receita, 4))
	b.WriteString(formatter.PadLeftZeros(tipoInscricao, 1))
	b.WriteString(formatter.PadLeftZeros(numeroInscricao, 14))
	b.WriteString(formatter.PadLeftZeros(periodo, 8))
	b.WriteString(formatNum(receitaBruta, 9))
	b.WriteString(formatNum(percentual, 4))
	b.WriteString(strings.Repeat(" ", 4)) // brancos
	b.WriteString(formatNum(valorPrincipal, 14))
	b.WriteString(formatNum(multa, 14))
	b.WriteString(formatNum(jurosEncargos, 14))
	b.WriteString(formatNum(valorTotal, 14))
	b.WriteString(formatter.PadLeftZeros(dataVencimento, 8))
	b.WriteString(formatter.PadLeftZeros(dataPagamento, 8))
	b.WriteString(strings.Repeat(" ", 30)) // brancos
	b.WriteString(formatter.PadRight(contribuinte, 30))
	return b.String()
}

func buildGPSFromMetadata(md map[string]interface{}, p *types.PaymentData) string {
	m := metaMap(md, "gps")
	if m == nil {
		return ""
	}
	tributo := "01"
	codigoPagto := metaString(m, "codigo_pagto")
	competencia := defaultIfEmpty(metaString(m, "competencia"), p.Competence)
	identificador := defaultIfEmpty(metaString(m, "identificador"), p.RecipientDocument)
	valorTributo := p.Amount
	valorOutrEntidade := 0.0
	atualizMonetaria := 0.0
	valorArrecadado := p.Amount
	dataArrecadacao := toDDMMYYYY(p.DueDate)
	usoEmpresa := metaString(m, "uso_empresa")
	contribuinte := p.RecipientCompanyName

	if v, ok := metaFloat(m, "valor_tributo"); ok {
		valorTributo = v
	}
	if v, ok := metaFloat(m, "valor_outr_entidade"); ok {
		valorOutrEntidade = v
	}
	if v, ok := metaFloat(m, "atualiz_monetaria"); ok {
		atualizMonetaria = v
	}
	if v, ok := metaFloat(m, "valor_arrecadado"); ok {
		valorArrecadado = v
	}
	if s := metaString(m, "data_arrecadacao"); s != "" {
		dataArrecadacao = toDDMMYYYY(s)
	}
	if s := metaString(m, "contribuinte"); s != "" {
		contribuinte = s
	}

	var b strings.Builder
	b.Grow(178)
	b.WriteString(formatter.PadLeftZeros(tributo, 2))
	b.WriteString(formatter.PadLeftZeros(codigoPagto, 4))
	b.WriteString(formatter.PadLeftZeros(competencia, 6))
	b.WriteString(formatter.PadLeftZeros(identificador, 14))
	b.WriteString(formatNum(valorTributo, 14))
	b.WriteString(formatNum(valorOutrEntidade, 14))
	b.WriteString(formatNum(atualizMonetaria, 14))
	b.WriteString(formatNum(valorArrecadado, 14))
	b.WriteString(formatter.PadLeftZeros(dataArrecadacao, 8))
	b.WriteString(strings.Repeat(" ", 8)) // brancos
	b.WriteString(formatter.PadRight(usoEmpresa, 50))
	b.WriteString(formatter.PadRight(contribuinte, 30))
	return b.String()
}

func (r *Resolver) registerBarcodeResolvers() {
	r.resolvers["payment.barcode"] = paymentResolver(func(p *types.PaymentData) string {
		return normalizeBarcode(p)
	})
	r.resolvers["payment.barcode_bank"] = func(ctx *Context, format string) string {
		bc := normalizeBarcode(ctx.CurrentPayment)
		if len(bc) >= 3 {
			return bc[0:3]
		}
		return ""
	}
	r.resolvers["payment.barcode_currency"] = func(ctx *Context, format string) string {
		bc := normalizeBarcode(ctx.CurrentPayment)
		if len(bc) >= 4 {
			return bc[3:4]
		}
		return ""
	}
	r.resolvers["payment.barcode_dv"] = func(ctx *Context, format string) string {
		bc := normalizeBarcode(ctx.CurrentPayment)
		if len(bc) >= 5 {
			return bc[4:5]
		}
		return ""
	}
	r.resolvers["payment.barcode_due_factor"] = func(ctx *Context, format string) string {
		bc := normalizeBarcode(ctx.CurrentPayment)
		if len(bc) >= 9 {
			return bc[5:9]
		}
		return ""
	}
	r.resolvers["payment.barcode_amount"] = func(ctx *Context, format string) string {
		bc := normalizeBarcode(ctx.CurrentPayment)
		if len(bc) >= 19 {
			return bc[9:19]
		}
		return ""
	}
	r.resolvers["payment.barcode_free_field"] = func(ctx *Context, format string) string {
		bc := normalizeBarcode(ctx.CurrentPayment)
		if len(bc) >= 44 {
			return bc[19:44]
		}
		return ""
	}
}
