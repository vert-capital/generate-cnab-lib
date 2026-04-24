package template

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed itau/*.json
var templatesFS embed.FS

const defaultLineLength = 240

var loadTemplates = sync.OnceValues(func() (map[string]Config, error) {
	files, err := templatesFS.ReadDir("itau")
	if err != nil {
		return nil, err
	}

	result := make(map[string]Config)
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := templatesFS.ReadFile("itau/" + f.Name())
		if err != nil {
			return nil, fmt.Errorf("erro ao ler %s: %w", f.Name(), err)
		}
		var tmpl Config
		if err := json.Unmarshal(data, &tmpl); err != nil {
			return nil, fmt.Errorf("erro ao fazer parse de %s: %w", f.Name(), err)
		}
		if err := validatePositions(tmpl); err != nil {
			return nil, fmt.Errorf("template %s inválido: %w", f.Name(), err)
		}
		if tmpl.LineLength == 0 {
			tmpl.LineLength = defaultLineLength
		}
		key := fmt.Sprintf("%s_%s", tmpl.BankCode, tmpl.Name)
		result[key] = tmpl
	}

	return result, nil
})

// validatePositions valida que todas as posições de campos estão dentro dos limites.
func validatePositions(tmpl Config) error {
	lineLength := tmpl.LineLength
	if lineLength == 0 {
		lineLength = defaultLineLength
	}
	for segName, seg := range tmpl.Segments {
		for fieldName, field := range seg.Fields {
			if field.Pos[0] < 1 {
				return fmt.Errorf("segmento '%s', campo '%s': posição inicial %d deve ser >= 1", segName, fieldName, field.Pos[0])
			}
			if field.Pos[1] < field.Pos[0] {
				return fmt.Errorf("segmento '%s', campo '%s': posição final %d menor que inicial %d", segName, fieldName, field.Pos[1], field.Pos[0])
			}
			if field.Pos[1] > lineLength {
				return fmt.Errorf("segmento '%s', campo '%s': posição final %d excede tamanho da linha %d", segName, fieldName, field.Pos[1], lineLength)
			}
		}
	}
	return nil
}

var (
	registeredTemplates   = make(map[string]Config)
	registeredTemplatesMu sync.RWMutex
)

// RegisterTemplate registra um template externo para uso na geração.
// Permite suporte a bancos não incluídos nos templates embarcados.
func RegisterTemplate(tmpl Config) error {
	if err := validatePositions(tmpl); err != nil {
		return fmt.Errorf("template inválido: %w", err)
	}
	if tmpl.BankCode == "" || tmpl.Name == "" {
		return fmt.Errorf("template deve ter bank_code e name definidos")
	}
	if tmpl.LineLength == 0 {
		tmpl.LineLength = defaultLineLength
	}
	key := fmt.Sprintf("%s_%s", tmpl.BankCode, tmpl.Name)
	registeredTemplatesMu.Lock()
	defer registeredTemplatesMu.Unlock()
	registeredTemplates[key] = tmpl
	return nil
}

// Load carrega todos os templates embutidos.
func Load() (map[string]Config, error) {
	return loadTemplates()
}

// Get retorna um template específico pelo banco e nome.
// Verifica templates registrados externamente antes dos embarcados.
func Get(bankCode, templateName string) (Config, error) {
	key := fmt.Sprintf("%s_%s", bankCode, templateName)

	// Verifica templates registrados externamente
	registeredTemplatesMu.RLock()
	if tmpl, ok := registeredTemplates[key]; ok {
		registeredTemplatesMu.RUnlock()
		return tmpl, nil
	}
	registeredTemplatesMu.RUnlock()

	// Fallback para templates embarcados
	templates, err := Load()
	if err != nil {
		return Config{}, err
	}

	tmpl, ok := templates[key]
	if !ok {
		return Config{}, fmt.Errorf("template não encontrado: %s", key)
	}
	return tmpl, nil
}
