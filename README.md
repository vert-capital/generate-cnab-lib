# Gerador CNAB

Biblioteca Go para geração e leitura de arquivos CNAB 240 do Itaú (SISPAG Versão 086 - Fevereiro/2024). Suporta pagamentos via PIX (conta), TED, Boletos e Tributos.

## Instalação

```bash
go get github.com/vert-capital/generate-cnab-lib
```

## Funcionalidades

- ✅ **Geração de CNAB 240** - Cria arquivo de remessa para diversos tipos de pagamento
- ✅ **Parse de Retorno** - Lê arquivo de retorno do banco e converte para JSON
- ✅ **Templates validados** - Itau - Em andamento - Bradesco - Em andamento - Santander - Em andamento - BTG Pactual - Em andamento

## Bancos Suportados

| Banco       | Código | Templates                                                                     | Observações                                                                                          |
| ----------- | ------ | ----------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Itaú        | 341    | pix_conta, transferencia, boleto, tributos                                     | Layout SISPAG Versão 086                                                                              |
| Bradesco    | 237    | pix_conta, transferencia, boleto, tributos                                     | Layout MULTIPAG                                                                                       |
| Santander   | 033    | pix_conta, transferencia, boleto, tributos, tributos_barras                    | Layout PagFor V11.7                                                                                   |
| BTG Pactual | 208    | pix_conta, transferencia, boleto, tributos                                     | Layout Febraban V10.9. Tributos: apenas Segmento O (c/ barras, forma 11) e N2/DARF Normal (forma 16) |

## Templates Disponíveis Itau

| Template                | Descrição                         | Tipo Pagamento    | Forma Pagamento            |
| ----------------------- | --------------------------------- | ----------------- | -------------------------- |
| `cnab240_pix_conta`     | PIX via Conta ou Chave Pix        | 98 (Diversos)     | 45                         |
| `cnab240_transferencia` | TED/DOC para outros bancos        | 20 (Fornecedores) | 01, 03, 07, 41, 43         |
| `cnab240_boleto`        | Pagamento de Boletos (Segmento J) | 30 (Títulos)      | 30=Itaú / 31=Outros Bancos |
| `cnab240_tributos`      | Tributos (Segmento O ou N)        | 22 (Tributos)     | conforme tipo              |

## Uso

### 1. PIX (`cnab240_pix_conta`)

O template suporta dois modos de envio PIX. Pelo menos um dos dois modos deve ser informado por pagamento:
- **PIX via Conta** (dados bancários): `recipient_agency` + `recipient_account`
- **PIX via Chave**: `recipient_pix_key` + `metadata.key_type`

#### 1.1 PIX via Conta

Transferência PIX para conta corrente/poupança utilizando dados bancários.

```json
{
  "external_id": "PIX-001",
  "origin_id": 1,
  "bank_code": "341",
  "template_name": "cnab240_pix_conta",
  "company": {
    "cnpj": "12345678000195",
    "company_name": "EMPRESA LTDA",
    "bank_code": "341",
    "agency": "1234",
    "account": "123456",
    "account_digit": "0",
    "address": "RUA DA EMPRESA",
    "address_number": "00100",
    "city": "SAO PAULO",
    "cep": "01001000",
    "state": "SP"
  },
  "payments": [
    {
      "external_id": "PAY-001",
      "recipient_document": "98765432000196",
      "recipient_company_name": "FORNECEDOR PIX",
      "recipient_bank": "341",
      "recipient_agency": "0001",
      "recipient_agency_digit": "9",
      "recipient_account": "12345",
      "recipient_account_digit": "1",
      "ispb": "60701190",
      "amount": 500.75,
      "due_date": "20260410"
    }
  ]
}
```

#### 1.2 PIX via Chave

Transferência PIX utilizando chave PIX (CPF, CNPJ, Email, Celular ou Chave Aleatória).

```json
{
  "external_id": "PIX-CHAVE-001",
  "origin_id": 1,
  "bank_code": "341",
  "template_name": "cnab240_pix_conta",
  "company": {
    "cnpj": "12345678000195",
    "company_name": "EMPRESA LTDA",
    "bank_code": "341",
    "agency": "1234",
    "account": "123456",
    "account_digit": "0"
  },
  "payments": [
    {
      "external_id": "PAY-001",
      "recipient_document": "98765432000196",
      "recipient_company_name": "FORNECEDOR PIX",
      "recipient_bank": "341",
      "ispb": "60701190",
      "amount": 500.75,
      "due_date": "20260410",
      "recipient_pix_key": "123e4567-e89b-12d3-a456-426614174000",
      "metadata": {
        "key_type": "04"
      }
    }
  ]
}
```

