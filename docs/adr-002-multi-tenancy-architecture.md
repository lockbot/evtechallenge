# ADR-002: Arquitetura Multi-Tenant com Goroutines Dinâmicos

## Status
Aceito

## Data
2024-09-03

## Contexto
A aplicação precisa suportar múltiplos clientes B2B (tenants) com isolamento completo de dados. O sistema deve:

- Manter dados de cada tenant separados
- Permitir inicialização sob demanda de novos tenants
- Gerenciar recursos de forma eficiente
- Fornecer isolamento de performance entre tenants
- Suportar warm-up/cool-down automático baseado em uso

## Decisão
Implementar arquitetura multi-tenant com **collections separadas por tenant** e **sistema de goroutines dinâmicos** com warm-up/cool-down automático.

## Consequências

### Positivas
- **Isolamento completo**: Cada tenant tem sua própria collection no Couchbase
- **Eficiência de recursos**: Goroutines só rodam para tenants ativos
- **Escalabilidade**: Sistema pode suportar centenas de tenants sem degradação
- **Flexibilidade**: Novos tenants são inicializados automaticamente
- **Performance**: Sem overhead para tenants inativos

### Negativas
- **Complexidade**: Sistema de goroutines com gerenciamento de estado
- **Debugging**: Mais difícil de rastrear problemas específicos de tenant
- **Recursos**: Cada tenant ativo consome memória para goroutine

## Implementação

### Arquitetura de Dados
1. **`_default`**: Collection master com dados FHIR (populada uma vez)
2. **`tenant_<id>`**: Collection específica para cada tenant (cópia de `_default`)

### Sistema de Goroutines
1. **`TenantGoroutineManager`**: Gerencia ciclo de vida dos tenants
2. **Mapa de canais**: `map[string]chan struct{}` para comunicação
3. **Timeout automático**: 30 minutos de inatividade = goroutine para
4. **Warm-up sob demanda**: Endpoint `/warm-up-tenant` para ativação

### Fluxo de Vida do Tenant
```
Inativo → Warm-up Request → Inicialização → Ativo → 30min Inatividade → Cold → Stop Goroutine
```

### Rotas Protegidas
- **Sempre acessíveis**: `hello`, `all-good`, `metrics`, `health`, `warm-up-tenant`
- **Protegidas por warm-up**: `encounters`, `patients`, `practitioners`, `review-request`

### Middleware de Proteção
- **`TenantWarmthMiddleware`**: Verifica se tenant está "quente" antes de permitir acesso
- **Resposta 503**: Quando tenant está "frio", instrui cliente a chamar warm-up

## Alternativas Consideradas

### 1. Multi-tenancy por Header (X-Tenant-ID)
- **Prós**: Implementação simples, sem overhead
- **Contras**: Sem isolamento real, risco de vazamento de dados

### 2. Banco de dados separado por tenant
- **Prós**: Isolamento máximo, performance previsível
- **Contras**: Complexidade de gerenciamento, custo alto

### 3. Schema único com filtros
- **Prós**: Simplicidade, uma collection
- **Contras**: Performance degrada com volume, difícil de otimizar

## Estrutura de Documentos

### Status de Ingestão FHIR
```json
{
  "_id": "_system/ingestion_status",
  "ready": true,
  "startedAt": "2024-01-01T10:00:00Z",
  "completedAt": "2024-01-01T10:05:00Z"
}
```

### Status do Tenant
```json
{
  "_id": "_system/tenant_status",
  "tenantId": "tenant123",
  "ready": true,
  "warmedAt": "2024-01-01T10:00:00Z",
  "lastRequest": "2024-01-01T10:30:00Z"
}
```

### Recursos FHIR (com campo de revisão)
```json
{
  "_id": "Encounter/123",
  "resourceType": "Encounter",
  "reviewed": false,
  "reviewTime": null
}
```

## Implementação Técnica

### Componentes Principais
- **`TenantGoroutineManager`**: Gerencia goroutines e estado dos tenants
- **`TenantWarmthMiddleware`**: Protege rotas FHIR
- **`TenantCollectionManager`**: Gerencia collections dos tenants
- **`/warm-up-tenant`**: Endpoint para ativação de tenants

### Gerenciamento de Estado
- **Thread-safe**: Uso de `sync.RWMutex` para operações concorrentes
- **Singleton global**: Instância única do gerenciador de tenants
- **Context cancellation**: Graceful shutdown de goroutines

### Timeouts e Inatividade
- **Check a cada minuto**: Ticker para verificar inatividade
- **30 minutos**: Timeout configurável para inatividade
- **Activity recording**: Cada request mantém tenant "quente"

## Referências
- [Go Concurrency Patterns](https://golang.org/doc/effective_go.html#concurrency)
- [Couchbase Collections](https://docs.couchbase.com/server/current/learn/data/collections.html)
- [Multi-tenancy Patterns](https://martinfowler.com/articles/microservices.html#SharedData)

## Revisão
Este ADR deve ser revisado quando:
- Houver mudanças nos requisitos de performance
- Novos padrões de multi-tenancy se tornarem disponíveis
- A equipe identificar problemas de escalabilidade
- Mudanças na estratégia de gerenciamento de recursos
