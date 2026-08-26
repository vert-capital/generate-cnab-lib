// Package validation contém funções para validar campos de entrada
package validation

import (
	"fmt"
	"strings"

	"github.com/vert-capital/generate-cnab-lib/internal/resolver"
	"github.com/vert-capital/generate-cnab-lib/internal/template"
	"github.com/vert-capital/generate-cnab-lib/types"
)

// ValidationError representa um erro de validação
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validação falhou para campo '%s': %s", e.Field, e.Message)
}

type validator struct {
	errors []ValidationError
}

func newValidator() *validator {
	return &validator{errors: make([]ValidationError, 0)}
}

func (v *validator) add(field, message string) {
	v.errors = append(v.errors, ValidationError{Field: field, Message: message})
}

func (v *validator) check(field, value string, rule template.InputValidation) {
	if !rule.Required && value == "" {
		return
	}
	if rule.Required && strings.TrimSpace(value) == "" {
		v.add(field, "campo obrigatório não preenchido")
		return
	}

	length := len(value)

	if rule.ExactLength > 0 && length != rule.ExactLength {
		v.add(field, fmt.Sprintf("tamanho deve ser exatamente %d caracteres, recebido %d", rule.ExactLength, length))
		return
	}
	if rule.MinLength > 0 && length < rule.MinLength {
		v.add(field, fmt.Sprintf("tamanho mínimo é %d caracteres, recebido %d", rule.MinLength, length))
		return
	}
	if rule.MaxLength > 0 && length > rule.MaxLength {
		v.add(field, fmt.Sprintf("tamanho máximo é %d caracteres, recebido %d", rule.MaxLength, length))
		return
	}
	if rule.NumericOnly {
		for _, r := range value {
			if r < '0' || r > '9' {
				v.add(field, "deve conter apenas números")
				return
			}
		}
	}
}

// ValidateInput valida os dados de entrada contra as regras fixas do sistema
// e contra as regras declarativas definidas no template JSON.
func ValidateInput(input types.Input, templateName string) []ValidationError {
	v := newValidator()

	// Regras fixas (invariantes para todos os templates)
	v.check("bank_code", input.BankCode, template.InputValidation{Required: true, ExactLength: 3, NumericOnly: true})
	v.check("company.cnpj", input.Company.CNPJ, template.InputValidation{Required: true, ExactLength: 14, NumericOnly: true})
	v.check("company.company_name", input.Company.CompanyName, template.InputValidation{Required: true, MaxLength: 30})
	v.check("company.agency", input.Company.Agency, template.InputValidation{Required: true, MaxLength: 5})
	v.check("company.account", input.Company.Account, template.InputValidation{Required: true, MaxLength: 12})
	v.check("company.account_digit", input.Company.AccountDigit, template.InputValidation{MaxLength: 1})

	if input.Company.BankCode != "" && input.BankCode != input.Company.BankCode {
		v.add("company.bank_code", fmt.Sprintf("diverge do bank_code do input '%s'", input.BankCode))
	}

	// Carrega validações declarativas do template
	tmpl, _ := template.Get(input.BankCode, templateName)
	resolv := resolver.New()

	for i := range input.Payments {
		prefix := fmt.Sprintf("payments[%d].", i)
		payment := &input.Payments[i]

		// Documento: CPF=11 ou CNPJ=14 (regra de negócio)
		if docLen := len(payment.RecipientDocument); docLen != 0 && docLen != 11 && docLen != 14 {
			v.add(prefix+"recipient_document",
				fmt.Sprintf("tamanho deve ser 11 (CPF) ou 14 (CNPJ) caracteres, recebido %d", docLen))
		}

		// Amount: deve ser > 0 (regra de negócio)
		if payment.Amount <= 0 {
			v.add(prefix+"amount", "valor deve ser maior que zero")
		}

		// Validações declarativas do template via Resolver
		ctx := &resolver.Context{Company: input.Company, CurrentPayment: payment, TemplateName: templateName}
		for _, rule := range tmpl.InputValidations {
			value, _ := resolv.Resolve(rule.Source, ctx, "")
			v.check(prefix+sourceLabel(rule.Source), value, rule)
		}

		// Tributos liquidados via código de barras (forma 13/91, ex.: DARE-SP,
		// GNRE, concessionárias) exigem o barcode: valor, código de receita e
		// datas são derivados dele. Sem barcode o banco rejeita o lote inteiro
		// (VALOR CALCULADO DIFERENTE, CODIGO DA RECEITA INVALIDO, etc.).
		if templateName == "cnab240_tributos" {
			forma, _ := resolv.Resolve("payment.payment_method", ctx, "")
			if resolver.IsBarcodeTributo(forma) {
				if strings.TrimSpace(payment.Barcode) == "" {
					v.add(prefix+"barcode",
						fmt.Sprintf("obrigatório para tributo com código de barras (forma %s, ex.: DARE/GNRE)", forma))
				} else {
					// A guia deve fechar em 44 dígitos sem os DVs de campo. A lib
					// aceita tanto o código de barras (44) quanto a representação
					// numérica (48); qualquer outro comprimento é inválido. No
					// arquivo vai a representação numérica de 48 posições.
					normalized, _ := resolv.Resolve("payment.barcode_tributo_44", ctx, "")
					v.check(prefix+"barcode", normalized, template.InputValidation{
						ExactLength: 44, NumericOnly: true,
					})
				}
			}
		}
	}

	return v.errors
}

// sourceLabel converte "payment.barcode" -> "barcode" para mensagens de erro legíveis.
func sourceLabel(source string) string {
	if i := strings.LastIndex(source, "."); i >= 0 {
		return source[i+1:]
	}
	return source
}
