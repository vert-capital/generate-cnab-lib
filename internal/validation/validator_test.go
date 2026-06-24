package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vert-capital/generate-cnab-lib/internal/template"
	"github.com/vert-capital/generate-cnab-lib/types"
)

// helper: cria um validator interno e chama check
func checkRule(value string, rule template.InputValidation) []ValidationError {
	v := newValidator()
	v.check("field", value, rule)
	return v.errors
}

func TestCheck_Required(t *testing.T) {
	errs := checkRule("", template.InputValidation{Required: true})
	assert.Len(t, errs, 1)
	assert.Equal(t, "field", errs[0].Field)
}

func TestCheck_NotRequired_Empty(t *testing.T) {
	errs := checkRule("", template.InputValidation{Required: false})
	assert.Empty(t, errs)
}

func TestCheck_ExactLength_Fail(t *testing.T) {
	errs := checkRule("123", template.InputValidation{Required: true, ExactLength: 14})
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "exatamente 14")
}

func TestCheck_ExactLength_OK(t *testing.T) {
	errs := checkRule("12345678000195", template.InputValidation{Required: true, ExactLength: 14})
	assert.Empty(t, errs)
}

func TestCheck_MinLength_Fail(t *testing.T) {
	errs := checkRule("AB", template.InputValidation{Required: true, MinLength: 3})
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "mínimo")
}

func TestCheck_MaxLength_Fail(t *testing.T) {
	errs := checkRule("ABCDEFGHIJ", template.InputValidation{Required: true, MaxLength: 5})
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "máximo")
}

func TestCheck_NumericOnly_Fail(t *testing.T) {
	errs := checkRule("1234ABC", template.InputValidation{Required: true, NumericOnly: true})
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "apenas números")
}

func TestCheck_NumericOnly_OK(t *testing.T) {
	errs := checkRule("12345678000195", template.InputValidation{Required: true, NumericOnly: true})
	assert.Empty(t, errs)
}

func TestValidateInput_BankCode(t *testing.T) {
	input := types.Input{
		BankCode: "",
		Company: types.CompanyData{
			CNPJ:        "12345678000195",
			CompanyName: "EMPRESA TESTE",
			Agency:      "1234",
			Account:     "123456",
		},
	}
	errs := ValidateInput(input, "cnab240_pix_conta")
	assert.True(t, len(errs) > 0)

	found := false
	for _, e := range errs {
		if e.Field == "bank_code" {
			found = true
		}
	}
	assert.True(t, found, "deve ter erro de bank_code")
}

func TestValidateInput_ValidPIX(t *testing.T) {
	input := types.Input{
		BankCode: "341",
		Company: types.CompanyData{
			CNPJ:         "12345678000195",
			CompanyName:  "EMPRESA TESTE",
			Agency:       "1234",
			Account:      "123456",
			AccountDigit: "5",
		},
		Payments: []types.PaymentData{
			{
				ExternalID:           "PAY-001",
				RecipientDocument:    "12345678901",
				RecipientCompanyName: "FAVORECIDO",
				RecipientBank:        "341",
				ISPB:                 "60701190",
				Amount:               100.00,
			},
		},
	}
	errs := ValidateInput(input, "cnab240_pix_conta")
	assert.Empty(t, errs)
}

func TestValidateInput_InvalidCNPJ(t *testing.T) {
	input := types.Input{
		BankCode: "341",
		Company: types.CompanyData{
			CNPJ:        "123",
			CompanyName: "EMPRESA TESTE",
			Agency:      "1234",
			Account:     "123456",
		},
	}
	errs := ValidateInput(input, "cnab240_pix_conta")
	found := false
	for _, e := range errs {
		if e.Field == "company.cnpj" {
			found = true
		}
	}
	assert.True(t, found, "deve ter erro de CNPJ")
}

func TestValidateInput_RecipientDocument(t *testing.T) {
	input := types.Input{
		BankCode: "341",
		Company: types.CompanyData{
			CNPJ:        "12345678000195",
			CompanyName: "EMPRESA TESTE",
			Agency:      "1234",
			Account:     "123456",
		},
		Payments: []types.PaymentData{
			{
				ExternalID:           "PAY-001",
				RecipientDocument:    "12345", // nem CPF nem CNPJ
				RecipientCompanyName: "FAVORECIDO",
			},
		},
	}
	errs := ValidateInput(input, "cnab240_transferencia")
	found := false
	for _, e := range errs {
		if e.Field == "payments[0].recipient_document" {
			found = true
		}
	}
	assert.True(t, found, "deve ter erro de recipient_document")
}

func TestValidateInput_Boleto_Barcode44(t *testing.T) {
	input := types.Input{
		BankCode: "341",
		Company: types.CompanyData{
			CNPJ:        "12345678000195",
			CompanyName: "EMPRESA TESTE",
			Agency:      "1234",
			Account:     "123456",
		},
		Payments: []types.PaymentData{
			{
				ExternalID:           "PAY-001",
				RecipientCompanyName: "FAVORECIDO",
				Barcode:              "34191790000015007501111222233334445556677777",
			},
		},
	}
	errs := ValidateInput(input, "cnab240_boleto")
	// barcode tem 44 chars - OK para boleto
	for _, e := range errs {
		assert.NotContains(t, e.Field, "barcode")
	}
}

