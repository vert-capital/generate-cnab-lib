package resolver

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/vert-capital/generate-cnab-lib/types"
)

func (r *Resolver) registerContextResolvers() {
	r.resolvers["context.now_date"] = func(ctx *Context, format string) string {
		return ctx.Now.Format("20060102")
	}
	r.resolvers["context.now_date_ddmmyyyy"] = func(ctx *Context, format string) string {
		return ctx.Now.Format("02012006")
	}
	r.resolvers["context.now_time"] = func(ctx *Context, format string) string {
		return ctx.Now.Format("150405")
	}
	r.resolvers["context.now_datetime"] = func(ctx *Context, format string) string {
		return ctx.Now.Format("20060102150405")
	}
	r.resolvers["context.record_count"] = func(ctx *Context, format string) string {
		return fmt.Sprintf("%06d", ctx.RecordCount)
	}
	r.resolvers["context.total_amount"] = func(ctx *Context, format string) string {
		cents := int64(math.Round(ctx.TotalAmount * 100))
		return strconv.FormatInt(cents, 10)
	}
	r.resolvers["context.lote_servico"] = func(ctx *Context, format string) string {
		return "0001"
	}
	r.resolvers["context.total_file_records"] = func(ctx *Context, format string) string {
		return fmt.Sprintf("%06d", ctx.TotalFileRecords)
	}
	r.resolvers["context.seq_num"] = func(ctx *Context, format string) string {
		return fmt.Sprintf("%05d", ctx.SeqNum)
	}
	r.resolvers["context.total_valor_principal"] = func(ctx *Context, format string) string {
		principal, _, _, _ := calcTributeTotals(ctx.Payments)
		cents := int64(math.Round(principal * 100))
		return strconv.FormatInt(cents, 10)
	}
	r.resolvers["context.total_outras_entidades"] = func(ctx *Context, format string) string {
		_, outras, _, _ := calcTributeTotals(ctx.Payments)
		cents := int64(math.Round(outras * 100))
		return strconv.FormatInt(cents, 10)
	}
	r.resolvers["context.total_val_acrescimos"] = func(ctx *Context, format string) string {
		_, _, acrescimos, _ := calcTributeTotals(ctx.Payments)
		cents := int64(math.Round(acrescimos * 100))
		return strconv.FormatInt(cents, 10)
	}
}

func calcTributeTotals(payments []types.PaymentData) (principal, outrasEntidades, acrescimos, arrecadado float64) {
	for _, p := range payments {
		md := p.Metadata
		switch strings.ToUpper(p.TaxType) {
		case "GPS":
			if m := metaMap(md, "gps"); m != nil {
				principal += getMetaFloatDefault(m, "valor_tributo", p.Amount)
				if v, ok := metaFloat(m, "valor_outr_entidade"); ok {
					outrasEntidades += v
				}
				if v, ok := metaFloat(m, "atualiz_monetaria"); ok {
					acrescimos += v
				}
				arrecadado += getMetaFloatDefault(m, "valor_arrecadado", p.Amount)
				continue
			}
			principal += p.Amount
			arrecadado += p.Amount
		case "DARF":
			if m := metaMap(md, "darf_normal"); m != nil {
				principal += getMetaFloatDefault(m, "valor_principal", p.Amount)
				if v, ok := metaFloat(m, "multa"); ok {
					acrescimos += v
				}
				if v, ok := metaFloat(m, "juros_encargos"); ok {
					acrescimos += v
				}
				arrecadado += getMetaFloatDefault(m, "valor_total", p.Amount)
				continue
			}
			principal += p.Amount
			arrecadado += p.Amount
		case "DARF_SIMPLES":
			if m := metaMap(md, "darf_simples"); m != nil {
				principal += getMetaFloatDefault(m, "valor_principal", p.Amount)
				if v, ok := metaFloat(m, "multa"); ok {
					acrescimos += v
				}
				if v, ok := metaFloat(m, "juros_encargos"); ok {
					acrescimos += v
				}
				arrecadado += getMetaFloatDefault(m, "valor_total", p.Amount)
				continue
			}
			principal += p.Amount
			arrecadado += p.Amount
		default:
			principal += p.Amount
			arrecadado += p.Amount
		}
	}
	return
}

func getMetaFloatDefault(md map[string]interface{}, key string, defaultVal float64) float64 {
	if v, ok := metaFloat(md, key); ok {
		return v
	}
	return defaultVal
}