**Mapeamento de `key_type` (código Itaú - Nota 37):**

| Tipo | Código |
|------|--------|
| Telefone | `01` |
| E-mail | `02` |
| CPF/CNPJ | `03` |
| Chave Aleatória (EVP) | `04` |

> Ao enviar `recipient_pix_key`, o campo `metadata.key_type` é obrigatório. Os campos `recipient_agency` e `recipient_account` podem ser omitidos.
>
> **Retrocompatibilidade:** A biblioteca também aceita os códigos `01` a `05` do padrão anterior (BACEN) e converte internamente para o padrão Itaú:
> - `05` (Chave Aleatória BACEN) → `04` (Itaú)
> - `04` (Celular BACEN) → `01` (Telefone Itaú)

**Campos específicos:**

| Campo                     | Obrigatório | Descrição                                   |
| ------------------------- | ----------- | ------------------------------------------- |
| `recipient_bank`          | Sim         | Código do banco do favorecido               |
| `recipient_agency`        | Condicional | Agência do favorecido (obrigatório para PIX via conta) |
| `recipient_account`       | Condicional | Conta do favorecido (obrigatório para PIX via conta)   |
| `recipient_account_digit` | Condicional | Dígito da conta (obrigatório para PIX via conta)       |
| `recipient_pix_key`       | Condicional | Chave PIX do favorecido (obrigatório para PIX via chave) |
| `metadata.key_type`       | Condicional | Tipo da chave PIX conforme tabela Itaú Nota 37 (obrigatório para PIX via chave) |
| `ispb`                    | Sim         | ISPB do banco destino (ex: 60701190 = Itaú) |

> **NOTA 44:** Para PIX conta pagamento, o campo agência (posição 225-229 do Segmento A) é preenchido automaticamente a partir de `company.agency`.
> **NOTA 8 (PIX):** Ao cancelar transferência PIX, não ocorre geração de arquivo retorno. Se o arquivo já foi gerado e a Van recebeu não será possivel cancelar via sistema

---

### 2. Transferência TED (`cnab240_transferencia`)

TED para contas de outros bancos.

> **NOTA 45:** DOC foi descontinuado em 15/01/2024. Utilize TED (`41` ou `43`) para transferências.

```json
{
  "external_id": "TED-001",
  "origin_id": 1,
  "bank_code": "341",
  "template_name": "cnab240_transferencia",
  "company": {
    "cnpj": "12345678000195",
    "company_name": "EMPRESA LTDA",
    "bank_code": "341",
    "agency": "1234",
    "agency_digit": "5",
    "account": "123456",
    "account_digit": "0",
    "address": "RUA DA EMPRESA",
    "address_number": "00100",
    "city": "SAO PAULO",
    "cep": "01001000",
    "state": "SP"
  },
  "payments": [
    {
      "external_id": "PAY-001",
      "recipient_document": "98765432000196",
      "recipient_company_name": "FORNECEDOR XYZ",
      "recipient_bank": "237",
      "recipient_agency": "5678",
      "recipient_agency_digit": "0",
      "recipient_account": "876543",
      "recipient_account_digit": "2",
      "payment_method": "TED",
      "ted_purpose": "00001",
      "amount": 1500.5,
      "due_date": "20260410"
    }
  ]
}
```

**Campos específicos:**

| Campo                     | Obrigatório | Descrição                                           |
| ------------------------- | ----------- | --------------------------------------------------- |
| `recipient_bank`          | Sim         | Código do banco do favorecido                       |
| `recipient_agency`        | Sim         | Agência do favorecido                               |
| `recipient_account`       | Sim         | Conta do favorecido                                 |
| `recipient_account_digit` | Não         | Dígito da conta                                     |
| `ispb`                    | Sim         | ISPB do banco destino                               |
| `ted_purpose`             | Não         | Código de finalidade (padrão: 00005 = Fornecedores) |
| `payment_method`          | Não         | Forma de pagamento (padrão: 41). Ver opções abaixo  |

**Formas de Pagamento (campo `payment_method`):**

| Código | Descrição                 | Quando usar                                     |
| ------ | ------------------------- | ----------------------------------------------- |
| `01`   | Crédito em Conta Corrente | Transferência para conta do mesmo banco         |
| `03`   | ~~DOC C~~                 | ⚠️ Descontinuado desde 15/01/2024 (NOTA 45)     |
| `07`   | ~~DOC D~~                 | ⚠️ Descontinuado desde 15/01/2024 (NOTA 45)     |
| `41`   | TED - Outro Titular       | TED para conta de outra pessoa/empresa (padrão) |
| `43`   | TED - Mesmo Titular       | TED para conta da mesma empresa (mesmo CNPJ)    |

