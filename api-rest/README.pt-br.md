# Serviço API REST

[![en](https://img.shields.io/badge/lang-en-red.svg)](https://github.com/lockbot/evtechallenge/blob/main/api-rest/README.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](https://github.com/lockbot/evtechallenge/blob/main/api-rest/README.pt-br.md)

Este serviço expõe uma API REST multi-tenant para acesso a dados clínicos e gerenciamento de revisões. Integra com Couchbase para persistência de dados e fornece observabilidade abrangente através de logging estruturado e métricas.

## Visão Geral da Arquitetura

O serviço de API implementa uma **arquitetura multi-tenant** com isolamento lógico através de documentos de revisão específicos por tenant. O estado de revisão de cada tenant é armazenado separadamente, garantindo isolamento completo de dados entre clientes.

### Design Multi-Tenant
- **Identificação de Tenant**: Todas as requisições requerem header `X-Tenant-ID`
- **Isolamento de Revisões**: Revisões são armazenadas como documentos separados (`Review/{tenantID}`)
- **Acesso a Dados**: Recursos FHIR são compartilhados, mas status de revisão é específico por tenant

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
- `COUCHBASE_BUCKET=evtechallenge`
- `API_PORT=8080`
- `ELASTICSEARCH_URL=http://elasticsearch:9200`


## Endpoints da API

### Saúde e Status
- `GET /` - Verificação de saúde com validação de tenant
- `GET /hello` - Endpoint simples de hello (requer header de tenant)
- `POST /all-good` - Endpoint de validação de lógica de negócio (requer header de tenant)
- `GET /metrics` - Endpoint de métricas Prometheus

### Endpoints de Recursos FHIR

Todos os endpoints requerem header `X-Tenant-ID` e retornam status de revisão para o tenant solicitante.

#### Encontros
- `GET /encounters` - Listar todos os encontros com status de revisão
- `GET /encounters/{id}` - Obter encontro específico com status de revisão

#### Pacientes
- `GET /patients` - Listar todos os pacientes com status de revisão
- `GET /patients/{id}` - Obter paciente específico com status de revisão

#### Profissionais
- `GET /practitioners` - Listar todos os profissionais com status de revisão
- `GET /practitioners/{id}` - Obter profissional específico com status de revisão

### Paginação

Todos os endpoints de lista suportam paginação usando parâmetros de query:

- `?count=<número>` - Número de itens por página (padrão: 100, máximo: 10000)
- `?page=<número>` - Número da página (padrão: 1)

**Exemplo:**
```bash
# Obter primeiros 50 encontros
GET /encounters?count=50&page=1

# Obter segunda página de 25 pacientes
GET /patients?count=25&page=2

# Obter primeiros 100 profissionais (padrão)
GET /practitioners
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
- `POST /review-request` - Marcar um recurso para revisão

## Sistema de Revisão Multi-Tenant

### Estrutura do Documento de Revisão
Cada tenant tem um documento de revisão com mapas separados para diferentes tipos de recursos:

```json
{
  "tenantId": "tenant-abc",
  "updated": "2024-01-15T10:30:00Z",
  "encounters": {
    "Encounter/456": { "reviewRequested": true, "reviewTime": "..." }
  },
  "patients": {
    "Patient/123": { "reviewRequested": true, "reviewTime": "..." }
  },
  "practitioners": {
    "Practitioner/789": { "reviewRequested": true, "reviewTime": "..." }
  }
}
```

### Endpoint de Requisição de Revisão
```bash
POST /review-request
Headers: X-Tenant-ID: your-tenant-id
Body: {
  "entity": "encounter",
  "id": "encounter-123"
}
```

### Formato de Resposta
Todos os endpoints de recursos retornam status de revisão:

**Recurso Individual:**
```json
{
  "reviewed": true,
  "reviewTime": "2024-01-15T10:30:00Z",
  "data": { /* dados do recurso FHIR */ }
}
```

**Lista de Recursos:**
```json
[
  {
    "id": "encounter-123",
    "resource": {
      "reviewed": true,
      "reviewTime": "2024-01-15T10:30:00Z",
      "entityType": "Encounter",
      "entityID": "encounter-123",
      /* ... outros dados FHIR */
    }
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
- **Header Ausente**: `400 Bad Request` - "missing required header: X-Tenant-ID"
- **Header Vazio**: `400 Bad Request` - "tenant ID cannot be empty"

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
