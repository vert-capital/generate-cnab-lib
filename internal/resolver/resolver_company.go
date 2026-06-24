package resolver

import (
	"strings"

	"github.com/vert-capital/generate-cnab-lib/internal/formatter"
)

func (r *Resolver) registerCompanyResolvers() {
	r.resolvers["company.cnpj"] = func(ctx *Context, format string) string {
		return ctx.Company.CNPJ
	}
	r.resolvers["company.name"] = func(ctx *Context, format string) string {
		return ctx.Company.CompanyName
	}
	r.resolvers["company.company_name"] = func(ctx *Context, format string) string {
		return ctx.Company.CompanyName
	}
	r.resolvers["company.bank_code"] = func(ctx *Context, format string) string {
		return ctx.Company.BankCode
	}
	r.resolvers["company.agency"] = func(ctx *Context, format string) string {
		return ctx.Company.Agency
	}
	r.resolvers["company.agency_digit"] = func(ctx *Context, format string) string {
		return ctx.Company.AgencyDigit
	}
	r.resolvers["company.account"] = func(ctx *Context, format string) string {
		return ctx.Company.Account
	}
	r.resolvers["company.account_digit"] = func(ctx *Context, format string) string {
		return ctx.Company.AccountDigit
	}
	r.resolvers["company.convenio"] = func(ctx *Context, format string) string {
		conv := ctx.Company.Convenio
		// Santander (033) - Nota G009: o campo de 20 posições é composto por
		// BBBB (banco) + AAAA (agência sem DV) + CCCCCCCCCCCC (convênio à direita).
		// Se o usuário já informou as 20 posições, usa como veio.
		if ctx.Company.BankCode == "033" && len(conv) < 20 {
			banco := formatter.PadLeftZeros(ctx.Company.BankCode, 4)
			agencia := formatter.PadLeftZeros(ctx.Company.Agency, 4)
			convenio := formatter.PadLeftZeros(conv, 12)
			return banco + agencia + convenio
		}
		return conv
	}
	r.resolvers["company.inscription_type"] = func(ctx *Context, format string) string {
		return "2"
	}
	r.resolvers["company.address"] = func(ctx *Context, format string) string {
		return ctx.Company.Address
	}
	r.resolvers["company.address_number"] = func(ctx *Context, format string) string {
		return ctx.Company.AddressNumber
	}
	r.resolvers["company.address_complement"] = func(ctx *Context, format string) string {
		return ctx.Company.AddressComplement
	}
	r.resolvers["company.neighborhood"] = func(ctx *Context, format string) string {
		return ctx.Company.Neighborhood
	}
	r.resolvers["company.city"] = func(ctx *Context, format string) string {
		return ctx.Company.City
	}
	r.resolvers["company.state"] = func(ctx *Context, format string) string {
		return ctx.Company.State
	}
	r.resolvers["company.cep"] = func(ctx *Context, format string) string {
		return ctx.Company.CEP
	}

	r.resolvers["company.cep_prefix"] = func(ctx *Context, format string) string {
		cep := strings.ReplaceAll(ctx.Company.CEP, "-", "")
		if len(cep) >= 5 {
			return cep[0:5]
		}
		return cep
	}
	r.resolvers["company.cep_suffix"] = func(ctx *Context, format string) string {
		cep := strings.ReplaceAll(ctx.Company.CEP, "-", "")
		if len(cep) >= 8 {
			return cep[5:8]
		}
		if len(cep) > 5 {
			return cep[5:]
		}
		return "000"
	}
}
