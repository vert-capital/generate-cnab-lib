// Package parser contém a lógica de parse de arquivos de retorno CNAB.
package parser

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vert-capital/generate-cnab-lib/internal/template"
	"github.com/vert-capital/generate-cnab-lib/types"
)

// cnabPrimaryLetters são as letras de segmento que iniciam um novo registro de detalhe
// conforme a norma CNAB 240. Letras não listadas aqui complementam o registro atual.
var cnabPrimaryLetters = map[string]bool{
	"A": true, "J": true, "O": true, "N": true,
	"P": true, "Q": true, "Y": true,
}

// AuthenticationFieldNames são os nomes que os templates de retorno dão ao campo de
// autenticação bancária do segmento Z. O Bradesco chama de "autenticacao_bancaria"
// (G044); os demais bancos chamam de "autenticacao". Toda variação de nome de banco
// se resolve aqui — quem consome o retorno lê ReturnRecord.Authentication e não
// precisa saber o nome do campo no arquivo.
var AuthenticationFieldNames = []string{"autenticacao", "autenticacao_bancaria", "autenticacao_eletronica"}

// canonicalStringFields mapeia nomes de campo candidatos do template para o campo
// correspondente em ReturnRecord. O primeiro candidato não-vazio vence.
// - PaymentDate recebe data_efetiva (data real) ou data_pagamento quando não há data real.
// - ScheduledDate recebe data_pagamento (data prevista).
var canonicalStringFields = []struct {
	candidates []string
	set        func(*types.ReturnRecord, string)
}{
	{[]string{"seu_numero", "referencia", "numero_documento", "txid"}, func(r *types.ReturnRecord, v string) { r.YourNumber = v }},
	{[]string{"nosso_numero", "end_to_end_id"}, func(r *types.ReturnRecord, v string) { r.OurNumber = v }},
	{[]string{"ocorrencias"}, func(r *types.ReturnRecord, v string) { r.OccurrenceCode = v }},
	{[]string{"data_efetiva", "data_pagamento", "data_hora_pagamento"}, func(r *types.ReturnRecord, v string) {
		if len(v) == 14 {
			v = v[:8]
		}
		r.PaymentDate = v
	}},
	{[]string{"data_pagamento"}, func(r *types.ReturnRecord, v string) { r.ScheduledDate = v }},
	{[]string{"nome_favorecido", "nome_concessionaria"}, func(r *types.ReturnRecord, v string) { r.RecipientName = v }},
	{[]string{"numero_inscricao", "numero_inscricao_favorecido"}, func(r *types.ReturnRecord, v string) { r.RecipientDocument = v }},
	{AuthenticationFieldNames, func(r *types.ReturnRecord, v string) { r.Authentication = v }},
}

// canonicalCurrencyFields mapeia campos monetários com as mesmas regras.
var canonicalCurrencyFields = []struct {
	candidates []string
	set        func(*types.ReturnRecord, float64)
}{
	{[]string{"valor_efetivo", "valor_pago", "valor_pagamento"}, func(r *types.ReturnRecord, v float64) { r.PaidAmount = v }},
	{[]string{"valor_titulo", "valor_a_pagar", "valor_pagamento"}, func(r *types.ReturnRecord, v float64) { r.OriginalAmount = v }},
}

// isSegmentoJ52 identifica o registro opcional J-52 do boleto: mesma letra 'J' do
// segmento principal, distinguido pelo identificador "52" nas posicoes 18-19.
func isSegmentoJ52(line string) bool {
	return len(line) >= 19 && line[13:14] == "J" && line[17:19] == "52"
}