⚠️ **Importante:** O sistema detecta automaticamente se é "mesmo titular" ou "outro titular" baseado no CNPJ da empresa vs CPF/CNPJ do favorecido quando o campo não é informado.

---

### 3. Pagamento de Boletos (`cnab240_boleto`)

Pagamento de títulos/boletos de cobrança.

```json
{
  "external_id": "BOLETO-001",
  "origin_id": 1,
  "bank_code": "341",
  "template_name": "cnab240_boleto",
  "company": {
    "cnpj": "12345678000195",
    "company_name": "EMPRESA LTDA",
    "bank_code": "341",
    "agency": "1234",
    "account": "123456",
    "account_digit": "0",
    "address": "RUA DA EMPRESA",
    "address_number": "00100",
    "city": "SAO PAULO",
    "cep": "01001000",
    "state": "SP"
  },
  "payments": [
    {
      "external_id": "PAY-001",
      "recipient_document": "98765432000196",
      "recipient_company_name": "BOLETO EMPRESA",
      "barcode": "341917900000000000000000000000000000000000000000",
      "amount": 1250.3,
      "due_date": "20260410"
    }
  ]
}
```

**Campo obrigatório:** `barcode` com a linha digitável (47 dígitos) ou código de barras (44 dígitos). O sistema converte automaticamente quando necessário.

**Campos específicos:**

| Campo                    | Obrigatório | Descrição                                         |
| ------------------------ | ----------- | ------------------------------------------------- |
| `barcode`                | **Sim**     | Código de barras completo do boleto (44 posições) |
| `recipient_company_name` | Sim         | Nome do favorecido (sacado)                       |

⚠️ **Importante:**

- **Forma de Pagamento:**
  - `30` = Bloqueto do Itaú (mesma titularidade)
  - `31` = Bloqueto de Outros Bancos (obrigatório informar dados do cedente)
  - `32` = Nota Fiscal (Liquidação Eletrônica)
- Para boletos de outros bancos (forma `31`), o sistema gera automaticamente o Segmento J-52 (dados do sacado/cedente). Certifique-se de que `recipient_document` e `recipient_company_name` estejam corretos.

---

### 4. Pagamento de Tributos (`cnab240_tributos`)

Suporta tributos **com** e **sem** código de barras. O template escolhe automaticamente o segmento correto:

- **Com código de barras** (Segmento `O`): concessionárias, GNRE, etc.
- **Sem código de barras** (Segmento `N`): DARF Normal, DARF Simples, GPS, GARE-SP, IPVA, DPVAT, FGTS.

#### Exemplo: DARF Simples (sem código de barras)

```json
{
  "external_id": "darf-simples-batch-001",
  "bank_code": "341",
  "company": {
    "cnpj": "12345667889765",
    "company_name": "TESTE COMP SECURITIZADORA",
    "bank_code": "341",
    "agency": "1234",
    "account": "123456",
    "account_digit": "5",
    "address": "RUA DA EMPRESA",
    "address_number": "00100",
    "city": "SAO PAULO",
    "cep": "01001000",
    "state": "SP"
  },
  "payments": [
    {
      "external_id": "DARF-1708-001",
      "template_name": "cnab240_tributos",
      "recipient_document": "12345667889765",
      "recipient_company_name": "TESTE COMP SECURITIZADORA",
      "tax_type": "DARF_SIMPLES",
      "revenue_code": "1708",
      "competence": "31032026",
      "amount": 37.5,
      "due_date": "20260420",
      "metadata": {
        "darf_simples": {
          "receita_bruta": 0,
          "percentual": 0
        }
      }
    }
  ]
}
```

#### Exemplo: Tributo com código de barras - OBS: NÃO HOMOLOGADO

```json
{
  "external_id": "tributo-batch-001",
  "bank_code": "341",
  "company": {
    "cnpj": "12345678000195",
    "company_name": "EMPRESA LTDA",
    "bank_code": "341",
    "agency": "1234",
    "account": "123456",
    "account_digit": "5",
    "address": "RUA DA EMPRESA",
    "address_number": "00100",
    "city": "SAO PAULO",
    "cep": "01001000",
    "state": "SP"
  },
  "payments": [
    {
      "external_id": "PAY-001",
      "template_name": "cnab240_tributos",
      "recipient_document": "12345678901",
      "recipient_company_name": "TESTE LTDA",
      "barcode": "85820000000150000123045123040123456789012345",
      "tax_type": "CONCESSIONARIA",
      "amount": 150.0,
      "due_date": "20260402",
      "description": "Pagamento Concessionaria"
    }
  ]
}
```

**Campos específicos:**

