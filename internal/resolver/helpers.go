package resolver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vert-capital/generate-cnab-lib/types"
)

// interfaceToString converte qualquer valor de interface para string.
func interfaceToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		if val {
			return "1"
		}
		return "0"
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// normalizeBarcode retorna o código de barras normalizado (44 posições).
// Se o barcode tiver 47 posições (linha digitável), converte para código de barras.
// Remove automaticamente pontos, espaços e hífens da entrada.
func normalizeBarcode(p *types.PaymentData) string {
	if p == nil {
		return ""
	}
	// Remove pontos, espaços e hífens (máscaras comuns de linha digitável)
	cleaned := strings.NewReplacer(".", "", " ", "", "-", "").Replace(p.Barcode)
	if len(cleaned) == 47 {
		if converted, err := ConvertLinhaDigitavelToBarcode(cleaned); err == nil {
			return converted
		}
	}
	return cleaned
}

// ConvertLinhaDigitavelToBarcode converte uma linha digitável de 47 posições
// em um código de barras de 44 posições (boleto de cobrança).
func ConvertLinhaDigitavelToBarcode(linha string) (string, error) {
	if len(linha) != 47 {
		return "", fmt.Errorf("linha digitável deve ter 47 posições, recebido %d", len(linha))
	}
	for _, r := range linha {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("linha digitável deve conter apenas números")
		}
	}
	// Estrutura da linha digitável (47):
	// 1-3: banco | 4: moeda | 5-9: campo livre pt1 | 10: DV1
	// 11-20: campo livre pt2 | 21: DV2
	// 22-31: campo livre pt3 | 32: DV3
	// 33: DV geral | 34-37: fator vencimento | 38-47: valor
	//
	// Código de barras (44):
	// 1-3: banco | 4: moeda | 5: DV geral | 6-9: fator | 10-19: valor | 20-44: campo livre
	barcode := linha[0:4] + linha[32:33] + linha[33:47] + linha[4:9] + linha[10:20] + linha[21:31]
	return barcode, nil
}

// normalizeArrecadacaoBarcode retorna o CÓDIGO DE BARRAS de 44 posições que o
// Segmento O do Itaú espera para tributos/concessionárias (Itaú SISPAG, Anexo B).
// O banco valida o DAC e o valor a partir deste código; enviar a representação
// numérica de 48 dígitos (com os DVs de bloco) corrompe a leitura.
//
// Aceita:
//   - 44 dígitos (código de barras): retorna como está;
//   - 48 dígitos (representação numérica / linha digitável): remove os 4 dígitos
//     verificadores de bloco (posições 12, 24, 36 e 48), reconstruindo os 44;
//   - qualquer outro tamanho: retorna apenas os dígitos (a validação sinaliza).
func normalizeArrecadacaoBarcode(raw string) string {
	digits := onlyDigits(raw)
	switch len(digits) {
	case 44:
		return digits
	case 48:
		return digits[0:11] + digits[12:23] + digits[24:35] + digits[36:47]
	default:
		return digits
	}
}

// onlyDigits remove qualquer caractere não numérico (pontos, espaços, hífens).
func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// TaxTypeToPaymentCode converte tipo de tributo/método textual para código de forma de pagamento.
func TaxTypeToPaymentCode(code string) string {
	switch code {
	case "TED":
		return "41"
	case "DOC":
		return "03"
	case "CREDITO", "CREDITO_EM_CONTA":
		return "01"
	case "BOLETO", "BOLETO_ITAU":
		return "30"
	case "BOLETO_OUTROS", "BOLETO_OUTROS_BANCOS":
		return "31"
	case "PIX":
		return "45"
	case "DARF":
		return "16"
	case "GPS":
		return "17"
	case "DARF_SIMPLES":
		return "18"
	case "GARE_SP", "GARE_SP_ICMS":
		return "22"
	case "IPVA":
		return "25"
	case "LICENCIAMENTO":
		return "26"
	case "DPVAT":
		return "27"
	case "FGTS":
		return "35"
	case "DARE", "DARE_SP", "GNRE", "CONCESSIONARIA", "TRIBUTO_CODIGO_BARRAS":
		// Tributos pagos via código de barras (DARE-SP, GNRE, etc.):
		// forma 91 (GNRE e Tributos com Código de Barras) + Segmento O.
		return "91"
	}
	return ""
}

// IsBarcodeTributo indica se a forma de pagamento de tributo é liquidada via
// código de barras (Segmento O). Nestes casos o campo barcode é obrigatório,
// pois valor, código de receita e datas são derivados do próprio código.
func IsBarcodeTributo(code string) bool {
	switch code {
	case "13", "91": // 13=Concessionárias, 91=GNRE e Tributos com Código de Barras
		return true
	}
	return false
}

// ArrecadacaoIndicador devolve o indicador de valor (posição 3) do código de
// barras de guia de arrecadação, ou "" quando o código não é de arrecadação
// (produto != 8) ou não tem 44/48 dígitos.
//
// O indicador diz por qual regra o DAC geral (posição 4) foi calculado:
// 6 e 7 = módulo 10; 8 e 9 = módulo 11.
func ArrecadacaoIndicador(barcode string) string {
	digits := normalizeArrecadacaoBarcode(barcode)
	if len(digits) != 44 || digits[0] != '8' {
		return ""
	}
	switch digits[2] {
	case '6', '7', '8', '9':
		return digits[2:3]
	}
	return ""
}

// TipoPagamentoTributoItau devolve o TIPO DE PAGAMENTO (header de lote, posições
// 10-11) do lote de tributos COM código de barras no Itaú, a partir do indicador
// da guia.
//
// O Itaú deriva do par tipo+forma como validar o código de barras, e as duas
// famílias de guia não passam pelo mesmo par (todos os casos abaixo são reais,
// conta 0910/15584-5, forma de pagamento 91):
//
//   - indicador 6/7 (módulo 10 — prefeitura, concessionária): TIPO 20.
//     Sob o tipo 22 a guia volta recusada com "RJ IP - DAC do código de barras
//     inválido", mesmo com o DAC correto em módulo 10 (MUNICIPIO DE BOITUVA,
//     remessa de 24/08/2026). Sob o tipo 20 as mesmas guias de módulo 10 são
//     pagas (remessa de 29/07/2026: Prefeitura do Rio, Juatuba e Boituva).
//     Trocar a forma para 13 mantendo o tipo 22 não resolve: o lote inteiro é
//     recusado com "IM - tipo x forma não compatível" (remessa de 25/08/2026).
//   - indicador 8/9 (módulo 11 — GNRE e demais tributos com código de barras):
//     TIPO 22, comprovado em produção com ocorrência 00.
//
// Sem código de barras de arrecadação (Segmento N, ou guia ausente) fica no 22,
// que é o tipo de serviço de tributos do layout.
func TipoPagamentoTributoItau(barcode string) string {
	switch ArrecadacaoIndicador(barcode) {
	case "6", "7":
		return "20"
	}
	return "22"
}
