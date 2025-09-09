# Serviço FHIR Client

[![en](https://img.shields.io/badge/lang-en-red.svg)](https://github.com/lockbot/evtechallenge/blob/main/fhir-client/README.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](https://github.com/lockbot/evtechallenge/blob/main/fhir-client/README.pt-br.md)

Serviço Go que ingere recursos FHIR da API pública HAPI FHIR para o Couchbase com processamento concorrente, resiliência e observabilidade abrangente.

## Visão Geral da Arquitetura

O cliente FHIR implementa um **sistema de ingestão de duas fases**:

1. **Ingestão Primária**: Busca e armazena encontros, pacientes e profissionais
2. **Resolução de Referências**: Sincroniza automaticamente recursos relacionados quando referenciados em encontros
3. **Flag de Banco Pronto**: Define uma flag global (`template/ingestion_status`) quando a ingestão está completa para coordenação do serviço de API

### Princípios de Design
- **Processamento Concorrente**: Múltiplas goroutines para ingestão paralela de recursos
- **Operações Resilientes**: Tratamento gracioso de falhas de API e timeouts
- **Integridade de Referências**: Resolução automática de referências FHIR
- **Observabilidade**: Logging abrangente e métricas para visibilidade operacional

## Início Rápido

1) Iniciar Couchbase e inicialização:
```bash
docker-compose up -d evtechallenge-db evtechallenge-db-setup
```

2) Iniciar o cliente:
```bash
docker-compose up -d fhir
```

## Configuração

Variáveis de ambiente:
- `COUCHBASE_URL=couchbase://evt-db`
- `COUCHBASE_USERNAME=evtechallenge_user`
- `COUCHBASE_PASSWORD=password`
- `COUCHBASE_BUCKET=EvTeChallenge`
- `FHIR_PORT=8081`
- `FHIR_LOG_LEVEL=info`
- `FHIR_BASE_URL=http://hapi.fhir.org/baseR4`
- `FHIR_TIMEOUT=30s`
- `ELASTICSEARCH_URL=http://elasticsearch:9200`


## Processo de Ingestão

### Tipos de Recursos Ingeridos
- **Encontros**: Foco principal com referências de pacientes/profissionais
- **Pacientes**: Referenciados por encontros via `subject.reference`
- **Profissionais**: Referenciados por encontros via `participant[].individual.reference`

### Fluxo de Dados
1. **Busca de Bundles**: Recupera bundles FHIR da API pública
2. **Classificação de Recursos**: Identifica tipos de recursos (Encounter/Patient/Practitioner)
3. **Armazenamento Primário**: Armazena recursos com campos desnormalizados
4. **Resolução de Referências**: Busca recursos referenciados ausentes
5. **Banco Pronto**: Define flag global (`template/ingestion_status`) quando completo

### Estrutura de Documento

**Documentos de Encontro** (`Encounter/{id}`):
```json
{
  "id": "encounter-123",
  "resourceType": "Encounter",
  "docId": "Encounter/encounter-123",
  "subjectPatientId": "patient-456",
  "practitionerIds": ["practitioner-789", "practitioner-101"],
  "subject": { "reference": "Patient/patient-456" },
  "participant": [
    { "individual": { "reference": "Practitioner/practitioner-789" } }
  ]
}
```

**Documentos de Paciente/Profissional** (`Patient/{id}`, `Practitioner/{id}`):
```json
{
  "id": "patient-456",
  "resourceType": "Patient",
  "docId": "Patient/patient-456",
  // ... dados do recurso FHIR
}
```

## Resolução de Referências

### Padrões de Referência Válidos
- `Patient/123` → Busca paciente com ID "123"
- `Practitioner/456` → Busca profissional com ID "456"

### Padrões de Referência Ignorados
- `urn:uuid:abc-123-def` → **Ignorado** (referências de bundle inline)
- Estas referências não podem ser resolvidas via API pública FHIR

### Tratamento de Referências Ausentes
- `subject.reference` ausente → Nenhuma sincronização de paciente tentada
- `participant[].individual.reference` ausente → Nenhuma sincronização de profissional tentada
- Chamadas de API falhadas → Logadas como avisos, ingestão continua

## Observabilidade

