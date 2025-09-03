# ADR-001: Autenticação com Keycloak

## Status
Aceito

## Data
2024-09-03

## Contexto
A aplicação precisa de um sistema de autenticação robusto para gerenciar usuários B2B (Business-to-Business) com diferentes níveis de acesso. O sistema deve ser capaz de:

- Autenticar usuários de diferentes organizações (tenants)
- Gerenciar permissões por tenant
- Fornecer tokens JWT seguros
- Integrar com a arquitetura multi-tenant existente

## Decisão
Implementar autenticação usando **Keycloak** como provedor de identidade e gerenciamento de acesso (IAM).

## Consequências

### Positivas
- **Segurança robusta**: Keycloak é uma solução enterprise-grade com recursos avançados de segurança
- **Padrões abertos**: Suporte a OAuth 2.0, OpenID Connect e SAML
- **Escalabilidade**: Pode lidar com milhares de usuários e tenants
- **Flexibilidade**: Configurável para diferentes cenários de uso
- **Integração**: APIs REST bem documentadas para automação

### Negativas
- **Complexidade**: Requer configuração inicial e manutenção
- **Recursos**: Consome mais memória e CPU que soluções mais simples
- **Curva de aprendizado**: Equipe precisa entender conceitos de IAM

## Implementação

### Componentes
1. **Keycloak Server**: Container Docker rodando na porta 8082
2. **Realm**: Configurado para o domínio da aplicação
3. **Client**: Aplicação REST configurada no Keycloak
4. **Users**: Usuários B2B organizados por grupos de tenant
5. **Groups**: Estrutura hierárquica para organizações

### Configuração
- **Porta**: 8082 (evitando conflitos com API e Prometheus)
- **Admin**: Usuário e senha configuráveis via variáveis de ambiente
- **Setup**: Script automatizado para configuração inicial
- **Integração**: Middleware de autenticação na API REST

### Variáveis de Ambiente
```bash
KEYCLOAK_URL=http://localhost:8082
KEYCLOAK_REALM=evtechallenge
KEYCLOAK_CLIENT_ID=api-rest
KEYCLOAK_CLIENT_SECRET=<secret>
KEYCLOAK_ADMIN_USERNAME=admin
KEYCLOAK_ADMIN_PASSWORD=<password>
```

## Alternativas Consideradas

### 1. Autenticação Simples (JWT manual)
- **Prós**: Implementação simples, controle total
- **Contras**: Menos seguro, sem gerenciamento de usuários, difícil de escalar

### 2. Auth0
- **Prós**: SaaS gerenciado, fácil de usar
- **Contras**: Custo por usuário, dependência externa, menos controle

### 3. Firebase Auth
- **Prós**: Integração fácil com Google, documentação excelente
- **Contras**: Vendor lock-in, menos flexível para B2B

## Referências
- [Keycloak Documentation](https://www.keycloak.org/documentation)
- [OAuth 2.0 Specification](https://tools.ietf.org/html/rfc6749)
- [OpenID Connect Specification](https://openid.net/connect/)

## Revisão
Este ADR deve ser revisado quando:
- Houver mudanças significativas nos requisitos de segurança
- Novas alternativas de autenticação se tornarem disponíveis
- A equipe identificar problemas de performance ou manutenção
