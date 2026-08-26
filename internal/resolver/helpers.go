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

// normalizeArrecadacaoBarcode retorna o CÓDIGO DE BARRAS de 44 posições da guia
// de arrecadação — a forma "sem DVs de campo", usada para derivar indicador,
// segmento e valor. NÃO é o que vai no arquivo: o campo CÓDIGO DE BARRAS do
// Segmento O tem 48 posições e recebe a representação numérica, ver
// RepresentacaoNumericaArrecadacao.
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
	// 13=Concessionárias, 19=IPTU/ISS e demais tributos municipais,
	// 91=GNRE e Tributos com Código de Barras.
	case "13", "19", "91":
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
// 10-11) do lote de tributos COM código de barras no Itaú.
//
// Tipo e forma andam em par: o Itaú recusa a combinação que não existe na tabela
// da página 4 do manual SISPAG ("IM - tipo x forma não compatível"). Por isso o
// tipo sai do mesmo segmento da guia que decide a forma, ver
// FormaPagamentoTributoItau:
//
//   - forma 19 (prefeituras) e forma 91 (GNRE e demais): TIPO 22 - TRIBUTOS;
//   - forma 13 (concessionárias): TIPO 20 - FORNECEDORES.
//
// Histórico: até 26/08/2026 o tipo saía do indicador de valor, com 20 para módulo
// 10 e 22 para módulo 11, porque sob o par 22 x 91 as guias de módulo 10 voltavam
// com "RJ IP - DAC do código de barras inválido" (MUNICIPIO DE BOITUVA, remessa de
// 24/08/2026) e sob 20 x 91 eram pagas (remessa de 29/07/2026). A causa daquele IP
// era o campo de código de barras gravado com 44 dígitos em vez da representação
// numérica de 48 — ver RepresentacaoNumericaArrecadacao —, não o tipo do lote.
//
// Sem código de barras de arrecadação (Segmento N, ou guia ausente) fica no 22,
// que é o tipo de serviço de tributos do layout.
func TipoPagamentoTributoItau(barcode string) string {
	if FormaPagamentoTributoItau(barcode) == "13" {
		return "20"
	}
	return "22"
}

// dvCampoArrecadacao calcula o dígito verificador de um campo (11 dígitos) da
// representação numérica de uma guia de arrecadação.
//
// O módulo sai do indicador de valor (3ª posição do código de barras), a mesma
// regra do DAC geral: 6 e 7 = módulo 10; 8 e 9 = módulo 11 (Itaú SISPAG, Anexo B,
// itens 3 e 4).
func dvCampoArrecadacao(campo string, indicador byte) byte {
	if indicador == '8' || indicador == '9' {
		// Módulo 11: pesos 2..9 da direita para a esquerda. DAC = 11 - resto e,
		// quando o resultado cai em 0, 1, 10 ou 11, o Itaú manda usar 1 (Anexo B,
		// item 3.2 — mesma regra do DAC geral).
		soma := 0
		peso := 2
		for i := len(campo) - 1; i >= 0; i-- {
			soma += int(campo[i]-'0') * peso
			peso++
			if peso > 9 {
				peso = 2
			}
		}
		dac := 11 - soma%11
		if dac == 0 || dac == 1 || dac >= 10 {
			return '1'
		}
		return byte('0' + dac)
	}

	// Módulo 10: multiplicadores 2, 1, 2, 1... da direita para a esquerda, somando
	// os algarismos de cada produto. DV = 10 - resto, e 10 vira 0 (Anexo B, item 4).
	soma := 0
	peso := 2
	for i := len(campo) - 1; i >= 0; i-- {
		produto := int(campo[i]-'0') * peso
		if produto > 9 {
			produto -= 9
		}
		soma += produto
		if peso == 2 {
			peso = 1
		} else {
			peso = 2
		}
	}
	dv := 10 - soma%10
	if dv == 10 {
		dv = 0
	}
	return byte('0' + dv)
}

// RepresentacaoNumericaArrecadacao devolve os 48 dígitos que o campo CÓDIGO DE
// BARRAS do Segmento O ocupa (posições 018 a 065 do registro).
//
// O Itaú lê esse campo como os 4 campos da representação numérica da guia, cada
// um com 11 dígitos + DV, ancorados em posições fixas do registro (Anexo B,
// item 2): campo 1 em 018-029, campo 2 em 030-041, campo 3 em 042-053 e campo 4
// em 054-065. Gravar só os 44 dígitos do código de barras deixa 062-065 em
// branco, o que parte o campo 4 no meio e faz o banco recusar o arquivo com
// "caracter inválido na posição 54" (ou "IP - DAC do código de barras inválido",
// quando a crítica vem pelo retorno).
//
// Aceita:
//   - 48 dígitos (representação numérica): devolve como veio — o manual exige que
//     o dado seja alocado "sem qualquer alteração em seu conteúdo";
//   - 44 dígitos (código de barras puro): recompõe os DVs de campo;
//   - qualquer outro tamanho: devolve apenas os dígitos (a validação sinaliza).
func RepresentacaoNumericaArrecadacao(raw string) string {
	digits := onlyDigits(raw)
	if len(digits) == 48 {
		return digits
	}
	if len(digits) != 44 {
		return digits
	}

	indicador := digits[2]
	var b strings.Builder
	b.Grow(48)
	for i := 0; i < 4; i++ {
		campo := digits[i*11 : (i+1)*11]
		b.WriteString(campo)
		b.WriteByte(dvCampoArrecadacao(campo, indicador))
	}
	return b.String()
}

// ArrecadacaoSegmento devolve o segmento da guia (2ª posição do código de barras),
// ou "" quando o código não é de arrecadação.
//
// 1 = Prefeituras (IPTU/ISS e demais tributos municipais), 2 = Saneamento,
// 3 = Energia Elétrica e Gás, 4 = Telecomunicações, 5 = Órgãos Governamentais,
// 6 = Carnês, 7 = Multas de trânsito, 9 = Uso exclusivo do banco.
func ArrecadacaoSegmento(barcode string) string {
	digits := normalizeArrecadacaoBarcode(barcode)
	if len(digits) != 44 || digits[0] != '8' {
		return ""
	}
	return digits[1:2]
}

// FormaPagamentoTributoItau devolve a FORMA DE PAGAMENTO (header de lote, posições
// 12-13) do lote de tributos COM código de barras no Itaú, a partir do segmento
// da guia.
//
// O Itaú valida o par tipo x forma contra a tabela da página 4 do manual SISPAG, e
// escolhe por ele como criticar a guia:
//
//   - segmento 1 (prefeituras): forma 19 - IPTU/ISS/OUTROS TRIBUTOS MUNICIPAIS,
//     que a tabela cruza com o tipo 22 (TRIBUTOS);
//   - segmentos 2, 3 e 4 (saneamento, energia/gás, telecom): forma 13 - PAGAMENTO
//     DE CONCESSIONÁRIAS, que a tabela cruza com o tipo 20 (FORNECEDORES);
//   - demais segmentos (órgãos governamentais, carnês, multas): forma 91 - GNRE E
//     TRIBUTOS COM CÓDIGO DE BARRAS, cruzada com o tipo 22.
//
// Sem guia de arrecadação (Segmento N, ou guia ausente) devolve "", e a forma
// continua saindo do payment_method informado pelo chamador.
func FormaPagamentoTributoItau(barcode string) string {
	switch ArrecadacaoSegmento(barcode) {
	case "1":
		return "19"
	case "2", "3", "4":
		return "13"
	case "5", "6", "7", "9":
		return "91"
	}
	return ""
}
