package resolver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vert-capital/generate-cnab-lib/types"
)

func TestNewResolver(t *testing.T) {
	r := New()
	require.NotNil(t, r)
	assert.NotNil(t, r.resolvers)
}

func TestResolveCompanyFields(t *testing.T) {
	r := New()
	ctx := &Context{
		Company: types.CompanyData{
			CNPJ:         "12345678000195",
			CompanyName:  "EMPRESA TESTE",
			BankCode:     "341",
			Agency:       "1234",
			AgencyDigit:  "5",
			Account:      "123456",
			AccountDigit: "7",
			Convenio:     "123456",
		},
		Now: time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC),
	}

	tests := []struct {
		source   string
		expected string
	}{
		{"company.cnpj", "12345678000195"},
		{"company.name", "EMPRESA TESTE"},
		{"company.bank_code", "341"},
		{"company.agency", "1234"},
		{"company.agency_digit", "5"},
		{"company.account", "123456"},
		{"company.account_digit", "7"},
		{"company.convenio", "123456"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			result, err := r.Resolve(tt.source, ctx, "")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolvePaymentFields(t *testing.T) {
	r := New()
	payment := &types.PaymentData{
		ExternalID:           "PAY-001",
		RecipientDocument:    "12345678901",
		RecipientCompanyName: "JOSE DA SILVA",
		RecipientBank:        "341",
		RecipientAgency:      "5678",
		RecipientAccount:     "876543",
		ISPB:                 "60701190",
		Amount:               1500.75,
		DueDate:              "20260330",
		Barcode:              "34191790000015007501111222233334445556677777",
		TXID:                 "TXID123456",
		OurNumber:            "ON123456",
	}
	ctx := &Context{
		CurrentPayment: payment,
	}

	tests := []struct {
		source   string
		expected string
	}{
		{"payment.amount", "000000000150075"},
		{"payment.external_id", "PAY-001"},
		{"payment.recipient_bank", "341"},
		{"payment.recipient_agency", "5678"},
		{"payment.recipient_account", "876543"},
		{"payment.recipient_name", "JOSE DA SILVA"},
		{"payment.recipient_document", "12345678901"},
		{"payment.recipient_document_type", "1"},
		{"payment.ispb", "60701190"},
		{"payment.tipo_conta", "01"},
		{"payment.barcode", "34191790000015007501111222233334445556677777"},
		{"payment.barcode_bank", "341"},
		{"payment.barcode_currency", "9"},
		{"payment.barcode_dv", "1"},
		{"payment.barcode_due_factor", "7900"},
		{"payment.barcode_amount", "0001500750"},
		{"payment.barcode_free_field", "1111222233334445556677777"},
		{"payment.due_date", "20260330"},
		{"payment.due_date_ddmmyyyy", "30032026"},
		{"payment.txid", "TXID123456"},
		{"payment.our_number", "ON123456"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			result, err := r.Resolve(tt.source, ctx, "")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveContextFields(t *testing.T) {
	now := time.Date(2026, 4, 1, 10, 30, 45, 0, SaoPauloLocation())
	r := New()
	ctx := &Context{
		Now:              now,
		RecordCount:      42,
		TotalFileRecords: 100,
		TotalAmount:      5000.00,
	}

	tests := []struct {
		source   string
		expected string
	}{
		{"context.now_date", "20260401"},
		{"context.now_date_ddmmyyyy", "01042026"},
		{"context.now_time", "103045"},
		{"context.now_datetime", "20260401103045"},
		{"context.record_count", "000042"},
		{"context.total_amount", "500000"},
		{"context.lote_servico", "0001"},
		{"context.total_file_records", "000100"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			result, err := r.Resolve(tt.source, ctx, "")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveISPBWithFallback(t *testing.T) {
	r := New()

	tests := []struct {
		name     string
		payment  types.PaymentData
		expected string
	}{
		{
			name: "with explicit ispb",
			payment: types.PaymentData{
				ISPB:          "00416968",
				RecipientBank: "341",
			},
			expected: "00416968",
		},
		{
			name: "empty ispb returns empty",
			payment: types.PaymentData{
				ISPB:          "",
				RecipientBank: "341",
			},
			expected: "",
		},
		{
			name:     "nil payment returns empty",
			payment:  types.PaymentData{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{CurrentPayment: &tt.payment}
			result, err := r.Resolve("payment.ispb", ctx, "")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveAgenciaConta(t *testing.T) {
	r := New()
	payment := &types.PaymentData{
		RecipientAgency:  "123",
		RecipientAccount: "456789",
	}
	ctx := &Context{
		CurrentPayment: payment,
	}

	result, err := r.Resolve("payment.agencia_conta", ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "00123 0000000456789 ", result)
	assert.Len(t, result, 20)
}

func TestResolveAgenciaConta_Itau(t *testing.T) {
	r := New()
	payment := &types.PaymentData{
		RecipientBank:         "341",
		RecipientAgency:       "0910",
		RecipientAccount:      "98980",
		RecipientAccountDigit: "5",
	}
	ctx := &Context{
		CurrentPayment: payment,
	}

	result, err := r.Resolve("payment.agencia_conta", ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "00910 000000098980 5", result)
	assert.Len(t, result, 20)
}

func TestResolveMetadata(t *testing.T) {
	r := New()
	payment := &types.PaymentData{
		ExternalID: "PAY-001",
		Metadata: map[string]interface{}{
			"recipient_email": "test@example.com",
			"finalidade_ted":  "00010",
			"tipo_pagamento":  "20",
			"valor_decimal":   123.45,
			"valor_inteiro":   100,
			"ativo":           true,
		},
	}
	ctx := &Context{
		CurrentPayment: payment,
	}

	tests := []struct {
		source   string
		expected string
	}{
		{"payment.metadata.recipient_email", "test@example.com"},
		{"payment.metadata.finalidade_ted", "00010"},
		{"payment.metadata.tipo_pagamento", "20"},
		{"payment.metadata.valor_decimal", "123.45"},
		{"payment.metadata.valor_inteiro", "100"},
		{"payment.metadata.ativo", "1"},
		{"payment.metadata.inexistente", ""},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			result, err := r.Resolve(tt.source, ctx, "")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveFinalidadeTED(t *testing.T) {
	r := New()

	t.Run("with TEDPurpose set", func(t *testing.T) {
		payment := &types.PaymentData{TEDPurpose: "00010"}
		ctx := &Context{CurrentPayment: payment}
		result, err := r.Resolve("payment.finalidade_ted", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "00010", result)
	})

	t.Run("default when empty", func(t *testing.T) {
		payment := &types.PaymentData{}
		ctx := &Context{CurrentPayment: payment}
		result, err := r.Resolve("payment.finalidade_ted", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "00005", result)
	})
}

func TestResolveUnknownSource(t *testing.T) {
	r := New()
	ctx := &Context{}

	result, err := r.Resolve("unknown.field", ctx, "")
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestSaoPauloLocation(t *testing.T) {
	loc := SaoPauloLocation()
	require.NotNil(t, loc)
	assert.Contains(t, loc.String(), "Sao_Paulo")
}

func TestResolveFormaPagamento(t *testing.T) {
	r := New()

	tests := []struct {
		name              string
		method            string
		taxType           string
		templateName      string
		recipientDocument string
		companyCNPJ       string
		recipientBank     string
		companyBankCode   string
		expected          string
	}{
		// Textual payment methods
		{"method TED", "TED", "", "", "", "", "", "", "41"},
		{"method DOC", "DOC", "", "", "", "", "", "", "03"},
		{"method CREDITO", "CREDITO", "", "", "", "", "", "", "01"},
		{"method CREDITO_EM_CONTA", "CREDITO_EM_CONTA", "", "", "", "", "", "", "01"},
		{"method BOLETO", "BOLETO", "", "", "", "", "", "", "30"},
		{"method BOLETO_ITAU", "BOLETO_ITAU", "", "", "", "", "", "", "30"},
		{"method BOLETO_OUTROS", "BOLETO_OUTROS", "", "", "", "", "", "", "31"},
		{"method PIX", "PIX", "", "", "", "", "", "", "45"},
		{"method DARF", "DARF", "", "", "", "", "", "", "16"},
		{"method GPS", "GPS", "", "", "", "", "", "", "17"},
		{"method DARF_SIMPLES", "DARF_SIMPLES", "", "", "", "", "", "", "18"},
		{"method GARE_SP", "GARE_SP", "", "", "", "", "", "", "22"},
		{"method GARE_SP_ICMS", "GARE_SP_ICMS", "", "", "", "", "", "", "22"},
		{"method IPVA", "IPVA", "", "", "", "", "", "", "25"},
		{"method DPVAT", "DPVAT", "", "", "", "", "", "", "27"},
		{"method FGTS", "FGTS", "", "", "", "", "", "", "35"},
		{"method lowercase ted", "ted", "", "", "", "", "", "", "41"},

		// Numeric passthrough
		{"numeric code 41", "41", "", "", "", "", "", "", "41"},
		{"numeric code 45", "45", "", "", "", "", "", "", "45"},

		// Template defaults (no method set)
		{"default transferencia", "", "", "cnab240_transferencia", "12345678901", "12345678000195", "077", "341", "41"},
		{"default boleto", "", "", "cnab240_boleto", "", "", "", "", "31"},
		{"default pix_conta", "", "", "cnab240_pix_conta", "", "", "", "", "45"},
		{"default fallback", "", "", "", "", "", "", "", "41"},

		// Tributos via template + tax_type
		{"tributos DARF", "", "DARF", "cnab240_tributos", "", "", "", "", "16"},
		{"tributos GPS", "", "GPS", "cnab240_tributos", "", "", "", "", "17"},
		{"tributos DARF_SIMPLES", "", "DARF_SIMPLES", "cnab240_tributos", "", "", "", "", "18"},
		{"tributos GARE_SP", "", "GARE_SP", "cnab240_tributos", "", "", "", "", "22"},
		{"tributos IPVA", "", "IPVA", "cnab240_tributos", "", "", "", "", "25"},
		{"tributos DPVAT", "", "DPVAT", "cnab240_tributos", "", "", "", "", "27"},
		{"tributos FGTS", "", "FGTS", "cnab240_tributos", "", "", "", "", "35"},
		{"tributos empty tax_type", "", "", "cnab240_tributos", "", "", "", "", "16"},
		{"tributos unknown tax_type", "", "UNKNOWN", "cnab240_tributos", "", "", "", "", "16"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment := &types.PaymentData{
				PaymentMethod:     tt.method,
				TaxType:           tt.taxType,
				RecipientDocument: tt.recipientDocument,
				RecipientBank:     tt.recipientBank,
			}
			ctx := &Context{
				CurrentPayment: payment,
				TemplateName:   tt.templateName,
				Company:        types.CompanyData{CNPJ: tt.companyCNPJ, BankCode: tt.companyBankCode},
			}
			result, err := r.Resolve("payment.payment_method", ctx, "")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("nil payment returns default", func(t *testing.T) {
		ctx := &Context{}
		result, err := r.Resolve("payment.payment_method", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "41", result)
	})

	t.Run("default transferencia outro titular", func(t *testing.T) {
		payment := &types.PaymentData{
			RecipientDocument: "12345678901",
			RecipientBank:     "077",
		}
		ctx := &Context{
			CurrentPayment: payment,
			TemplateName:   "cnab240_transferencia",
			Company:        types.CompanyData{CNPJ: "12345678000195", BankCode: "341"},
		}
		result, err := r.Resolve("payment.payment_method", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "41", result)
	})

	t.Run("default transferencia mesmo titular", func(t *testing.T) {
		payment := &types.PaymentData{
			RecipientDocument: "12345678000195",
			RecipientBank:     "077",
		}
		ctx := &Context{
			CurrentPayment: payment,
			TemplateName:   "cnab240_transferencia",
			Company:        types.CompanyData{CNPJ: "12345678000195", BankCode: "341"},
		}
		result, err := r.Resolve("payment.payment_method", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "43", result)
	})

	t.Run("default transferencia mesmo banco", func(t *testing.T) {
		payment := &types.PaymentData{
			RecipientDocument: "12345678000195",
			RecipientBank:     "341",
		}
		ctx := &Context{
			CurrentPayment: payment,
			TemplateName:   "cnab240_transferencia",
			Company:        types.CompanyData{CNPJ: "12345678000195", BankCode: "341"},
		}
		result, err := r.Resolve("payment.payment_method", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "01", result)
	})

	t.Run("method TED forca credito em conta mesmo banco", func(t *testing.T) {
		payment := &types.PaymentData{
			PaymentMethod:     "TED",
			RecipientDocument: "12345678000195",
			RecipientBank:     "341",
		}
		ctx := &Context{
			CurrentPayment: payment,
			TemplateName:   "cnab240_transferencia",
			Company:        types.CompanyData{CNPJ: "12345678000195", BankCode: "341"},
		}
		result, err := r.Resolve("payment.payment_method", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "01", result)
	})

	t.Run("numeric 41 forca credito em conta mesmo banco", func(t *testing.T) {
		payment := &types.PaymentData{
			PaymentMethod:     "41",
			RecipientDocument: "12345678901",
			RecipientBank:     "341",
		}
		ctx := &Context{
			CurrentPayment: payment,
			TemplateName:   "cnab240_transferencia",
			Company:        types.CompanyData{CNPJ: "12345678000195", BankCode: "341"},
		}
		result, err := r.Resolve("payment.payment_method", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "01", result)
	})
}

func TestResolveCamara(t *testing.T) {
	r := New()

	t.Run("mesmo banco", func(t *testing.T) {
		payment := &types.PaymentData{RecipientBank: "341"}
		ctx := &Context{CurrentPayment: payment, Company: types.CompanyData{BankCode: "341"}}
		result, err := r.Resolve("payment.camara", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "000", result)
	})

	t.Run("banco diferente", func(t *testing.T) {
		payment := &types.PaymentData{RecipientBank: "077"}
		ctx := &Context{CurrentPayment: payment, Company: types.CompanyData{BankCode: "341"}}
		result, err := r.Resolve("payment.camara", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "018", result)
	})

	t.Run("nil payment", func(t *testing.T) {
		ctx := &Context{Company: types.CompanyData{BankCode: "341"}}
		result, err := r.Resolve("payment.camara", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "018", result)
	})
}

func TestResolveRecipientDocumentType(t *testing.T) {
	r := New()

	tests := []struct {
		name     string
		document string
		expected string
	}{
		{"CPF 11 digits", "12345678901", "1"},
		{"CNPJ 14 digits", "12345678000195", "2"},
		{"empty document", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment := &types.PaymentData{RecipientDocument: tt.document}
			ctx := &Context{CurrentPayment: payment}
			result, err := r.Resolve("payment.recipient_document_type", ctx, "")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("nil payment", func(t *testing.T) {
		ctx := &Context{}
		result, err := r.Resolve("payment.recipient_document_type", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})
}

func TestResolveBarcodeCurrencyConsistency(t *testing.T) {
	r := New()

	t.Run("with barcode", func(t *testing.T) {
		payment := &types.PaymentData{
			Barcode: "34191790000015007501111222233334445556677777",
		}
		ctx := &Context{CurrentPayment: payment}
		result, err := r.Resolve("payment.barcode_currency", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "9", result)
	})

	t.Run("empty barcode returns empty", func(t *testing.T) {
		payment := &types.PaymentData{Barcode: ""}
		ctx := &Context{CurrentPayment: payment}
		result, err := r.Resolve("payment.barcode_currency", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("nil payment returns empty", func(t *testing.T) {
		ctx := &Context{}
		result, err := r.Resolve("payment.barcode_currency", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})
}

func TestResolveMetadataEdgeCases(t *testing.T) {
	r := New()

	t.Run("empty key from trailing dot", func(t *testing.T) {
		payment := &types.PaymentData{
			Metadata: map[string]interface{}{"": "should not match"},
		}
		ctx := &Context{CurrentPayment: payment}
		result, err := r.Resolve("payment.metadata.", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("nil metadata", func(t *testing.T) {
		payment := &types.PaymentData{}
		ctx := &Context{CurrentPayment: payment}
		result, err := r.Resolve("payment.metadata.key", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("bytes value in metadata", func(t *testing.T) {
		payment := &types.PaymentData{
			Metadata: map[string]interface{}{
				"raw": []byte("hello"),
			},
		}
		ctx := &Context{CurrentPayment: payment}
		result, err := r.Resolve("payment.metadata.raw", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})
}

func TestTaxTypeToPaymentCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"TED", "41"},
		{"DOC", "03"},
		{"CREDITO", "01"},
		{"PIX", "45"},
		{"DARF", "16"},
		{"GPS", "17"},
		{"FGTS", "35"},
		{"UNKNOWN", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, TaxTypeToPaymentCode(tt.input))
		})
	}
}

func TestResolvePixKeyType(t *testing.T) {
	r := New()

	tests := []struct {
		name     string
		keyType  string
		pixKey   string
		expected string
	}{
		// Nomes textuais (case-insensitive)
		{"EMAIL uppercase", "EMAIL", "test@example.com", "02"},
		{"email lowercase", "email", "test@example.com", "02"},
		{"E-MAIL with hyphen", "E-MAIL", "test@example.com", "02"},
		{"TELEFONE", "TELEFONE", "+5511999999999", "01"},
		{"PHONE", "PHONE", "+5511999999999", "01"},
		{"CPF", "CPF", "12345678901", "03"},
		{"CNPJ", "CNPJ", "12345678000195", "03"},
		{"CHAVE_ALEATORIA", "CHAVE_ALEATORIA", "abc123", "04"},
		{"CHAVE ALEATORIA", "CHAVE ALEATORIA", "abc123", "04"},
		{"CHAVE ALEATÓRIA", "CHAVE ALEATÓRIA", "abc123", "04"},
		{"EVP", "EVP", "abc123", "04"},

		// Códigos numéricos diretos (Itaú)
		{"code 01 Telefone", "01", "+5511999999999", "01"},
		{"code 02 Email", "02", "test@example.com", "02"},
		{"code 03 CPF/CNPJ", "03", "12345678901", "03"},
		{"code 04 legado Celular→Telefone", "04", "+5511999999999", "01"},

		// Backwards compatibility (códigos README antigos)
		{"legacy 04 Celular→Telefone", "04", "+5511999999999", "01"},
		{"legacy 05 EVP→04", "05", "abc123", "04"},

		// Inferência por UUID
		{"infer UUID", "", "abc12345-1234-1234-1234-123456789abc", "04"},

		// Vazio e Inferência
		{"empty", "", "", ""},
		{"nil metadata", "", "test@example.com", "02"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment := &types.PaymentData{
				RecipientPixKey: tt.pixKey,
			}
			if tt.keyType != "" || tt.name == "empty" {
				payment.Metadata = map[string]interface{}{
					"key_type": tt.keyType,
				}
			}
			ctx := &Context{CurrentPayment: payment}
			result, err := r.Resolve("payment.pix_key_type", ctx, "")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("nil payment", func(t *testing.T) {
		ctx := &Context{}
		result, err := r.Resolve("payment.pix_key_type", ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})
}

func TestResolvePaymentMethod(t *testing.T) {
	r := New()
	payment := &types.PaymentData{PaymentMethod: "45"}
	ctx := &Context{CurrentPayment: payment}

	result, err := r.Resolve("payment.payment_method", ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "45", result)
}

func TestConvertLinhaDigitavelToBarcode(t *testing.T) {
	tests := []struct {
		name     string
		linha    string
		expected string
		wantErr  bool
	}{
		{
			name:     "conversão válida 47 posições",
			linha:    "34191090080004519091500573140001214260002835578",
			expected: "34192142600028355781090000045190910057314000",
			wantErr:  false,
		},
		{
			name:     "tamanho incorreto 44 posições",
			linha:    "34191790000015007501111222233334445556677777",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "contém caracteres não numéricos",
			linha:    "34191.090080004519091500573140001214260002835578",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertLinhaDigitavelToBarcode(tt.linha)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
			assert.Len(t, result, 44)
		})
	}
}

func TestResolveBarcode_FromLinhaDigitavel47(t *testing.T) {
	r := New()
	// Linha digitável real de boleto
	linha := "34191090080004519091500573140001214260002835578"
	expectedBarcode := "34192142600028355781090000045190910057314000"
	payment := &types.PaymentData{Barcode: linha}
	ctx := &Context{CurrentPayment: payment}

	barcode, err := r.Resolve("payment.barcode", ctx, "")
	require.NoError(t, err)
	assert.Equal(t, expectedBarcode, barcode)

	bank, err := r.Resolve("payment.barcode_bank", ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "341", bank)

	currency, err := r.Resolve("payment.barcode_currency", ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "9", currency)

	dv, err := r.Resolve("payment.barcode_dv", ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "2", dv)

	dueFactor, err := r.Resolve("payment.barcode_due_factor", ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "1426", dueFactor)

	amount, err := r.Resolve("payment.barcode_amount", ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "0002835578", amount)

	freeField, err := r.Resolve("payment.barcode_free_field", ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "1090000045190910057314000", freeField)
}
