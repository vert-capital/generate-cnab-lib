package formatter

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// NormalizeASCII remove acentos e caracteres não-ASCII, mantendo apenas caracteres
// imprimíveis ASCII. Necessário porque arquivos CNAB são posicionais e cada
// posição corresponde a exatamente 1 byte.
func NormalizeASCII(s string) string {
	// Decompõe caracteres acentuados em base + combinação (NFD).
	// Ex.: "ê" → "e" + combining circumflex accent.
	decomposed := norm.NFD.String(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range decomposed {
		// Descarta marcas combinantes (acentos).
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		// Mantém apenas ASCII imprimível e espaço.
		if r >= 0x20 && r <= 0x7E {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// FormatValue formata um valor de acordo com o tipo e tamanho.
// Campos numéricos retornam erro se excederem o tamanho.
// Campos alfanuméricos são truncados (comportamento padrão CNAB posicional).
func FormatValue(value string, length int, fieldType string) (string, error) {
	if fieldType == "num" {
		if len(value) > length {
			return "", fmt.Errorf("valor '%s' (%d caracteres) excede o tamanho do campo (%d)", value, len(value), length)
		}
		return fmt.Sprintf("%0*s", length, value), nil
	}
	// Normaliza para ASCII puro (CNAB é posicional, 1 byte = 1 posição).
	value = NormalizeASCII(strings.ToUpper(value))
	if len(value) > length {
		value = value[:length]
	}
	return fmt.Sprintf("%-*s", length, value), nil
}

// FormatCurrency formata valor monetário para CNAB (2 decimais, sem ponto).
func FormatCurrency(amount float64) string {
	cents := int64(math.Round(amount * 100))
	return fmt.Sprintf("%015d", cents)
}

// PadRight preenche a string com espaços à direita.
func PadRight(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	return s + strings.Repeat(" ", length-len(s))
}

// PadLeftZeros preenche a string com zeros à esquerda.
func PadLeftZeros(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	return strings.Repeat("0", length-len(s)) + s
}
