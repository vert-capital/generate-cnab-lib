package template

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTemplates(t *testing.T) {
	templates, err := Load()
	require.NoError(t, err)
	require.NotNil(t, templates)

	// Verifica se os templates principais foram carregados
	expectedTemplates := []string{
		"341_cnab240_pix_conta",
		"341_cnab240_boleto",
		"341_cnab240_transferencia",
		"341_cnab240_tributos",
		"341_cnab240_pix_conta_retorno",
		"341_cnab240_boleto_retorno",
		"341_cnab240_transferencia_retorno",
		"341_cnab240_tributos_retorno",
		"033_cnab240_pix_conta_retorno",
		"033_cnab240_boleto_retorno",
		"033_cnab240_transferencia_retorno",
		"033_cnab240_tributos_retorno",
		"208_cnab240_pix_conta",
		"208_cnab240_boleto",
		"208_cnab240_transferencia",
		"208_cnab240_tributos",
		"208_cnab240_pix_conta_retorno",
		"208_cnab240_boleto_retorno",
		"208_cnab240_transferencia_retorno",
		"208_cnab240_tributos_retorno",
	}

	for _, key := range expectedTemplates {
		t.Run(key, func(t *testing.T) {
			tmpl, ok := templates[key]
			assert.True(t, ok, "Template %s não encontrado", key)
			if ok {
				assert.NotEmpty(t, tmpl.Name)
				assert.NotEmpty(t, tmpl.BankCode)
				assert.NotEmpty(t, tmpl.Segments)
			}
		})
	}
}

func TestGetTemplate(t *testing.T) {
	tests := []struct {
		name         string
		bankCode     string
		templateName string
		wantError    bool
	}{
		{
			name:         "PIX conta template",
			bankCode:     "341",
			templateName: "cnab240_pix_conta",
			wantError:    false,
		},
		{
			name:         "Boleto template",
			bankCode:     "341",
			templateName: "cnab240_boleto",
			wantError:    false,
		},
		{
			name:         "Transferencia template",
			bankCode:     "341",
			templateName: "cnab240_transferencia",
			wantError:    false,
		},
		{
			name:         "Tributos template",
			bankCode:     "341",
			templateName: "cnab240_tributos",
			wantError:    false,
		},
		{
			name:         "Retorno PIX template",
			bankCode:     "341",
			templateName: "cnab240_pix_conta_retorno",
			wantError:    false,
		},
		{
			name:         "Template inexistente",
			bankCode:     "341",
			templateName: "template_nao_existe",
			wantError:    true,
		},
		{
			name:         "Banco inexistente",
			bankCode:     "999",
			templateName: "cnab240_pix_conta",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := Get(tt.bankCode, tt.templateName)
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, tmpl.Name)
			assert.Equal(t, tt.bankCode, tmpl.BankCode)
			assert.NotEmpty(t, tmpl.Segments)
		})
	}
}

func TestTemplateStructure(t *testing.T) {
	tmpl, err := Get("341", "cnab240_pix_conta")
	require.NoError(t, err)

	// Verifica estrutura básica
	assert.Equal(t, "cnab240_pix_conta", tmpl.Name)
	assert.Equal(t, "341", tmpl.BankCode)
	assert.Equal(t, "240", tmpl.CnabType)
	assert.Equal(t, 240, tmpl.LineLength)

	// Verifica segmentos obrigatórios
	requiredSegments := []string{
		"header_arquivo",
		"header_lote",
		"segmento_a",
		"segmento_b",
		"trailer_lote",
		"trailer_arquivo",
	}

	for _, segName := range requiredSegments {
		t.Run(segName, func(t *testing.T) {
			seg, ok := tmpl.Segments[segName]
			assert.True(t, ok, "Segmento %s não encontrado", segName)
			if ok {
				assert.NotEmpty(t, seg.Fields)
			}
		})
	}
}

func TestTemplateBoletoStructure(t *testing.T) {
	tmpl, err := Get("341", "cnab240_boleto")
	require.NoError(t, err)

	// Boleto usa segmento J em vez de A
	assert.Contains(t, tmpl.Segments, "segmento_j")
	assert.NotContains(t, tmpl.Segments, "segmento_a")
}

func TestTemplateTransferenciaStructure(t *testing.T) {
	tmpl, err := Get("341", "cnab240_transferencia")
	require.NoError(t, err)

	// Verifica campos específicos de transferência
	segA, ok := tmpl.Segments["segmento_a"]
	require.True(t, ok)

	// Deve ter campos para finalidade TED
	assert.Contains(t, segA.Fields, "finalidade_ted")
	assert.Contains(t, segA.Fields, "finalidade_doc")
}

func TestTemplateTributosStructure(t *testing.T) {
	tmpl, err := Get("341", "cnab240_tributos")
	require.NoError(t, err)

	// Tributos usa segmento O
	assert.Contains(t, tmpl.Segments, "segmento_o")

	// Verifica campos do segmento O
	segO, ok := tmpl.Segments["segmento_o"]
	require.True(t, ok)
	assert.Contains(t, segO.Fields, "codigo_barras")
	assert.Contains(t, segO.Fields, "nome_concessionaria")
}