func TestValidateInput_Boleto_LinhaDigitavel47(t *testing.T) {
	input := types.Input{
		BankCode: "341",
		Company: types.CompanyData{
			CNPJ:        "12345678000195",
			CompanyName: "EMPRESA TESTE",
			Agency:      "1234",
			Account:     "123456",
		},
		Payments: []types.PaymentData{
			{
				ExternalID:           "PAY-001",
				RecipientCompanyName: "FAVORECIDO",
				Barcode:              "34191090080004519091500573140001214260002835578",
			},
		},
	}
	errs := ValidateInput(input, "cnab240_boleto")
	// linha digitável 47 posições deve ser convertida e validada como 44
	for _, e := range errs {
		assert.NotContains(t, e.Field, "barcode")
	}
}

func TestValidateInput_Tributo_Barcode48Validated(t *testing.T) {
	input := types.Input{
		BankCode: "341",
		Company: types.CompanyData{
			CNPJ:        "12345678000195",
			CompanyName: "EMPRESA TESTE",
			Agency:      "1234",
			Account:     "123456",
		},
		Payments: []types.PaymentData{
			{
				ExternalID:           "PAY-001",
				RecipientCompanyName: "FAVORECIDO",
				Barcode:              "34191790000015007501111222233334445556677777",
				TaxType:              "DARF",
				Amount:               150.0,
			},
		},
	}
	errs := ValidateInput(input, "cnab240_tributos")
	found := false
	for _, e := range errs {
		if e.Field == "payments[0].barcode" {
			found = true
		}
	}
	assert.True(t, found, "tributo com barcode 44 chars deve falhar (requer 48)")
}

func TestValidateInput_Tributo_SemBarcode_OK(t *testing.T) {
	input := types.Input{
		BankCode: "341",
		Company: types.CompanyData{
			CNPJ:        "12345678000195",
			CompanyName: "EMPRESA TESTE",
			Agency:      "1234",
			Account:     "123456",
		},
		Payments: []types.PaymentData{
			{
				ExternalID:           "PAY-001",
				RecipientCompanyName: "FAVORECIDO",
				TaxType:              "DARF_SIMPLES",
				Amount:               37.50,
			},
		},
	}
	errs := ValidateInput(input, "cnab240_tributos")
	for _, e := range errs {
		assert.NotEqual(t, "payments[0].barcode", e.Field, "DARF Simples não deve exigir barcode")
	}
}

func TestValidateInput_Tributo_TaxTypeRequired(t *testing.T) {
	input := types.Input{
		BankCode: "341",
		Company: types.CompanyData{
			CNPJ:        "12345678000195",
			CompanyName: "EMPRESA TESTE",
			Agency:      "1234",
			Account:     "123456",
		},
		Payments: []types.PaymentData{
			{
				ExternalID:           "PAY-001",
				RecipientCompanyName: "FAVORECIDO",
				Barcode:              "341917900000150075011112222333344455566777770000",
			},
		},
	}
	errs := ValidateInput(input, "cnab240_tributos")
	found := false
	for _, e := range errs {
		if e.Field == "payments[0].tax_type" {
			found = true
		}
	}
	assert.True(t, found, "tributo sem tax_type deve falhar")
}

func TestValidationError_Error(t *testing.T) {
	e := ValidationError{Field: "campo", Message: "obrigatório"}
	assert.Equal(t, "validação falhou para campo 'campo': obrigatório", e.Error())
}

func TestValidateInput_AmountZero(t *testing.T) {
	input := types.Input{
		BankCode: "341",
		Company: types.CompanyData{
			CNPJ:         "12345678000195",
			CompanyName:  "EMPRESA TESTE",
			Agency:       "1234",
			Account:      "123456",
			AccountDigit: "5",
		},
		Payments: []types.PaymentData{
			{
				ExternalID:           "PAY-001",
				RecipientDocument:    "12345678901",
				RecipientCompanyName: "FAVORECIDO",
				RecipientBank:        "341",
				ISPB:                 "60701190",
				Amount:               0,
			},
		},
	}
	errs := ValidateInput(input, "cnab240_pix_conta")
	found := false
	for _, e := range errs {
		if e.Field == "payments[0].amount" {
			found = true
		}
	}
	assert.True(t, found, "deve ter erro de amount zero")
}

func TestValidateInput_AmountNegative(t *testing.T) {
	input := types.Input{
		BankCode: "341",
		Company: types.CompanyData{
			CNPJ:         "12345678000195",
			CompanyName:  "EMPRESA TESTE",
			Agency:       "1234",
			Account:      "123456",
			AccountDigit: "5",
		},
		Payments: []types.PaymentData{
			{
				ExternalID:           "PAY-001",
				RecipientDocument:    "12345678901",
				RecipientCompanyName: "FAVORECIDO",
				RecipientBank:        "341",
				ISPB:                 "60701190",
				Amount:               -100.00,
			},
		},
	}
	errs := ValidateInput(input, "cnab240_pix_conta")
	found := false
	for _, e := range errs {
		if e.Field == "payments[0].amount" {
			found = true
		}
	}
	assert.True(t, found, "deve ter erro de amount negativo")
}
