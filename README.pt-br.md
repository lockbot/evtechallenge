# EVT Challenge - Plataforma de Dados Clínicos

[![en](https://img.shields.io/badge/lang-en-red.svg)](https://github.com/lockbot/evtechallenge/blob/main/README.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](https://github.com/lockbot/evtechallenge/blob/main/README.pt-br.md)

Uma plataforma de ingestão e API de dados clínicos multi-tenant construída com Go, apresentando ingestão de dados FHIR, persistência Couchbase e observabilidade abrangente.

## Visão Geral da Arquitetura

Esta plataforma consiste em **dois microsserviços** trabalhando juntos para fornecer uma solução completa de dados clínicos:

### Serviços
- **FHIR Client** (`fhir-client/`): Ingere recursos FHIR de API pública para o Couchbase
- **API REST** (`api-rest/`): API REST multi-tenant para acesso a dados e gerenciamento de revisões

### Infraestrutura
- **Couchbase**: Banco de dados de documentos multi-tenant com suporte N1QL
- **Elasticsearch**: Logs centralizados com logs JSON estruturados
- **Prometheus**: Coleta de métricas e monitoramento
- **Grafana**: Visualização e dashboards

## Início Rápido

### Stack Completa (Recomendado)
Inicie a plataforma completa com observabilidade:
```bash
docker-compose --profile observability up
```

### Serviços Individuais

#### Iniciar Apenas o Banco de Dados
```bash
docker-compose up -d evtechallenge-db evtechallenge-db-setup
```

#### Iniciar Apenas o FHIR Client
```bash
docker-compose up -d --no-deps fhir
```

#### Iniciar Apenas a API REST
```bash
docker-compose up -d --no-deps api
```

### Gerenciamento de Serviços

#### Parar Serviços Individuais
```bash
# Parar apenas o cliente FHIR
docker-compose stop fhir

# Parar apenas a API
docker-compose stop api

# Parar apenas o banco de dados
docker-compose stop evtechallenge-db
```

#### Limpeza Completa ⚠️
```bash
# Parar todos os serviços e remover volumes (⚠️AVISO⚠️: DELETA O BANCO DE DADOS)
docker-compose down -v

# Parar todos os serviços mas preservar dados
docker-compose down
```

## Configuração

Crie um arquivo `.env` na raiz do repositório:

```bash
# Configuração da API
API_PORT=8080
API_LOG_LEVEL="info"

# Configuração do FHIR Client
FHIR_PORT=8081
FHIR_LOG_LEVEL="info"
FHIR_BASE_URL=http://hapi.fhir.org/baseR4
FHIR_TIMEOUT=30s

# Configuração do Couchbase
COUCHBASE_URL=couchbase://evt-db
COUCHBASE_ADMINISTRATOR_USERNAME=Administrator
COUCHBASE_ADMINISTRATOR_PASSWORD=password
COUCHBASE_USERNAME=evtechallenge_user
COUCHBASE_PASSWORD=password
COUCHBASE_BUCKET=evtechallenge
COUCHBASE_MANAGEMENT_HOST=evt-db:8091

# Observabilidade (opcional)
ENABLE_ELASTICSEARCH=false
ENABLE_SYSTEM_METRICS=false
ENABLE_BUSINESS_METRICS=false
ELASTICSEARCH_URL=http://elasticsearch:9200

# Configuração do Grafana (opcional - padrões funcionam)
GRAFANA_ADMIN_PASSWORD=admin
GRAFANA_PORT=3000

# Configuração do Elasticsearch (opcional - padrões funcionam)
ELASTICSEARCH_PORT=9200
ELASTICSEARCH_TRANSPORT_PORT=9300

# Configuração do Prometheus (opcional - padrões funcionam)
PROMETHEUS_PORT=9090

# Configuração do Keycloak
KEYCLOAK_URL=http://keycloak:8080
KEYCLOAK_PORT=8082
KEYCLOAK_REALM=evtechallenge
KEYCLOAK_CLIENT_ID=api-client
KEYCLOAK_CLIENT_SECRET=
KEYCLOAK_ADMIN_USER=admin
KEYCLOAK_ADMIN_PASSWORD=admin
# Note: These map to KC_BOOTSTRAP_ADMIN_USERNAME and KC_BOOTSTRAP_ADMIN_PASSWORD in docker-compose.yml
KEYCLOAK_LOG_LEVEL=INFO

# Configuração de Tenants
TENANT1_USERNAME=tenant1
TENANT1_PASSWORD=tnt1
TENANT2_USERNAME=tenant2
TENANT2_PASSWORD=tnt2
```

## Endpoints da API

### Autenticação (Sem tenant necessário)
- `POST /auth/login` - Login do usuário
- `POST /auth/refresh` - Renovar token
- `GET /auth/userinfo` - Obter informações do usuário
- `GET /health` - Verificação de saúde do sistema

### Recursos FHIR (Roteamento baseado em tenant)
- `GET /api/{tenant}/encounters` - Listar encontros do tenant
- `GET /api/{tenant}/encounters/{id}` - Obter encontro específico
- `GET /api/{tenant}/patients` - Listar pacientes do tenant
- `GET /api/{tenant}/patients/{id}` - Obter paciente específico
- `GET /api/{tenant}/practitioners` - Listar profissionais do tenant
- `GET /api/{tenant}/practitioners/{id}` - Obter profissional específico

### Sistema de Revisão (Roteamento baseado em tenant)
- `POST /api/{tenant}/review-request` - Enviar solicitação de revisão

### Sistema
- `GET /` - Informações da API
- `GET /metrics` - Métricas do Prometheus


### Autenticação
Todos os endpoints baseados em tenant requerem autenticação JWT via header `Authorization: Bearer <token>`. O tenant no caminho da URL deve corresponder ao tenant no token JWT.

## Decisões Técnicas

### Decisões de Arquitetura

#### **Separação de Microsserviços**
**Decisão**: Separar ingestão (cliente FHIR) e API (serviço REST) em containers distintos.

**Justificativa**:
- **Escalabilidade Independente**: Pode escalar ingestão e API separadamente baseado na carga
- **Isolamento de Falhas**: Falhas da API não afetam a ingestão de dados
- **Flexibilidade de Deploy**: Pode fazer deploy de atualizações independentemente
- **Otimização de Recursos**: Diferentes requisitos de recursos para cada serviço

#### **Couchbase como Banco de Dados Primário**
**Decisão**: Usar Couchbase para persistência de dados ao invés de RDBMS tradicional.

**Justificativa**:
- **Flexibilidade de Schema**: Estrutura de dados FHIR varia e evolui
- **Desenvolvimento Rápido**: Sem migrações de schema ou modelagem complexa
- **Escalabilidade**: Escalabilidade horizontal com sharding automático
- **Multi-Modelo**: Suporte a chave-valor, documento e consultas N1QL
- **Performance**: Cache em memória com persistência em disco

**Trade-offs**:
- Menos conformidade ACID que bancos tradicionais
- Curva de aprendizado para N1QL vs SQL
- Complexidade operacional para gerenciamento de cluster

#### **Design Multi-Tenant**
**Decisão**: Implementar isolamento de tenant através de documentos de revisão separados.

**Justificativa**:
- **Isolamento Lógico**: Estado de revisão de cada tenant é completamente separado
- **Dados Compartilhados**: Recursos FHIR são compartilhados (custo-efetivo)
- **Escalabilidade**: Fácil adicionar novos tenants sem mudanças de schema
- **Segurança**: Limites claros de dados entre tenants

**Implementação**:
- Identificação de tenant via token JWT e validação do caminho da URL
- Documentos de revisão armazenados como `Review/{tenantID}` com mapas separados para cada tipo de recurso
- Todos os endpoints da API requerem autenticação JWT com validação de tenant

### Decisões de Modelagem de Dados

#### **Estrutura de Documento**
**Decisão**: Desnormalizar relacionamentos para performance de consulta.

**Documentos de Encontro**:
```json
{
  "id": "encounter-123",
  "resourceType": "Encounter",
  "docId": "Encounter/encounter-123",
  "subjectPatientId": "patient-456",
  "practitionerIds": ["practitioner-789"],
  "subject": { "reference": "Patient/patient-456" },
  "participant": [...]
}
```

**Benefícios**:
- Consultas rápidas sem joins
- Acesso direto a IDs relacionados
- Mantém estrutura FHIR original
- Suporta acesso tanto chave-valor quanto N1QL

#### **Estratégia de Resolução de Referências**
**Decisão**: Resolução automática de referências FHIR com tratamento gracioso de falhas.

**Referências Válidas**: `Patient/123`, `Practitioner/456`
**Referências Ignoradas**: `urn:uuid:abc-123-def` (referências de bundle inline)

**Benefícios**:
- Relacionamentos de dados completos
- Trata dados FHIR inconsistentes graciosamente
- Distingue entre referências resolvíveis e não resolvíveis

### Decisões de Observabilidade

#### **Logging Estruturado e Métricas**
**Decisão**: Usar zerolog com formatação JSON e integração Elasticsearch, além de coleta abrangente de métricas Prometheus.

**Benefícios**:
- Logs legíveis por máquina para análise
- Agregação de logs centralizada e correlação entre serviços
- Cobertura abrangente de métricas (requisições HTTP, lógica de negócio, recursos do sistema, chamadas de API FHIR)

## Monitoramento e Observabilidade

### Dashboards Grafana
Acesso em `http://localhost:3000`

**Credenciais de Login**:
- Usuário: `admin`
- Senha: `admin`

**Dashboards Disponíveis**:
- **Métricas de Sistema**: Uso de memória, CPU, contagem de threads
- **Performance da API**: Taxas de requisição, tempos de resposta, taxas de erro
- **Ingestão FHIR**: Contagem de recursos, taxas de sucesso de chamadas de API
- **Métricas de Negócio**: Requisições de revisão, atividade de tenant

**Nota**: Os logs estão disponíveis através da integração Elasticsearch nos dashboards do Grafana.

## Desenvolvimento

### Estrutura do Projeto
```
evtechallenge/
├── api-rest/           # Serviço de API REST multi-tenant
├── fhir-client/        # Serviço de ingestão de dados FHIR
├── config/             # Arquivos de configuração
│   ├── grafana/        # Dashboards e configuração do Grafana
│   └── prometheus/     # Configuração básica do Prometheus
├── docker-compose.yml  # Orquestração de serviços
└── README.md          # Este arquivo
```

### Fluxo de Desenvolvimento
**Arquivos Principais**:
- `docker-compose.yml`: Definições de serviços e rede
- `api-rest/internal/api/`: Implementação do serviço de API
- `fhir-client/internal/fhir/`: Lógica de ingestão FHIR
- `config/grafana/dashboards/`: Dashboards pré-configurados do Grafana
- `config/prometheus/`: Configuração básica do Prometheus (conexão/healthcheck)

**Adicionando Novas Funcionalidades**:
1. **Endpoints de API**: Adicionar em `api-rest/internal/api/handlers.go`
2. **Modelos de Dados**: Definir em `api-rest/internal/api/types.go`
3. **Operações de Banco**: Implementar em `api-rest/internal/api/database.go`
4. **Lógica de Revisão**: Estender `api-rest/internal/api/review.go`

## Solução de Problemas

### Problemas Comuns

#### Falhas de Conexão com Banco de Dados
```bash
# Verificar status do Couchbase
docker-compose logs evtechallenge-db

# Reiniciar banco de dados
docker-compose restart evtechallenge-db
```

#### Serviço de API Indisponível
```bash
# Verificar logs da API
docker-compose logs api

# Verificar se o banco está pronto
curl http://localhost:8080/api/tenant1/patients
```

#### Problemas de Ingestão FHIR
```bash
# Verificar logs do cliente FHIR
docker-compose logs fhir

# Verificar acesso à API externa
curl https://hapi.fhir.org/baseR4/Patient?_count=1
```

**Verificações de Saúde**:
- **Saúde da API**: `GET /` (requer header de tenant)
- **Saúde do Banco**: Verificar UI web do Couchbase em `http://localhost:8091`
- **Saúde das Métricas**: `GET /metrics`

## Documentação

- **API REST**: [api-rest/README.md](api-rest/README.md)
- **FHIR Client**: [fhir-client/README.md](fhir-client/README.md)
- **Docker Compose**: [docker-compose.yml](docker-compose.yml)
- **Configuração do Keycloak**: [docs/keycloak-setup.md](docs/keycloak-setup.md)
- **ADR (Registros de Decisões Arquiteturais)**: [docs/README.md](docs/README.md)

## Segurança e Melhorias Futuras

**Considerações de Segurança**:
- **Isolamento multi-tenant** garante separação de dados
- **Validação de entrada** em todos os endpoints da API
- **Variáveis de ambiente** para configuração sensível
- **Sem credenciais hardcoded** no código fonte

