package resolver

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vert-capital/generate-cnab-lib/types"
)

var (
	saoPauloLocation *time.Location
	saoPauloOnce     sync.Once
)

// Context contém os dados disponíveis para resolução de campos.
type Context struct {
	Company          types.CompanyData
	Payments         []types.PaymentData
	CurrentPayment   *types.PaymentData
	PaymentIndex     int
	SeqNum           int
	Now              time.Time
	RecordCount      int
	TotalFileRecords int
	TotalAmount      float64
	LineLength       int
	TemplateName     string
	Warnings         *[]string // ponteiro para compartilhar entre cópias de contexto
}

// Resolver resolve campos baseado em source (ex: "company.cnpj", "payment.amount").
type Resolver struct {
	resolvers map[string]ValueResolver
}

// ValueResolver função que resolve um campo.
type ValueResolver func(ctx *Context, format string) string

// New cria um novo resolvedor de campos.
func New() *Resolver {
	r := &Resolver{
		resolvers: make(map[string]ValueResolver),
	}
	r.registerDefaults()
	return r
}

// Resolve um campo baseado no source.
func (r *Resolver) Resolve(source string, ctx *Context, format string) (string, error) {
	if resolver, ok := r.resolvers[source]; ok {
		return resolver(ctx, format), nil
	}

	// Tratamento especial para metadata (campos dinâmicos)
	parts := strings.Split(source, ".")
	if parts[0] == "payment" && len(parts) >= 3 && parts[1] == "metadata" {
		key := parts[2]
		if key == "" {
			return "", nil
		}
		if ctx.CurrentPayment == nil || ctx.CurrentPayment.Metadata == nil {
			return "", nil
		}
		if val, ok := ctx.CurrentPayment.Metadata[key]; ok {
			return interfaceToString(val), nil
		}
		return "", nil
	}

	return "", fmt.Errorf("source não encontrado: %s", source)
}

// Register registra um novo resolvedor.
func (r *Resolver) Register(source string, resolver ValueResolver) {
	r.resolvers[source] = resolver
}

// paymentResolver cria um ValueResolver que extrai um campo do pagamento atual.
// Elimina a repetição do padrão nil-check em resolvers simples.
func paymentResolver(fn func(*types.PaymentData) string) ValueResolver {
	return func(ctx *Context, format string) string {
		if ctx.CurrentPayment != nil {
			return fn(ctx.CurrentPayment)
		}
		return ""
	}
}

// ---------------------------------------------------------------------------
// Registro de resolvers por domínio
// ---------------------------------------------------------------------------

func (r *Resolver) registerDefaults() {
	r.registerCompanyResolvers()
	r.registerPaymentResolvers()
	r.registerBarcodeResolvers()
	r.registerContextResolvers()
}

// ---------------------------------------------------------------------------
// Funções utilitárias
// ---------------------------------------------------------------------------

// SaoPauloLocation retorna o timezone de São Paulo.
func SaoPauloLocation() *time.Location {
	saoPauloOnce.Do(func() {
		var err error
		saoPauloLocation, err = time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			saoPauloLocation = time.FixedZone("America/Sao_Paulo", -3*60*60)
		}
	})
	return saoPauloLocation
}
