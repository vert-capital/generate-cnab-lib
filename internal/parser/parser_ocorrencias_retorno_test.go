package parser

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vert-capital/generate-cnab-lib/internal/template"
)

// posicoesOcorrenciasCNAB240 sao as posicoes (1-based) do campo "Códigos das Ocorrências"
// em TODO registro CNAB240 — header de lote, qualquer segmento de detalhe e trailer de
// lote. Nao ha excecao por banco nem por familia de pagamento.
var posicoesOcorrenciasCNAB240 = [2]int{231, 240}

// TestTodoTemplateDeRetornoLeOcorrenciasNasPosicoesPadrao trava a invariante que faltava
// quando o segmento J do boleto do Itau declarava "ocorrencias" em [225, 232].
//
// O efeito daquele deslocamento nao era ler o campo errado e parar por ai: as posicoes
// 225-230 carregam o fim do numero do documento do banco, entao o campo cru chegava ao
// consumidor como "000019BD" em vez de "BD". Quem classifica o retorno fatia esse campo em
// codigos de dois digitos (ate cinco, por norma), lia "00" — PAGAMENTO EFETUADO — e dava o
// pagamento por liquidado. Um retorno preliminar ("BD", agendado) e ate um recusado
// ("RJAL") viravam PAID.
//
// Por isso a verificacao e sobre TODOS os templates de retorno, e nao so sobre o segmento
// consertado: o erro e invisivel no arquivo (o campo tem conteudo plausivel) e so aparece
// como pagamento liquidado que nunca foi pago.
func TestTodoTemplateDeRetornoLeOcorrenciasNasPosicoesPadrao(t *testing.T) {
	templates, err := template.Load()
	require.NoError(t, err)

	verificados := 0
	for chave, tmpl := range templates {
		if tmpl.FileType != "RETORNO" {
			continue
		}

		for segKey, seg := range tmpl.Segments {
			campo, ok := seg.Fields["ocorrencias"]
			if !ok {
				continue
			}
			verificados++

			t.Run(fmt.Sprintf("%s/%s", chave, segKey), func(t *testing.T) {
				assert.Equal(t, posicoesOcorrenciasCNAB240, [2]int{campo.Pos[0], campo.Pos[1]},
					"campo 'ocorrencias' fora das posições 231-240: o campo cru sai contaminado "+
						"com bytes do campo vizinho e o classificador lê códigos que o banco não enviou")
			})
		}
	}

	require.NotZero(t, verificados, "nenhum campo de ocorrências foi verificado")
}

// TestParseSegmentoJRetornoItauAgendado usa o retorno REAL que expos o defeito: remessa de
// boleto transmitida pela Accestage (conta 15584-5, 28/08/2026), respondida com "BD"
// (pagamento agendado) no primeiro registro e "RJAL" (registro rejeitado / codigo do banco
// favorecido invalido) no segundo. Antes da correcao os dois voltavam com o campo
// contaminado e eram lidos como pagamento efetuado.
func TestParseSegmentoJRetornoItauAgendado(t *testing.T) {
	// Linhas reais do arquivo, preservadas byte a byte ate a posicao 240 (o padding a
	// direita e reconstruido por padCNAB240 para o teste nao depender de espaco no fim da
	// linha do fonte).
	lines := []string{
		padCNAB240("34100000      080225005683000109                    00910 000000015584 5VERT COMPANHIA SECURITIZADORA BANCO ITAU                              22808202612001200000000000000"),
		padCNAB240("34100011C2030030 225005683000109                    00910 000000015584 5VERT COMPANHIA SECURITIZADORA"),
		padCNAB240("3410001300001J00034191156200000126521570004021570546052713000CAMILOTTI, CASTELLANI, HADDAD,280820260000000000126520000000000000000000000000000002808202600000000001265200000000000000075109                         000678131094000019BD"),
		padCNAB240("3410001300002J000522025005683000109VERT COMPANHIA SECURITIZADORA           2018182187000157CAMILOTTI, CASTELLANI, HADDAD,          0000000000000000                    000000000000000                              000678131094000027BD"),
		padCNAB240("3410001300003B   218182187000157"),
		padCNAB240("3410001300004J00023799155400001583630525090000344641800218240CONDOMINIO EDIFICIO SORAYA    280820260000000001583630000000000000000000000000000002808202600000000015836300000000000000075272                         000678131094000035RJAL"),
		padCNAB240("3410001300005J000522025005683000109VERT COMPANHIA SECURITIZADORA           2060008711000134CONDOMINIO EDIFICIO SORAYA              0000000000000000                    000000000000000                              000678131094000043RJAL"),
		padCNAB240("3410001300006B   260008711000134"),
		padCNAB240("34100015         000008000000000000171015000000000000000000"),
		padCNAB240("34199999         000001000009"),
	}

	result, err := Parse(context.Background(), joinLines(lines), "341", "cnab240_boleto_retorno")
	require.NoError(t, err)

	// Os registros J-52 (linhas 4 e 7) complementam o boleto e nao sao pagamento proprio.
	require.Len(t, result.Records, 2)

	assert.Equal(t, "75109", result.Records[0].YourNumber)
	assert.Equal(t, "BD", result.Records[0].OccurrenceCode)
	assert.Equal(t, "Pagamento Agendado", result.Records[0].OccurrenceDescription)

	assert.Equal(t, "75272", result.Records[1].YourNumber)
	assert.Equal(t, "RJAL", result.Records[1].OccurrenceCode)
}

// padCNAB240 completa a linha com espacos ate as 240 posicoes do layout.
func padCNAB240(prefix string) string {
	if len(prefix) >= 240 {
		return prefix[:240]
	}
	padded := make([]byte, 240)
	copy(padded, prefix)
	for i := len(prefix); i < 240; i++ {
		padded[i] = ' '
	}
	return string(padded)
}
