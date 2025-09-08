# Architecture Decision Record (ADR): Autenticação JWT e Roteamento Baseado em Tenant

**Data**: 2024-01-15  
**Responsável**: Equipe de Desenvolvimento  
**Status**: Aceito  

## Contexto

Ao implementar autenticação e autorização para a plataforma de dados clínicos EVT Challenge, foi necessário decidir sobre a estratégia de autenticação multi-tenant e como identificar tenants nas requisições. O sistema precisa integrar com Keycloak para autenticação JWT, implementar controle de acesso baseado em roles (RBAC), e garantir isolamento de dados entre tenants.

## Decisão

Decidimos implementar autenticação JWT com Keycloak e roteamento baseado em tenant através de URLs estruturadas (`/api/{tenant}/...`), mantendo compatibilidade com o sistema anterior baseado em headers `X-Tenant-ID` através de rotas legacy.

## Justificativa

A decisão de usar roteamento baseado em tenant foi baseada nas seguintes razões:

- **Clareza de API**: URLs explícitas tornam a API mais clara e auto-documentada
- **Roteamento Nativo**: Aproveita capacidades nativas do roteador HTTP para isolamento
- **Auditoria Simplificada**: Logs e métricas naturalmente incluem informação de tenant
- **Cache e CDN**: Facilita implementação de cache por tenant
- **Padrão da Indústria**: Segue padrões estabelecidos por serviços como AWS, Azure, etc.
- **Validação de Tenant**: Permite validação de tenant diretamente no roteamento
- **Integração com Keycloak**: JWT tokens contêm informações de tenant para validação cruzada

## Implementação

### Estrutura de Rotas
```
# Rotas principais baseadas em tenant
/api/{tenant}/patients          # Listar pacientes do tenant
/api/{tenant}/patients/{id}     # Obter paciente específico
/api/{tenant}/encounters        # Listar encontros do tenant
/api/{tenant}/encounters/{id}   # Obter encontro específico
/api/{tenant}/practitioners     # Listar profissionais do tenant
/api/{tenant}/practitioners/{id} # Obter profissional específico
/api/{tenant}/review-request    # Criar solicitação de revisão

# Rotas legacy para compatibilidade (usam X-Tenant-ID header)
/legacy/encounters              # Listar encontros (legacy)
/legacy/patients                # Listar pacientes (legacy)
/legacy/practitioners           # Listar profissionais (legacy)
/legacy/review-request          # Criar solicitação de revisão (legacy)
```

### Autenticação JWT
- **Validação de Assinatura**: Verificação da assinatura JWT com chaves públicas do Keycloak
- **Extração de Tenant**: Tenant ID extraído do campo `preferred_username` do JWT
- **Validação Cruzada**: Tenant da URL deve corresponder ao tenant do JWT
- **Controle de Acesso**: RBAC baseado em roles definidos no Keycloak

### Middleware de Autenticação
```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Extrair token JWT do header Authorization
        // 2. Validar assinatura com chaves públicas do Keycloak
        // 3. Extrair tenant do JWT claims
        // 4. Validar tenant da URL contra tenant do JWT
        // 5. Adicionar contexto de autenticação à requisição
        // 6. Continuar para próximo handler
    })
}
```

## Alternativas Consideradas

Outras alternativas consideradas incluíram:

- **Headers Customizados**: Manter `X-Tenant-ID` header, mas resulta em APIs menos claras
- **Subdomínios**: Usar `tenant1.api.evtechallenge.com`, mas adiciona complexidade de DNS
- **Query Parameters**: Usar `?tenant=tenant1`, mas não é RESTful
- **Scopes/Collections do Couchbase**: Usar isolamento a nível de banco, mas não resolve autenticação

## Consequências

A escolha do roteamento baseado em tenant traz consigo as seguintes consequências:

- **Validação Dupla**: Necessário validar tenant tanto na URL quanto no JWT
- **Roteamento Complexo**: Middleware de roteamento mais complexo para extrair tenant
- **Migração**: Requer migração gradual de clientes existentes para nova estrutura de URLs (rotas legacy mantidas para compatibilidade)
- **Documentação**: APIs precisam ser documentadas com exemplos de tenant
- **Testes**: Testes precisam incluir diferentes cenários de tenant

## Integração com Couchbase

**Importante**: Esta implementação NÃO utiliza scopes/collections do Couchbase para isolamento de tenant. O isolamento é feito a nível de aplicação através de:

- **Filtros de Consulta**: Adicionar `WHERE tenantId = ?` nas consultas N1QL
- **Chaves de Documento**: Prefixar chaves com tenant ID quando necessário
- **Validação de Acesso**: Verificar permissões antes de acessar dados

Esta abordagem é mais simples e adequada para o caso de uso atual, onde não há necessidade de isolamento físico completo entre tenants.

## Referências

- **Keycloak healthcheck**: [Tracking instance status with health checks](https://www.keycloak.org/observability/health)
- **Couchbase Go SDK**: [Working with Collections](https://docs.couchbase.com/go-sdk/current/howtos/working-with-collections.html#2.3@go-sdk:hello-world:sample-application.adoc)
- **Couchbase Multi-tenancy**: [Scopes and Collections for Modern Multi-tenant Applications](https://www.couchbase.com/blog/scopes-and-collections-for-modern-multi-tenant-applications-couchbase-7-0/)
- **Try Couchbase Go Example**: [GitHub Repository](https://github.com/couchbaselabs/try-cb-golang/blob/7.0/main.go)
- **JWT Best Practices**: [RFC 7519 - JSON Web Token](https://tools.ietf.org/html/rfc7519)
- **Keycloak Integration**: [Keycloak Documentation](https://www.keycloak.org/documentation)