| Campo                    | Obrigatório | Descrição                                                                           |
| ------------------------ | ----------- | ----------------------------------------------------------------------------------- |
| `barcode`                | Condicional | Obrigatório apenas para tributos **com** código de barras (48 posições numéricas)   |
| `tax_type`               | **Sim**     | Tipo de tributo. Determina o segmento e a forma de pagamento automaticamente        |
| `revenue_code`           | Condicional | Código da receita (usado em DARF, DARF Simples, GPS, etc.)                          |
| `competence`             | Condicional | Período de apuração/competência no formato `DDMMAAAA`                               |
| `metadata.dados_tributo` | Condicional | String pronta de 178 posições (modo legado). Ainda suportado para compatibilidade   |
| `metadata.darf_normal`   | Condicional | Objeto JSON com campos do DARF Normal. A lib monta `dados_tributo` automaticamente  |
| `metadata.darf_simples`  | Condicional | Objeto JSON com campos do DARF Simples. A lib monta `dados_tributo` automaticamente |
| `metadata.gps`           | Condicional | Objeto JSON com campos da GPS. A lib monta `dados_tributo` automaticamente          |

**Valores para `tax_type`:**

| Valor          | Forma Pagamento | Descrição                  | Código de Barras | Homologado |
| -------------- | --------------- | -------------------------- | ---------------- | ---------- |
| `DARF`         | 16              | DARF Normal                | Não              | Não        |
| `GPS`          | 17              | Guia da Previdência Social | Não              | Não        |
| `DARF_SIMPLES` | 18              | DARF Simples               | Não              | Sim        |
| `GARE_SP_ICMS` | 22              | GARE - SP ICMS             | Não              | Não        |
| `IPVA`         | 25              | IPVA                       | Não              | Não        |
| `DPVAT`        | 27              | DPVAT                      | Não              | Não        |
| `FGTS`         | 35              | FGTS - GFIP                | Não              | Não        |

**Campos estruturados para auto-build de `dados_tributo`:**

A biblioteca pode montar automaticamente o campo `dados_tributo` (178 posições) a partir de objetos JSON no `metadata`. Se você não passar a string pronta em `metadata.dados_tributo`, a lib procurará pelas chaves abaixo conforme o `tax_type`:

#### `metadata.darf_normal`

| Campo              | Tipo   | Padrão                           | Descrição                              |
| ------------------ | ------ | -------------------------------- | -------------------------------------- |
| `receita`          | string | `payment.revenue_code`           | Código da receita (4 posições)         |
| `tipo_inscricao`   | string | `2`                              | `1`=CPF, `2`=CNPJ                      |
| `numero_inscricao` | string | `payment.recipient_document`     | CPF/CNPJ do contribuinte (14 posições) |
| `periodo`          | string | `payment.competence`             | Período de apuração DDMMAAAA           |
| `referencia`       | string | —                                | Número de referência (17 posições)     |
| `valor_principal`  | number | `payment.amount`                 | Valor principal                        |
| `multa`            | number | `0`                              | Valor da multa                         |
| `juros_encargos`   | number | `0`                              | Valor de juros/encargos                |
| `valor_total`      | number | `payment.amount`                 | Valor total a pagar                    |
| `data_vencimento`  | string | `payment.due_date` (AAAAMMDD)    | Data de vencimento                     |
| `data_pagamento`   | string | `payment.due_date` (AAAAMMDD)    | Data do pagamento                      |
| `contribuinte`     | string | `payment.recipient_company_name` | Nome do contribuinte (30 posições)     |

#### `metadata.darf_simples`

| Campo              | Tipo   | Padrão                           | Descrição                              |
| ------------------ | ------ | -------------------------------- | -------------------------------------- |
| `receita`          | string | `payment.revenue_code`           | Código da receita (4 posições)         |
| `tipo_inscricao`   | string | `2`                              | `1`=CPF, `2`=CNPJ                      |
| `numero_inscricao` | string | `payment.recipient_document`     | CPF/CNPJ do contribuinte (14 posições) |
| `periodo`          | string | `payment.competence`             | Período de apuração DDMMAAAA           |
| `receita_bruta`    | number | `0`                              | Valor da receita bruta acumulada       |
| `percentual`       | number | `0`                              | Percentual sobre receita bruta         |
| `valor_principal`  | number | `payment.amount`                 | Valor principal                        |
| `multa`            | number | `0`                              | Valor da multa                         |
| `juros_encargos`   | number | `0`                              | Valor de juros/encargos                |
| `valor_total`      | number | `payment.amount`                 | Valor total a pagar                    |
| `data_vencimento`  | string | `payment.due_date` (AAAAMMDD)    | Data de vencimento                     |
| `data_pagamento`   | string | `payment.due_date` (AAAAMMDD)    | Data do pagamento                      |
| `contribuinte`     | string | `payment.recipient_company_name` | Nome do contribuinte (30 posições)     |

