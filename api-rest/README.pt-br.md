# Serviço API REST

[![en](https://img.shields.io/badge/lang-en-red.svg)](https://github.com/lockbot/evtechallenge/blob/main/api-rest/README.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](https://github.com/lockbot/evtechallenge/blob/main/api-rest/README.pt-br.md)

Este serviço expõe uma API REST multi-tenant para acesso a dados clínicos e gerenciamento de revisões. Integra com Couchbase para persistência de dados e fornece observabilidade abrangente através de logging estruturado e métricas.

## Visão Geral da Arquitetura

O serviço de API implementa uma **arquitetura multi-tenant** com isolamento lógico completo através de scopes e collections do Couchbase. Cada tenant possui seu próprio scope dedicado com collections separadas, garantindo isolamento completo de dados e escalabilidade automática.

### Design Multi-Tenant
- **Scopes de Tenant**: Cada tenant recebe seu próprio scope do Couchbase (ex: `tenant1`, `tenant2`)
- **Criação Automática**: Scopes e collections são criados automaticamente no primeiro acesso do tenant
- **Isolamento de Dados**: Separação física completa dos dados do tenant
- **Integração de Revisão**: Campos de revisão (`reviewed`, `reviewTime`) são incorporados diretamente nos documentos FHIR
- **Performance**: Consultas diretas sem filtros de tenant, aproveitando índices nativos do Couchbase

## Início Rápido

1) Iniciar serviços necessários
```bash
docker-compose up -d evtechallenge-db evtechallenge-db-setup
```

2) Iniciar a API (se habilitada no compose) ou executar localmente
```bash
# via compose (descomente no docker-compose.yml)
docker-compose up -d api

# ou executar localmente
go run ./api-rest
```

## Configuração

Variáveis de ambiente:
- `COUCHBASE_URL=couchbase://evt-db`
- `COUCHBASE_USERNAME=evtechallenge_user`
- `COUCHBASE_PASSWORD=password`
- `COUCHBASE_BUCKET=EvTeChallenge`
- `API_PORT=8080`
- `API_LOG_LEVEL=info`
- `ELASTICSEARCH_URL=http://elasticsearch:9200`


## Endpoints da API

### Saúde e Status
- `GET /` - Verificação de saúde com validação de tenant
- `GET /hello` - Endpoint simples de hello (requer header de tenant)
- `POST /all-good` - Endpoint de validação de lógica de negócio (requer header de tenant)
- `GET /metrics` - Endpoint de métricas Prometheus

### Endpoints de Recursos FHIR

Todos os endpoints usam roteamento baseado em tenant (`/api/{tenant}/...`) e retornam recursos FHIR com status de revisão incorporado.

#### Encontros
- `GET /api/{tenant}/encounters` - Listar todos os encontros com status de revisão incorporado
- `GET /api/{tenant}/encounters/{id}` - Obter encontro específico com status de revisão incorporado

#### Pacientes
- `GET /api/{tenant}/patients` - Listar todos os pacientes com status de revisão incorporado
- `GET /api/{tenant}/patients/{id}` - Obter paciente específico com status de revisão incorporado

#### Profissionais
- `GET /api/{tenant}/practitioners` - Listar todos os profissionais com status de revisão incorporado
- `GET /api/{tenant}/practitioners/{id}` - Obter profissional específico com status de revisão incorporado

### Paginação

Todos os endpoints de lista suportam paginação usando parâmetros de query:

- `?count=<número>` - Número de itens por página (padrão: 100, máximo: 10000)
- `?page=<número>` - Número da página (padrão: 1)

**Exemplo:**
```bash
# Obter primeiros 50 encontros para tenant1
GET /api/tenant1/encounters?count=50&page=1

# Obter segunda página de 25 pacientes para tenant1
GET /api/tenant1/patients?count=25&page=2

# Obter primeiros 100 profissionais para tenant1 (padrão)
GET /api/tenant1/practitioners
```

**Formato de Resposta Paginada:**
```json
{
  "data": [
    {
      "id": "encounter-123",
      "resource": { /* dados do recurso FHIR */ }
    }
  ],
  "pagination": {
    "page": 1,
    "count": 50,
    "offset": 0,
    "totalItems": 50,
    "hasNext": true
  }
}
```

**Nota:** O Couchbase tem um limite padrão de 100 documentos por consulta. Use paginação para acessar conjuntos de dados maiores de forma eficiente.

### Gerenciamento de Revisões
- `POST /api/{tenant}/review-request` - Marcar um recurso para revisão

## Arquitetura Multi-Tenant

### Estrutura de Scope do Tenant
Cada tenant possui seu próprio scope do Couchbase com collections dedicadas:

