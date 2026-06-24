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
	case "DPVAT":
		return "27"
	case "FGTS":
		return "35"
	}
	return ""
}