#### `metadata.gps`

| Campo                 | Tipo   | Padrão                           | Descrição                           |
| --------------------- | ------ | -------------------------------- | ----------------------------------- |
| `codigo_pagto`        | string | —                                | Código de pagamento (4 posições)    |
| `competencia`         | string | `payment.competence`             | Mês/ano da competência MMAAAA       |
| `identificador`       | string | `payment.recipient_document`     | CNPJ/CEI/NIT/PIS (14 posições)      |
| `valor_tributo`       | number | `payment.amount`                 | Valor previsto do pagamento         |
| `valor_outr_entidade` | number | `0`                              | Valor de outras entidades           |
| `atualiz_monetaria`   | number | `0`                              | Atualização monetária               |
| `valor_arrecadado`    | number | `payment.amount`                 | Valor arrecadado                    |
| `data_arrecadacao`    | string | `payment.due_date` (AAAAMMDD)    | Data da arrecadação                 |
| `uso_empresa`         | string | —                                | Informações complementares (50 pos) |
| `contribuinte`        | string | `payment.recipient_company_name` | Nome do contribuinte (30 posições)  |

---

## Campos Comuns (todos os templates)

| Campo                               | Obrigatório | Formato           | Descrição                               |
| ----------------------------------- | ----------- | ----------------- | --------------------------------------- |
| `external_id`                       | Sim         | string            | Identificador único do lote             |
| `bank_code`                         | Sim         | string            | Código do banco (341 = Itaú)            |
| `company.cnpj`                      | Sim         | string (14)       | CNPJ da empresa pagadora                |
| `company.company_name`              | Sim         | string            | Nome da empresa                         |
| `company.agency`                    | Sim         | string            | Agência da conta de débito              |
| `company.account`                   | Sim         | string            | Conta corrente de débito                |
| `company.account_digit`             | Sim         | string            | Dígito da conta                         |
| `company.address`                   | Não         | string            | Endereço da empresa                     |
| `company.address_number`            | Não         | string            | Número do endereço                      |
| `company.city`                      | Não         | string            | Cidade da empresa                       |
| `company.cep`                       | Não         | string            | CEP da empresa                          |
| `company.state`                     | Não         | string            | Sigla do estado (ex: SP)                |
| `payments[].external_id`            | Sim         | string (20)       | Identificador do pagamento (seu número) |
| `payments[].template_name`          | Sim         | string            | Template a ser usado                    |
| `payments[].recipient_document`     | Sim         | string (11 ou 14) | CPF/CNPJ do favorecido                  |
| `payments[].recipient_company_name` | Sim         | string (30)       | Nome do favorecido                      |
| `payments[].amount`                 | Sim         | number            | Valor do pagamento                      |
| `payments[].due_date`               | Sim         | string (AAAAMMDD) | Data de pagamento                       |
| `payments[].description`            | Não         | string (40)       | Descrição/observação                    |

---

## Uso em Go

### Gerar Arquivo CNAB

```go
import (
    "context"
    "encoding/json"
    cnab "github.com/vert-capital/generate-cnab-lib"
)

var input cnab.Input
jsonBytes := []byte(`{...}`)  // JSON conforme exemplos acima
if err := json.Unmarshal(jsonBytes, &input); err != nil {
    panic(err)
}

result, err := cnab.Generate(context.Background(), input, input.Payments[0].TemplateName)
if err != nil {
    panic(err)
}

// result.Content - conteúdo do arquivo CNAB (240 posições por linha)
// result.TotalRecords - quantidade de pagamentos
// result.TotalAmount - valor total
os.WriteFile("remessa.rem", []byte(result.Content), 0644)
```

### Uso Programático (Structs)

```go
import (
    "context"
    cnab "github.com/vert-capital/generate-cnab-lib"
)

input := cnab.Input{
    ExternalID: "batch-001",
    BankCode:   "341",
    Company: cnab.CompanyData{
        CNPJ:         "12345678000195",
        CompanyName:  "EMPRESA LTDA",
        BankCode:     "341",
        Agency:       "1234",
        Account:      "123456",
        AccountDigit: "5",
    },
    Payments: []cnab.PaymentData{
        {
            ExternalID:           "PAY-001",
            TemplateName:         "cnab240_pix_conta",
            RecipientDocument:    "12345678901",
            RecipientCompanyName: "JOSE DA SILVA",
            RecipientBank:        "341",
            RecipientAgency:      "5678",
            RecipientAccount:     "876543",
            ISPB:                 "60701190",
            Amount:               1500.75,
            DueDate:              "20260402",
        },
    },
}

result, err := cnab.Generate(context.Background(), input, "cnab240_pix_conta")
```

