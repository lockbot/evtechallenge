# Architecture Decision Record (ADR): Autenticação JWT e Roteamento Baseado em Tenant

**Data**: 2024-01-15  
**Responsável**: Equipe de Desenvolvimento  
**Status**: Aceito  

## Contexto

Ao implementar autenticação e autorização para a plataforma de dados clínicos EVT Challenge, foi necessário decidir sobre a estratégia de autenticação multi-tenant e como identificar tenants nas requisições. O sistema precisa integrar com Keycloak para autenticação JWT, implementar controle de acesso baseado em roles (RBAC), e garantir isolamento de dados entre tenants.

## Decisão

Decidimos implementar autenticação JWT com Keycloak e roteamento baseado em tenant através de URLs estruturadas (`/api/{tenant}/...`).

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
- **Documentação**: APIs precisam ser documentadas com exemplos de tenant
- **Testes**: Testes precisam incluir diferentes cenários de tenant

## Integração com Couchbase

**Arquitetura de Isolamento**: Esta implementação utiliza scopes e collections do Couchbase para isolamento completo de tenant:

### Estrutura de Isolamento

- **Scopes por Tenant**: Cada tenant possui seu próprio scope (ex: `tenant1`, `tenant2`)
- **Collections por Tipo**: Cada scope contém collections para `encounters`, `patients`, `practitioners` e `defaulty`
- **Criação Automática**: Scopes e collections são criados automaticamente no primeiro acesso do tenant
- **Cópia de Dados**: Dados são copiados do DefaultScope para o tenant scope sob demanda

### Benefícios da Arquitetura

- **Isolamento Físico**: Dados de cada tenant são completamente separados
- **Performance**: Consultas diretas sem filtros de tenant, aproveitando índices nativos
- **Escalabilidade**: Novos tenants são criados automaticamente sem impacto em tenants existentes
- **Segurança**: Impossível acesso acidental a dados de outros tenants
- **Compliance**: Isolamento completo atende requisitos de compliance e auditoria

### Processo de Acesso

1. **Validação JWT**: Token é validado e tenant extraído
2. **Verificação de Scope**: API verifica se scope do tenant existe
3. **Criação Automática**: Se não existe, scope e collections são criados
4. **Cópia de Dados**: Dados são copiados do DefaultScope (apenas na primeira vez)
5. **Acesso Direto**: Consultas são feitas diretamente no tenant scope

## Referências

- **Keycloak healthcheck**: [Tracking instance status with health checks](https://www.keycloak.org/observability/health)
- **Couchbase Go SDK**: [Working with Collections](https://docs.couchbase.com/go-sdk/current/howtos/working-with-collections.html#2.3@go-sdk:hello-world:sample-application.adoc)
- **Couchbase Multi-tenancy**: [Scopes and Collections for Modern Multi-tenant Applications](https://www.couchbase.com/blog/scopes-and-collections-for-modern-multi-tenant-applications-couchbase-7-0/)
- **Try Couchbase Go Example**: [GitHub Repository](https://github.com/couchbaselabs/try-cb-golang/blob/7.0/main.go)
- **JWT Best Practices**: [RFC 7519 - JSON Web Token](https://tools.ietf.org/html/rfc7519)
- **Keycloak Integration**: [Keycloak Documentation](https://www.keycloak.org/documentation)