func TestTemplateRetornoStructure(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
	}{
		{
			name:         "Retorno PIX",
			templateName: "cnab240_pix_conta_retorno",
		},
		{
			name:         "Retorno Boleto",
			templateName: "cnab240_boleto_retorno",
		},
		{
			name:         "Retorno Transferencia",
			templateName: "cnab240_transferencia_retorno",
		},
		{
			name:         "Retorno Tributos",
			templateName: "cnab240_tributos_retorno",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := Get("341", tt.templateName)
			require.NoError(t, err)

			// Templates de retorno devem ter header_arquivo
			assert.Contains(t, tmpl.Segments, "header_arquivo")

			// Deve ter ocorrências definidas
			assert.NotNil(t, tmpl.Occurrences)
			assert.NotEmpty(t, tmpl.Occurrences)
		})
	}
}

func TestFieldPositions(t *testing.T) {
	tmpl, err := Get("341", "cnab240_pix_conta")
	require.NoError(t, err)

	headerArquivo, ok := tmpl.Segments["header_arquivo"]
	require.True(t, ok)

	// Verifica campos obrigatórios do header
	requiredFields := map[string]struct {
		start int
		end   int
		dtype string
	}{
		"codigo_banco":     {1, 3, "num"},
		"lote_servico":     {4, 7, "num"},
		"tipo_registro":    {8, 8, "num"},
		"tipo_inscricao":   {18, 18, "num"},
		"numero_inscricao": {19, 32, "num"},
		"nome_empresa":     {73, 102, "alfa"},
	}

	for fieldName, expected := range requiredFields {
		t.Run(fieldName, func(t *testing.T) {
			field, ok := headerArquivo.Fields[fieldName]
			require.True(t, ok, "Campo %s não encontrado", fieldName)

			assert.Equal(t, expected.start, field.Pos[0], "Posição inicial incorreta")
			assert.Equal(t, expected.end, field.Pos[1], "Posição final incorreta")
			assert.Equal(t, expected.dtype, field.Type, "Tipo incorreto")
		})
	}
}

func TestCacheBehavior(t *testing.T) {
	// O primeiro Load() deve carregar os templates
	templates1, err := Load()
	require.NoError(t, err)

	// Chamadas subsequentes devem retornar o mesmo cache
	templates2, err := Load()
	require.NoError(t, err)

	// Deve ser o mesmo ponteiro (cache)
	assert.Equal(t, templates1, templates2)
}

func TestConfigUnmarshalJSON_LegacyFormat(t *testing.T) {
	jsonData := `{
		"name": "test_legacy",
		"bank_code": "341",
		"cnab_type": "240",
		"line_length": 240,
		"segments": {
			"header_arquivo": {
				"codigo_banco": {"pos": [1, 3], "type": "num", "value": "341"},
				"tipo_registro": {"pos": [8, 8], "type": "num", "value": "0"}
			}
		}
	}`

	var config Config
	err := json.Unmarshal([]byte(jsonData), &config)
	require.NoError(t, err)
	assert.Equal(t, "test_legacy", config.Name)
	assert.Equal(t, "341", config.BankCode)
	assert.Equal(t, 240, config.LineLength)

	seg, ok := config.Segments["header_arquivo"]
	require.True(t, ok)
	assert.Len(t, seg.Fields, 2)
	assert.Equal(t, "341", seg.Fields["codigo_banco"].Value)
	assert.Equal(t, [2]int{1, 3}, seg.Fields["codigo_banco"].Pos)
}

func TestConfigUnmarshalJSON_NewFormat(t *testing.T) {
	jsonData := `{
		"name": "test_new",
		"bank_code": "341",
		"cnab_type": "240",
		"line_length": 240,
		"segments": {
			"header_arquivo": {
				"record_type": "0",
				"fields": {
					"codigo_banco": {"pos": [1, 3], "type": "num", "value": "341"},
					"tipo_registro": {"pos": [8, 8], "type": "num", "value": "0"}
				}
			}
		}
	}`

	var config Config
	err := json.Unmarshal([]byte(jsonData), &config)
	require.NoError(t, err)
	assert.Equal(t, "test_new", config.Name)

	seg, ok := config.Segments["header_arquivo"]
	require.True(t, ok)
	assert.Equal(t, "0", seg.RecordType)
	assert.Len(t, seg.Fields, 2)
}

func TestConfigUnmarshalJSON_InvalidJSON(t *testing.T) {
	var config Config
	err := json.Unmarshal([]byte(`{invalid`), &config)
	assert.Error(t, err)
}