**Exemplo com PIX via Chave:**

```go
inputPixKey := cnab.Input{
    ExternalID: "PIX-CHAVE-001",
    BankCode:   "341",
    Company: cnab.CompanyData{
        CNPJ:         "12345678000195",
        CompanyName:  "EMPRESA LTDA",
        BankCode:     "341",
        Agency:       "1234",
        Account:      "123456",
        AccountDigit: "5",
    },
    Payments: []cnab.PaymentData{
        {
            ExternalID:           "PAY-001",
            TemplateName:         "cnab240_pix_conta",
            RecipientDocument:    "12345678901",
            RecipientCompanyName: "JOSE DA SILVA",
            RecipientBank:        "341",
            RecipientPixKey:      "123e4567-e89b-12d3-a456-426614174000",
            ISPB:                 "60701190",
            Amount:               500.75,
            DueDate:              "20260410",
            Metadata: map[string]interface{}{
                "key_type": "05",
            },
        },
    },
}

result, err := cnab.Generate(context.Background(), inputPixKey, "cnab240_pix_conta")
```

---

## Parse de Arquivo de Retorno

```go
import (
    "context"
    cnab "github.com/vert-capital/generate-cnab-lib"
)

// Lê o arquivo de retorno enviado pelo banco
content, _ := os.ReadFile("retorno.rem")

// Faz o parse
result, err := cnab.ParseReturnFile(context.Background(), string(content), "341", "cnab240_pix_conta")
if err != nil {
    log.Fatal(err)
}

// Acessa os dados
fmt.Printf("Empresa: %s\n", result.CompanyName)
fmt.Printf("CNPJ: %s\n", result.CompanyCNPJ)
fmt.Printf("Total de registros: %d\n", result.TotalRecords)

for _, record := range result.Records {
    fmt.Printf("Seu Número: %s\n", record.YourNumber)
    fmt.Printf("Ocorrência: %s - %s\n", record.OccurrenceCode, record.OccurrenceDescription)
    fmt.Printf("Valor Pago: R$ %.2f\n", record.PaidAmount)
    fmt.Printf("Data Pagamento: %s\n", record.PaymentDate)
}

// Exporta para JSON
jsonBytes, _ := json.MarshalIndent(result, "", "  ")
os.WriteFile("retorno.json", jsonBytes, 0644)
```

**Campos do retorno:**

| Campo                             | Descrição                             |
| --------------------------------- | ------------------------------------- |
| `BankCode`                        | Código do banco                       |
| `CompanyCNPJ`                     | CNPJ da empresa                       |
| `CompanyName`                     | Nome da empresa                       |
| `GenerationDate`                  | Data de geração do arquivo            |
| `TotalRecords`                    | Quantidade de registros               |
| `Records`                         | Lista de pagamentos processados       |
| `Records[].YourNumber`            | Número do documento (sua referência)  |
| `Records[].OurNumber`             | Número atribuído pelo banco           |
| `Records[].OccurrenceCode`        | Código da ocorrência (ex: 00, BD, RJ) |
| `Records[].OccurrenceDescription` | Descrição da ocorrência               |
| `Records[].PaidAmount`            | Valor efetivamente pago               |
| `Records[].PaymentDate`           | Data efetiva do pagamento             |
| `Records[].PrimarySegment`        | Campos do segmento principal parseado |
| `Records[].SecondarySegment`      | Campos do segmento complementar       |

### Exemplos de JSON de saída do parse

#### 1. PIX via Conta (`cnab240_pix_conta_retorno`)

```json
{
  "bank_code": "341",
  "company_cnpj": "12345678000195",
  "company_name": "EMPRESA LTDA",
  "generation_date": "20260414",
  "total_records": 1,
  "records": [
    {
      "your_number": "PAY-001",
      "our_number": "E004010021202604140001ABCD1234",
      "occurrence_code": "00",
      "occurrence_description": "Crédito ou Débito Efetivado",
      "paid_amount": 1500.75,
      "payment_date": "20260414",
      "scheduled_date": "",
      "segment_type": "J",
      "primary_segment": {
        "codigo_banco": "341",
        "lote_servico": "0001",
        "tipo_registro": "3",
        "numero_registro": "00001",
        "segmento": "J",
        "tipo_movimento": "000",
        "codigo_reg_opcional": "52",
        "tipo_inscricao_devedor": "2",
        "numero_inscricao_devedor": "12345678000195",
        "nome_devedor": "EMPRESA LTDA",
        "tipo_inscricao_favorecido": "1",
        "numero_inscricao_favorecido": "12345678901",
        "nome_favorecido": "JOSE DA SILVA",
        "chave_pagamento": "12345678901",
        "txid": "E004010021202604140001ABCD1234"
      },
      "secondary_segment": {
        "codigo_banco": "341",
        "lote_servico": "0001",
        "tipo_registro": "3",
        "numero_registro": "00002",
        "segmento": "Z",
        "tipo_autenticacao": "0001",
        "autenticacao": "A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6A7B8",
        "end_to_end_id": "E004010021202604140001ABCD1234",
        "data_hora_pagamento": "20260414153045",
        "valor_pago": "000000000150075",
        "status_pagamento": "00",
        "codigo_erro": "",
        "ocorrencias": "00"
      }
    }
  ],
  "occurrences": { "00": "Crédito ou Débito Efetivado" }
}
```

