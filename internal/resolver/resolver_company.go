package resolver

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
		if ctx.Company.AgencyDigit == "" {
			return "0"
		}
		return ctx.Company.AgencyDigit
	}
	r.resolvers["company.account"] = func(ctx *Context, format string) string {
		return ctx.Company.Account
	}
	r.resolvers["company.account_digit"] = func(ctx *Context, format string) string {
		return ctx.Company.AccountDigit
	}
	r.resolvers["company.convenio"] = func(ctx *Context, format string) string {
		return ctx.Company.Convenio
	}
	r.resolvers["company.covenant"] = func(ctx *Context, format string) string {
		return ctx.Company.Convenio
	}
	r.resolvers["company.inscription_type"] = func(ctx *Context, format string) string {
		return "2"
	}
	r.resolvers["company.tipo_inscricao"] = func(ctx *Context, format string) string {
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
	r.resolvers["company.cep_complement"] = func(ctx *Context, format string) string {
		return ""
	}
}