func TestConfigUnmarshalJSON_AllTopLevelFields(t *testing.T) {
	jsonData := `{
		"name": "test_full",
		"bank_code": "341",
		"cnab_type": "240",
		"file_type": "REMESSA",
		"version": "1.0",
		"description": "Template de teste",
		"line_length": 240,
		"line_separator": "\r\n",
		"detail_segments": ["A", "B"],
		"ocorrencias": {"00": "Sucesso", "BD": "Erro"},
		"segments": {
			"header_arquivo": {
				"campo1": {"pos": [1, 3], "type": "num", "value": "341"}
			}
		}
	}`

	var config Config
	err := json.Unmarshal([]byte(jsonData), &config)
	require.NoError(t, err)
	assert.Equal(t, "test_full", config.Name)
	assert.Equal(t, "341", config.BankCode)
	assert.Equal(t, "240", config.CnabType)
	assert.Equal(t, "REMESSA", config.FileType)
	assert.Equal(t, "1.0", config.Version)
	assert.Equal(t, "Template de teste", config.Description)
	assert.Equal(t, 240, config.LineLength)
	assert.Equal(t, "\r\n", config.LineSeparator)
	assert.Equal(t, []string{"A", "B"}, config.DetailSegments)
	assert.Equal(t, "Sucesso", config.Occurrences["00"])
	assert.Equal(t, "Erro", config.Occurrences["BD"])
}

func TestSegmentUnmarshalJSON_SortedFields(t *testing.T) {
	jsonData := `{
		"campo_c": {"pos": [20, 30], "type": "alfa"},
		"campo_a": {"pos": [1, 9], "type": "num"},
		"campo_b": {"pos": [10, 19], "type": "alfa"}
	}`

	var seg Segment
	err := json.Unmarshal([]byte(jsonData), &seg)
	require.NoError(t, err)
	assert.Len(t, seg.SortedFields, 3)
	assert.Equal(t, "campo_a", seg.SortedFields[0].Name)
	assert.Equal(t, "campo_b", seg.SortedFields[1].Name)
	assert.Equal(t, "campo_c", seg.SortedFields[2].Name)
}

func TestSegmentUnmarshalJSON_WithRecordType(t *testing.T) {
	jsonData := `{
		"record_type": "3",
		"campo1": {"pos": [1, 10], "type": "num"}
	}`

	var seg Segment
	err := json.Unmarshal([]byte(jsonData), &seg)
	require.NoError(t, err)
	assert.Equal(t, "3", seg.RecordType)
	assert.Len(t, seg.Fields, 1)
	assert.Contains(t, seg.Fields, "campo1")
}

func TestSegmentUnmarshalJSON_SkipsUnderscoreKeys(t *testing.T) {
	jsonData := `{
		"_comment": "isso deve ser ignorado",
		"campo1": {"pos": [1, 10], "type": "num"}
	}`

	var seg Segment
	err := json.Unmarshal([]byte(jsonData), &seg)
	require.NoError(t, err)
	assert.Len(t, seg.Fields, 1)
	assert.NotContains(t, seg.Fields, "_comment")
}

func TestValidatePositions_Valid(t *testing.T) {
	tmpl := Config{
		LineLength: 240,
		Segments: map[string]Segment{
			"header": {
				Fields: map[string]Field{
					"campo1": {Pos: [2]int{1, 3}},
					"campo2": {Pos: [2]int{4, 240}},
				},
			},
		},
	}
	err := validatePositions(tmpl)
	assert.NoError(t, err)
}

func TestValidatePositions_InvalidStart(t *testing.T) {
	tmpl := Config{
		LineLength: 240,
		Segments: map[string]Segment{
			"header": {
				Fields: map[string]Field{
					"campo_ruim": {Pos: [2]int{0, 3}},
				},
			},
		},
	}
	err := validatePositions(tmpl)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "posição inicial")
}

func TestValidatePositions_EndBeforeStart(t *testing.T) {
	tmpl := Config{
		LineLength: 240,
		Segments: map[string]Segment{
			"header": {
				Fields: map[string]Field{
					"campo_ruim": {Pos: [2]int{10, 5}},
				},
			},
		},
	}
	err := validatePositions(tmpl)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "posição final")
}

func TestValidatePositions_ExceedsLineLength(t *testing.T) {
	tmpl := Config{
		LineLength: 240,
		Segments: map[string]Segment{
			"header": {
				Fields: map[string]Field{
					"campo_ruim": {Pos: [2]int{1, 241}},
				},
			},
		},
	}
	err := validatePositions(tmpl)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "excede tamanho")
}

func TestValidatePositions_DefaultLineLength(t *testing.T) {
	tmpl := Config{
		LineLength: 0, // deve usar 240 como padrão
		Segments: map[string]Segment{
			"header": {
				Fields: map[string]Field{
					"campo1": {Pos: [2]int{1, 240}},
				},
			},
		},
	}
	err := validatePositions(tmpl)
	assert.NoError(t, err)
}

func TestSegmentUnmarshalJSON_RecordTypeMalformed(t *testing.T) {
	// record_type como número em vez de string deve retornar erro
	jsonData := `{
		"record_type": 123,
		"codigo_banco": {"pos": [1, 3], "type": "num", "value": "341"}
	}`

	var seg Segment
	err := json.Unmarshal([]byte(jsonData), &seg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "record_type")
}