#### 2. Transferência TED (`cnab240_transferencia_retorno`)

```json
{
  "bank_code": "341",
  "company_cnpj": "12345678000195",
  "company_name": "EMPRESA LTDA",
  "generation_date": "20260414",
  "total_records": 1,
  "records": [
    {
      "your_number": "PAY-001",
      "our_number": "98765432",
      "occurrence_code": "00",
      "occurrence_description": "Pagamento Efetuado",
      "paid_amount": 5000.0,
      "payment_date": "20260414",
      "scheduled_date": "20260414",
      "recipient_name": "FORNECEDOR LTDA",
      "recipient_document": "98765432000195",
      "segment_type": "A",
      "primary_segment": {
        "codigo_banco": "341",
        "lote_servico": "0001",
        "tipo_registro": "3",
        "numero_registro": "00001",
        "segmento": "A",
        "tipo_movimento": "000",
        "banco_favorecido": "341",
        "agencia_conta": "5678            876543",
        "nome_favorecido": "FORNECEDOR LTDA",
        "seu_numero": "PAY-001",
        "data_pagamento": "20260414",
        "moeda": "REA",
        "valor_pagamento": "000000000050000",
        "nosso_numero": "98765432",
        "data_efetiva": "20260414",
        "valor_efetivo": "000000000050000",
        "finalidade_ted": "00005",
        "aviso": "0",
        "ocorrencias": "00"
      },
      "secondary_segment": {
        "codigo_banco": "341",
        "lote_servico": "0001",
        "tipo_registro": "3",
        "numero_registro": "00002",
        "segmento": "Z",
        "autenticacao": "A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6A7B8",
        "seu_numero": "PAY-001",
        "nosso_numero": "98765432"
      }
    }
  ],
  "occurrences": { "00": "Pagamento Efetuado" }
}
```

#### 3. Boleto (`cnab240_boleto_retorno`)

```json
{
  "bank_code": "341",
  "company_cnpj": "12345678000195",
  "company_name": "EMPRESA LTDA",
  "generation_date": "20260414",
  "total_records": 1,
  "records": [
    {
      "your_number": "PAY-001",
      "our_number": "123456789012345",
      "occurrence_code": "00",
      "occurrence_description": "Pagamento Efetuado",
      "paid_amount": 1500.75,
      "payment_date": "20260414",
      "scheduled_date": "",
      "recipient_name": "FORNECEDOR LTDA",
      "recipient_document": "",
      "segment_type": "J",
      "primary_segment": {
        "codigo_banco": "341",
        "lote_servico": "0001",
        "tipo_registro": "3",
        "numero_registro": "00001",
        "segmento": "J",
        "tipo_movimento": "000",
        "codigo_barras": "34191681600015000751000123456789012345678901",
        "nome_favorecido": "FORNECEDOR LTDA",
        "data_vencimento": "20260420",
        "valor_titulo": "000000000150075",
        "desconto": "000000000000000",
        "acrescimo": "000000000000000",
        "data_pagamento": "20260414",
        "valor_pagamento": "000000000150075",
        "nosso_numero": "123456789012345",
        "ocorrencias": "00"
      },
      "secondary_segment": {
        "codigo_banco": "341",
        "lote_servico": "0001",
        "tipo_registro": "3",
        "numero_registro": "00003",
        "segmento": "Z",
        "autenticacao": "A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6A7B8",
        "protocolo": "PROTOC12345",
        "nosso_numero": "123456789012345"
      }
    }
  ],
  "occurrences": { "00": "Pagamento Efetuado" }
}
```

#### 4. Tributos (`cnab240_tributos_retorno`)

