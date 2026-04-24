// Package types contém os tipos compartilhados entre os pacotes da biblioteca CNAB.
package types

// Input representa o payload de entrada para geração do CNAB.
type Input struct {
	ExternalID string        `json:"external_id"`
	OriginID   int           `json:"origin_id"`
	BankCode   string        `json:"bank_code"`
	Company    CompanyData   `json:"company"`
	Payments   []PaymentData `json:"payments"`
}

// CompanyData dados da empresa pagadora.
type CompanyData struct {
	CNPJ              string `json:"cnpj"`
	CompanyName       string `json:"company_name"`
	BankCode          string `json:"bank_code"`
	Agency            string `json:"agency"`
	AgencyDigit       string `json:"agency_digit"`
	Account           string `json:"account"`
	AccountDigit      string `json:"account_digit"`
	Convenio          string `json:"convenio,omitempty"`
	Address           string `json:"address,omitempty"`
	AddressNumber     string `json:"address_number,omitempty"`
	AddressComplement string `json:"address_complement,omitempty"`
	Neighborhood      string `json:"neighborhood,omitempty"`
	City              string `json:"city,omitempty"`
	State             string `json:"state,omitempty"`
	CEP               string `json:"cep,omitempty"`
}

// PaymentData dados de um pagamento individual.
type PaymentData struct {
	ExternalID            string                 `json:"external_id"`
	TemplateName          string                 `json:"template_name,omitempty"`
	RecipientDocument     string                 `json:"recipient_document"`
	RecipientCompanyName  string                 `json:"recipient_company_name"`
	RecipientBank         string                 `json:"recipient_bank"`
	RecipientAgency       string                 `json:"recipient_agency"`
	RecipientAgencyDigit  string                 `json:"recipient_agency_digit,omitempty"`
	RecipientAccount      string                 `json:"recipient_account"`
	RecipientPixKey       string                 `json:"recipient_pix_key,omitempty"`
	RecipientAccountDigit string                 `json:"recipient_account_digit,omitempty"`
	ISPB                  string                 `json:"ispb,omitempty"`
	AccountType           string                 `json:"account_type,omitempty"` // 01=CC, 02=Poupança, 03=Pagamento
	PaymentMethod         string                 `json:"payment_method,omitempty"`
	TEDPurpose            string                 `json:"ted_purpose,omitempty"`
	TaxType               string                 `json:"tax_type,omitempty"`
	Amount                float64                `json:"amount"`
	PaymentType           string                 `json:"payment_type,omitempty"`
	Description           string                 `json:"description,omitempty"`
	DueDate               string                 `json:"due_date,omitempty"` // AAAAMMDD
	Barcode               string                 `json:"barcode,omitempty"`
	TXID                  string                 `json:"txid,omitempty"`
	OurNumber             string                 `json:"our_number,omitempty"`
	Discount              float64                `json:"discount,omitempty"`
	Interest              float64                `json:"interest,omitempty"`
	InvoiceNumber         string                 `json:"invoice_number,omitempty"`
	RevenueCode           string                 `json:"revenue_code,omitempty"`
	Competence            string                 `json:"competence,omitempty"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// Result resultado da geração do arquivo CNAB.
type Result struct {
	Content      string
	TotalRecords int
	TotalAmount  float64
	Warnings     []string
}

// ReturnRecord representa um registro de detalhe do arquivo de retorno.
type ReturnRecord struct {
	YourNumber            string            `json:"your_number"`
	OurNumber             string            `json:"our_number"`
	OccurrenceCode        string            `json:"occurrence_code"`
	OccurrenceDescription string            `json:"occurrence_description"`
	PaymentDate           string            `json:"payment_date"`
	ScheduledDate         string            `json:"scheduled_date"`
	PaidAmount            float64           `json:"paid_amount"`
	OriginalAmount        float64           `json:"original_amount"`
	RecipientName         string            `json:"recipient_name"`
	RecipientDocument     string            `json:"recipient_document"`
	SegmentType           string            `json:"segment_type"`
	PrimarySegment        map[string]string `json:"primary_segment,omitempty"`
	SecondarySegment      map[string]string `json:"secondary_segment,omitempty"`
}

// ReturnFile representa o arquivo de retorno CNAB parseado.
type ReturnFile struct {
	BankCode       string            `json:"bank_code"`
	CompanyCNPJ    string            `json:"company_cnpj"`
	CompanyName    string            `json:"company_name"`
	GenerationDate string            `json:"generation_date"`
	TotalRecords   int               `json:"total_records"`
	Records        []ReturnRecord    `json:"records"`
	Occurrences    map[string]string `json:"occurrences"`
}
