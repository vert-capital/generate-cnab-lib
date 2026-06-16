package formatter

import (
	"fmt"
	"math"
	"strings"
)

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