// buildSegmentIndex varre todos os segmentos do template e constrói um mapa
// letra → chave-do-segmento, usando o campo "segmento" em pos [14,14].
// Se esse campo não existir (comum em templates de remessa), a letra é derivada
// do nome da chave: "segmento_b" → "B", "segmento_z_retorno" → "Z", "segmento_n1_gps" → "N".
// Quando múltiplos segmentos compartilham a mesma letra, a chave lexicograficamente
// menor é usada para garantir resultado determinístico.
func buildSegmentIndex(tmpl template.Config) map[string]string {
	keys := make([]string, 0, len(tmpl.Segments))
	for k := range tmpl.Segments {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	index := make(map[string]string)
	for _, segKey := range keys {
		seg := tmpl.Segments[segKey]

		// Apenas segmentos de detalhe (tipo_registro = "3")
		tr, ok := seg.Fields["tipo_registro"]
		if !ok || tr.Value != "3" {
			continue
		}

		// Tenta obter a letra do campo "segmento" no template
		letter := ""
		if sf, ok := seg.Fields["segmento"]; ok && sf.Value != "" {
			letter = sf.Value
		} else if strings.HasPrefix(segKey, "segmento_") {
			// Fallback: usa o primeiro caractere do sufixo da chave.
			// Ex: "segmento_b" → "B", "segmento_z_retorno" → "Z", "segmento_n1_gps" → "N"
			suffix := strings.TrimPrefix(segKey, "segmento_")
			if len(suffix) > 0 {
				letter = strings.ToUpper(string(suffix[0]))
			}
		}

		if letter != "" {
			if _, exists := index[letter]; !exists {
				index[letter] = segKey
			}
		}
	}
	return index
}

// mergeFields acumula os campos de vários segmentos secundários no mesmo mapa.
// Chaves comuns a todos os segmentos (codigo_banco, segmento, ...) são sobrescritas
// pelo segmento mais recente, mas um valor vazio nunca apaga um valor já preenchido —
// é isso que impede um segmento opcional depois do Z de zerar a autenticação.
func mergeFields(dst *map[string]string, fields map[string]string) {
	if *dst == nil {
		*dst = make(map[string]string, len(fields))
	}
	for k, v := range fields {
		if v == "" {
			if _, exists := (*dst)[k]; exists {
				continue
			}
		}
		(*dst)[k] = v
	}
}

// applyCanonicalFields popula os campos tipados do ReturnRecord a partir do mapa
// de campos brutos extraídos da linha, usando as tabelas canônicas string/moeda.
func applyCanonicalFields(record *types.ReturnRecord, fields map[string]string, tmpl template.Config) {
	for _, m := range canonicalStringFields {
		for _, candidate := range m.candidates {
			if val := fields[candidate]; val != "" {
				m.set(record, val)
				break
			}
		}
	}
	for _, m := range canonicalCurrencyFields {
		for _, candidate := range m.candidates {
			if val := fields[candidate]; val != "" {
				m.set(record, parseCurrency(val))
				break
			}
		}
	}
	if tmpl.Occurrences != nil {
		if desc, ok := tmpl.Occurrences[record.OccurrenceCode]; ok {
			record.OccurrenceDescription = desc
		}
	}
}

// Parse parseia um arquivo de retorno CNAB e retorna os dados estruturados.
func Parse(ctx context.Context, content string, bankCode, templateName string) (*types.ReturnFile, error) {
	tmpl, err := template.Get(bankCode, templateName)
	if err != nil {
		return nil, fmt.Errorf("template não encontrado: %w", err)
	}

	segmentIndex := buildSegmentIndex(tmpl)

	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("arquivo vazio")
	}

	returnFile := &types.ReturnFile{
		BankCode:    bankCode,
		Records:     []types.ReturnRecord{},
		Occurrences: tmpl.Occurrences,
	}

	recordIndex := -1

	for _, line := range lines {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")
		if len(line) < 8 {
			continue
		}

		switch line[7:8] {
		case "0": // Header arquivo
			returnFile = parseHeaderArquivo(line, tmpl, returnFile)

		case "1": // Header lote

		case "3": // Detalhe
			if len(line) < 14 {
				continue
			}
			segmento := line[13:14]
			if isSegmentoJ52(line) {
				// Registro opcional do boleto (dados do sacador/sacado). Repete a letra
				// 'J' do segmento principal, mas nao e um pagamento: tratado como
				// registro proprio, ele empurrava o segmento Z seguinte (autenticacao
				// bancaria) para um registro fantasma e o pagamento real ficava sem
				// autenticacao. Nada no retorno depende do conteudo dele.
				continue
			}
			segKey, ok := segmentIndex[segmento]
			if !ok {
				continue
			}
			fields := extractFields(line, tmpl.Segments[segKey])

			if cnabPrimaryLetters[segmento] {
				record := types.ReturnRecord{SegmentType: segmento, PrimarySegment: fields}
				applyCanonicalFields(&record, fields, tmpl)
				returnFile.Records = append(returnFile.Records, record)
				recordIndex = len(returnFile.Records) - 1
			} else if recordIndex >= 0 {
				// Mescla: um registro pode ter vários segmentos secundários (B, C, Z...).
				// Sobrescrever fazia o último vencer, e a autenticação do segmento Z só
				// sobrevivia porque hoje o Z é a última linha do registro.
				mergeFields(&returnFile.Records[recordIndex].SecondarySegment, fields)
				applyCanonicalFields(&returnFile.Records[recordIndex], fields, tmpl)
			}

		case "5": // Trailer lote
		case "9": // Trailer arquivo
		}
	}

	returnFile.TotalRecords = len(returnFile.Records)
	return returnFile, nil
}

func parseHeaderArquivo(line string, tmpl template.Config, rf *types.ReturnFile) *types.ReturnFile {
	if seg, ok := tmpl.Segments["header_arquivo"]; ok {
		fields := extractFields(line, seg)
		rf.CompanyCNPJ = getFieldValue(fields, "numero_inscricao")
		rf.CompanyName = getFieldValue(fields, "nome_empresa")
		rf.GenerationDate = getFieldValue(fields, "data_geracao")
	}
	return rf
}

func extractFields(line string, seg template.Segment) map[string]string {
	fields := make(map[string]string)
	for fieldName, fieldConfig := range seg.Fields {
		start := fieldConfig.Pos[0]
		end := fieldConfig.Pos[1]
		if start < 1 || end < start || end > len(line) {
			continue
		}
		fields[fieldName] = strings.TrimSpace(line[start-1 : end])
	}
	return fields
}

func getFieldValue(fields map[string]string, key string) string {
	return fields[key]
}

func parseCurrency(valStr string) float64 {
	valStr = strings.TrimSpace(valStr)
	if valStr == "" {
		return 0
	}
	if len(valStr) < 3 {
		valStr = strings.Repeat("0", 3-len(valStr)) + valStr
	}
	intPart := valStr[:len(valStr)-2]
	decPart := valStr[len(valStr)-2:]
	if intPart == "" {
		intPart = "0"
	}
	result, err := strconv.ParseFloat(intPart+"."+decPart, 64)
	if err != nil {
		return 0
	}
	return result
}