**DefaultScope** (Dados Template):
- `encounters`: Dados FHIR originais de encontros
- `patients`: Dados FHIR originais de pacientes  
- `practitioners`: Dados FHIR originais de profissionais
- `_default`: Status de ingestão do sistema (`template/ingestion_status`)

**Scopes de Tenant** (ex: `tenant1`, `tenant2`):
- `encounters`: Dados de encontros específicos do tenant com campos de revisão incorporados
- `patients`: Dados de pacientes específicos do tenant com campos de revisão incorporados
- `practitioners`: Dados de profissionais específicos do tenant com campos de revisão incorporados
- `defaulty`: Status de ingestão do tenant (`tenant/ingestion_status`)

### Integração de Revisão
Campos de revisão são incorporados diretamente nos documentos FHIR:

```json
{
  "id": "encounter-123",
  "resourceType": "Encounter",
  "reviewed": true,
  "reviewTime": "2024-01-15T10:30:00Z",
  "subject": { "reference": "Patient/patient-456" },
  "participant": [...]
}
```

### Endpoint de Requisição de Revisão
```bash
POST /api/tenant1/review-request
Headers: Authorization: Bearer <jwt-token>
Body: {
  "entity": "encounter",
  "id": "encounter-123"
}
```

### Formato de Resposta
Todos os endpoints de recursos retornam recursos FHIR com status de revisão incorporado:

**Recurso Individual:**
```json
{
  "id": "encounter-123",
  "resourceType": "Encounter",
  "reviewed": true,
  "reviewTime": "2024-01-15T10:30:00Z",
  "subject": { "reference": "Patient/patient-456" },
  "participant": [...]
}
```

**Lista de Recursos:**
```json
[
  {
    "id": "encounter-123",
    "resourceType": "Encounter",
    "reviewed": true,
    "reviewTime": "2024-01-15T10:30:00Z",
    "subject": { "reference": "Patient/patient-456" },
    "participant": [...]
  }
]
```

## Relacionamentos de Dados FHIR

### Estrutura de Encontro
Encontros contêm referências a pacientes e profissionais:

```json
{
  "id": "encounter-123",
  "resourceType": "Encounter",
  "subject": {
    "reference": "Patient/patient-456"  // ID do Paciente
  },
  "participant": [
    {
      "individual": {
        "reference": "Practitioner/practitioner-789"  // ID do Profissional
      }
    }
  ]
}
```

### Regras de Extração de Identificadores

O sistema extrai identificadores de referências FHIR seguindo estas regras:

1. **Referências Válidas**: Formato `ResourceType/ID`
   - `Patient/123` → extrai `123`
   - `Practitioner/456` → extrai `456`

2. **Referências Ignoradas**: Formato `urn:uuid:`
   - `urn:uuid:abc-123-def` → **ignorado** (não resolvível via API FHIR)
   - Estas são referências de bundle inline que não podem ser sincronizadas

3. **Referências Ausentes**: Tratadas graciosamente
   - `subject.reference` ausente → sem sincronização de paciente
   - `participant[].individual.reference` ausente → sem sincronização de profissional

### Desnormalização de Dados

Para performance e eficiência de consulta, o sistema desnormaliza relacionamentos:

- **Documentos de Encontro** incluem:
  - `subjectPatientId`: Referência direta ao ID do paciente
  - `practitionerIds`: Array de IDs de profissionais
  - `docId`: Chave canônica do documento (`Encounter/{id}`)

- **Documentos de Paciente/Profissional** incluem:
  - `docId`: Chave canônica do documento (`Patient/{id}` ou `Practitioner/{id}`)

## Tratamento de Erros

### Validação de Tenant
- **Tenant Inválido**: `400 Bad Request` - "invalid tenant in URL path"
- **JWT Incompatível**: `403 Forbidden` - "tenant in URL does not match JWT token"

### Operações de Recursos
- **Não Encontrado**: `404 Not Found` - "resource not found"
- **Banco Indisponível**: `503 Service Unavailable` - "database not initialized"
- **Entidade Inválida**: `400 Bad Request` - "invalid entity" (para requisições de revisão)

## Observabilidade

### Logging
- Logs JSON estruturados com zerolog
- ID do tenant incluído em todas as entradas de log
- Correlação requisição/resposta
- Contexto de erro e stack traces

### Métricas
- Contagens e durações de requisições HTTP
- Métricas de lógica de negócio (requisições de revisão, falhas de validação)
- Métricas de sistema (memória, threads, conexões)
- Disponível no endpoint `/metrics`

### Monitoramento
- Dashboards Grafana disponíveis em `http://localhost:3000`
- Métricas Prometheus para alertas e tendências