### Logging
- **Logs JSON estruturados** com zerolog
- **Rastreamento no nível de recurso** (operações de busca, armazenamento, sincronização)
- **Contexto de erro** com stack traces
- **Métricas de performance** (duração de busca, tempo de armazenamento)

### Métricas
- **Chamadas de API FHIR**: Taxas de sucesso/falha, tempos de resposta
- **Operações Couchbase**: Contagens de upsert, duração, erros
- **Contagem de recursos**: Encontros, pacientes, profissionais ingeridos
- **Métricas de sistema**: Uso de memória, contagem de goroutines

### Monitoramento
- **Dashboards Grafana**: `http://localhost:3000`
- **Métricas Prometheus**: Disponível para alertas

## Tratamento de Erros e Resiliência

### Falhas de API
- **Tratamento de timeout**: Timeouts configuráveis com lógica de retry
- **Erros HTTP**: Degradação graciosa com logging detalhado
- **Problemas de rede**: Retry automático com backoff exponencial

### Inconsistências de Dados
- **Campos ausentes**: Tratamento gracioso de dados FHIR incompletos
- **Referências inválidas**: Avisos logados, ingestão continua
- **Recursos duplicados**: Operações de upsert idempotentes

### Problemas de Banco de Dados
- **Falhas de conexão**: Tentativas automáticas de reconexão
- **Erros de armazenamento**: Logging detalhado de erros com contexto
- **Falhas de consulta**: Fallback para operações chave-valor

## Sugestões de Melhorias

### 🔧 **Flag para Identificadores Falhados**
**Estado Atual**: Resolução de identificadores falhados é logada mas não sinalizada para revisão.

**Melhoria Sugerida**:
```go
// Adicionar à estrutura ReviewDocument
type ReviewDocument struct {
    TenantID string                 `json:"tenantId"`
    Encounters map[string]interface{} `json:"encounters"`
    Patients map[string]interface{} `json:"patients"`
    Practitioners map[string]interface{} `json:"practitioners"`
    FailedIdentifiers []FailedIdentifier `json:"failedIdentifiers,omitempty"`
    Updated  time.Time              `json:"updated"`
}

type FailedIdentifier struct {
    Reference   string `json:"reference"`
    ResourceType string `json:"resourceType"`
    Reason      string `json:"reason"` // "urn:uuid", "api_failure", "not_found"
    Timestamp   time.Time `json:"timestamp"`
}
```

**Benefícios**:
- Rastrear todas as resoluções de identificadores falhadas
- Distinguir entre `urn:uuid` (esperado) vs falhas de API (acionável)
- Fornecer trilha de auditoria para problemas de qualidade de dados
- Habilitar alertas automatizados para falhas sistemáticas

### 🔧 **Classificação de Erros Aprimorada**
**Estado Atual**: Todas as falhas são logadas como avisos.

**Melhoria Sugerida**:
```go
type IngestionError struct {
    Type        string `json:"type"` // "urn_uuid", "api_failure", "network_timeout"
    Reference   string `json:"reference"`
    ResourceType string `json:"resourceType"`
    Severity    string `json:"severity"` // "info", "warning", "error"
    Retryable   bool   `json:"retryable"`
}
```

**Benefícios**:
- Melhor categorização de erros para monitoramento
- Distinguir entre falhas esperadas vs inesperadas
- Habilitar alertas direcionados e resposta
- Suporte para estratégias de retry

### 🔧 **Rastreamento de Status de Ingestão**
**Estado Atual**: Flag simples `template/ingestion_status` no DefaultScope.

**Implementação Atual**:
```go
type IngestionStatus struct {
    Ready       bool      `json:"ready"`
    Message     string    `json:"message"`
    Updated     time.Time `json:"updated"`
}
```

**Melhoria Sugerida**:
```go
type IngestionStatus struct {
    Status      string    `json:"status"` // "running", "completed", "failed"
    StartTime   time.Time `json:"startTime"`
    EndTime     time.Time `json:"endTime,omitempty"`
    Resources   ResourceCounts `json:"resources"`
    Errors      []IngestionError `json:"errors,omitempty"`
    FailedRefs  []FailedIdentifier `json:"failedRefs,omitempty"`
}

type ResourceCounts struct {
    Encounters    int `json:"encounters"`
    Patients      int `json:"patients"`
    Practitioners int `json:"practitioners"`
}
```
