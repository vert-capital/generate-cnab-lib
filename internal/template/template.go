package template

import (
	"encoding/json"
	"fmt"
	"sort"
)

// InputValidation representa uma regra de validação declarativa associada a um source de entrada.
type InputValidation struct {
	Source      string `json:"source"`
	Required    bool   `json:"required,omitempty"`
	MinLength   int    `json:"min_length,omitempty"`
	MaxLength   int    `json:"max_length,omitempty"`
	ExactLength int    `json:"exact_length,omitempty"`
	NumericOnly bool   `json:"numeric_only,omitempty"`
}

// Config representa a configuração de um template CNAB.
type Config struct {
	Name             string             `json:"name"`
	BankCode         string             `json:"bank_code"`
	CnabType         string             `json:"cnab_type"`
	FileType         string             `json:"file_type"`
	Version          string             `json:"version"`
	Description      string             `json:"description"`
	LineLength       int                `json:"line_length"`
	LineSeparator    string             `json:"line_separator,omitempty"`
	DetailSegments   []string           `json:"detail_segments"`
	Segments         map[string]Segment `json:"segments"`
	Occurrences      map[string]string  `json:"ocorrencias,omitempty"`
	InputValidations []InputValidation  `json:"input_validations,omitempty"`
}

// FieldEntry representa um campo nomeado com sua configuração, pré-ordenado por posição.
type FieldEntry struct {
	Name   string
	Config Field
}

// Segment representa um segmento no template.
type Segment struct {
	RecordType   string           `json:"record_type,omitempty"`
	Fields       map[string]Field `json:"-"`
	SortedFields []FieldEntry     `json:"-"`
}

// Field representa um campo em um segmento.
type Field struct {
	Pos         [2]int `json:"pos"`
	Type        string `json:"type"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
	Format      string `json:"format,omitempty"`
	Required    bool   `json:"required,omitempty"`
	MinLength   int    `json:"min_length,omitempty"`
	MaxLength   int    `json:"max_length,omitempty"`
	ExactLength int    `json:"exact_length,omitempty"`
	NumericOnly bool   `json:"numeric_only,omitempty"`
}

// UnmarshalJSON implementa unmarshaling para Config.
// Os campos escalares são tratados pelo decoder padrão via alias.
// Os segmentos são tratados pelo Segment.UnmarshalJSON (aceita campos diretos ou "fields").
func (c *Config) UnmarshalJSON(data []byte) error {
	type Alias Config
	return json.Unmarshal(data, &struct{ *Alias }{Alias: (*Alias)(c)})
}

// UnmarshalJSON implementa unmarshaling flexível para Segment.
// Aceita formato novo (com "fields") e legado (campos diretos).
func (s *Segment) UnmarshalJSON(data []byte) error {
	var newFormat struct {
		RecordType string           `json:"record_type"`
		Fields     map[string]Field `json:"fields"`
	}

	if err := json.Unmarshal(data, &newFormat); err == nil && len(newFormat.Fields) > 0 {
		s.RecordType = newFormat.RecordType
		s.Fields = newFormat.Fields
		s.buildSortedFields()
		return nil
	}

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	s.Fields = make(map[string]Field)

	for key, rawValue := range rawMap {
		if len(key) > 0 && key[0] == '_' {
			continue
		}
		if key == "record_type" {
			var rt string
			if err := json.Unmarshal(rawValue, &rt); err != nil {
				return fmt.Errorf("campo 'record_type': %w", err)
			}
			s.RecordType = rt
			continue
		}

		var field Field
		if err := json.Unmarshal(rawValue, &field); err != nil {
			return fmt.Errorf("campo '%s': %w", key, err)
		}
		s.Fields[key] = field
	}

	s.buildSortedFields()
	return nil
}

// buildSortedFields constrói a lista de campos pré-ordenada por posição.
func (s *Segment) buildSortedFields() {
	s.SortedFields = make([]FieldEntry, 0, len(s.Fields))
	for name, config := range s.Fields {
		s.SortedFields = append(s.SortedFields, FieldEntry{Name: name, Config: config})
	}
	sort.Slice(s.SortedFields, func(i, j int) bool {
		return s.SortedFields[i].Config.Pos[0] < s.SortedFields[j].Config.Pos[0]
	})
}