```json
{
  "bank_code": "341",
  "company_cnpj": "12345678000195",
  "company_name": "EMPRESA LTDA",
  "generation_date": "20260414",
  "total_records": 1,
  "records": [
    {
      "your_number": "PAY-001",
      "our_number": "",
      "occurrence_code": "00",
      "occurrence_description": "Pagamento Efetuado",
      "paid_amount": 150.0,
      "payment_date": "20260414",
      "scheduled_date": "",
      "recipient_name": "CONCESSIONARIA LTDA",
      "recipient_document": "",
      "segment_type": "O",
      "primary_segment": {
        "codigo_banco": "341",
        "lote_servico": "0001",
        "tipo_registro": "3",
        "numero_registro": "00001",
        "segmento": "O",
        "tipo_movimento": "000",
        "codigo_barras": "85820000000150000123045123040123456789012345",
        "nome_favorecido": "CONCESSIONARIA LTDA",
        "data_vencimento": "20260420",
        "data_pagamento": "20260414",
        "valor_pagamento": "000000000015000",
        "valor_documento": "000000000015000",
        "numero_documento": "PAY-001",
        "ocorrencias": "00"
      },
      "secondary_segment": {
        "codigo_banco": "341",
        "lote_servico": "0001",
        "tipo_registro": "3",
        "numero_registro": "00002",
        "segmento": "Z",
        "autenticacao": "A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6A7B8",
        "seu_numero": "PAY-001",
        "nosso_numero": ""
      }
    }
  ],
  "occurrences": { "00": "Pagamento Efetuado" }
}
```

---

## Templates de Retorno

| Template                        | Descrição                 | Status     |
| ------------------------------- | ------------------------- | ---------- |
| `cnab240_pix_conta_retorno`     | Retorno de PIX            | ✅ Testado |
| `cnab240_boleto_retorno`        | Retorno de Boletos        | ✅ Testado |
| `cnab240_transferencia_retorno` | Retorno de Transferências | ✅ Testado |
| `cnab240_tributos_retorno`      | Retorno de Tributos       | ✅ Testado |

---

## Validações Importantes

### PIX (cnab240_pix_conta)

- Código de compensação do banco favorecido deve ser informado
- ISPB é obrigatório para identificação da instituição
- CPF/CNPJ do favorecido é validado
- Pelo menos um dos dois modos deve ser informado por pagamento:
  - **PIX via Conta:** `recipient_agency` + `recipient_account`
  - **PIX via Chave:** `recipient_pix_key` + `metadata.key_type`
- **NOTA 44:** Agência é preenchida automaticamente na posição 225-229 do Segmento A (obrigatório para PIX via conta)
- **NOTA 46:** Campo `external_id` (Seu Número) é obrigatório
- **NOTA 8:** Cancelamento de PIX não gera arquivo retorno; PIX agendado só pode ser cancelado via Itaú na Internet

### TED (cnab240_transferencia)

- Câmara de compensação é preenchida automaticamente como `888`
- Tipo de pagamento é `20` (Fornecedores)
- Forma de pagamento pode ser: `01` (Crédito), `41` (TED Outro Titular) ou `43` (TED Mesmo Titular)
- ⚠️ DOC (`03` e `07`) descontinuado desde 15/01/2024 (NOTA 45)
- Padrão é `41` quando não informado
- Finalidade TED padrão é `00005` (pagamento a fornecedores)

### Boleto (cnab240_boleto)

- Código de barras deve ter 44 posições
- Segmento J-52 é gerado automaticamente para boletos de outros bancos
- Informe corretamente o `recipient_document` e `recipient_company_name` para o J-52

### Tributo (cnab240_tributos)

- O campo `tax_type` é obrigatório e determina a forma de pagamento e o segmento a ser usado
- **Com código de barras** (`CONCESSIONARIA`, `GNRE`): `barcode` obrigatório com 48 posições numéricas. Usa Segmento `O`
- No Itaú, o campo 018-065 do Segmento O leva a **representação numérica de 48 dígitos** (11 dígitos + DV por campo). O código de barras de 44 é aceito e convertido, com os DVs de campo recompostos — gravar os 44 crus deixa 062-065 em branco e o banco recusa com "caracter inválido na posição 54"
- No Itaú, o par **tipo x forma** do header do lote sai do segmento da guia: prefeitura (segmento 1) `22 x 19`, concessionária (2, 3 e 4) `20 x 13`, órgão governamental e demais `22 x 91`. Um `payment_method` informado no payload é ignorado nesse caso
- **Sem código de barras** (`DARF`, `DARF_SIMPLES`, `GPS`, `GARE_SP_ICMS`, `IPVA`, `DPVAT`, `FGTS`): `metadata.dados_tributo` obrigatório com 178 posições. Usa Segmento `N`

## Licença

MIT
