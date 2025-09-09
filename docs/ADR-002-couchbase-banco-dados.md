# Architecture Decision Record (ADR): Couchbase como Banco de Dados Primário

**Data**: 2024-01-15  
**Responsável**: Equipe de Desenvolvimento  
**Status**: Aceito  

## Contexto

Ao projetar a persistência de dados para a plataforma de dados clínicos EVT Challenge, foi necessário escolher um banco de dados adequado para armazenar recursos FHIR. Os dados FHIR possuem estrutura variável e evoluem frequentemente, requerendo flexibilidade de schema e capacidade de lidar com relacionamentos complexos entre diferentes tipos de recursos.

## Decisão

Decidimos utilizar Couchbase como banco de dados primário para persistência dos dados FHIR com isolamento multi-tenant através de scopes e collections.

## Justificativa

A decisão de usar Couchbase foi baseada nas seguintes razões:

- **Flexibilidade de Schema**: Os dados FHIR possuem estrutura variável e evoluem constantemente. Couchbase permite armazenar documentos JSON sem schema rígido
- **Desenvolvimento Rápido**: Elimina a necessidade de migrações de schema complexas ou modelagem de dados tradicional
- **Escalabilidade**: Oferece escalabilidade horizontal com sharding automático, essencial para crescimento futuro
- **Multi-Modelo**: Suporta operações chave-valor, documento e consultas N1QL, oferecendo flexibilidade de acesso aos dados
- **Performance**: Cache em memória com persistência em disco, otimizando performance de leitura
- **Multi-Tenancy**: Isolamento lógico completo através de scopes por tenant e collections por tipo de recurso
- **Consistência Eventual**: Adequado para dados clínicos onde consistência eventual é aceitável

## Alternativas Consideradas

Outras alternativas consideradas incluíram:

- **MongoDB**: Banco NoSQL similar, mas Couchbase oferece melhor performance de cache
- **PostgreSQL**: Banco relacional robusto, mas requer modelagem de schema complexa para dados FHIR variáveis

## Arquitetura de Isolamento Multi-Tenant

### Estrutura de Scopes e Collections

**DefaultScope**: Contém os dados FHIR originais ingeridos pelo fhir-client
- `encounters`: Recursos FHIR de encontros
- `patients`: Recursos FHIR de pacientes  
- `practitioners`: Recursos FHIR de profissionais
- `_default`: Status de ingestão do sistema (`template/ingestion_status`)

**Tenant Scopes**: Cada tenant possui seu próprio scope (ex: `tenant1`, `tenant2`)
- `encounters`: Cópia dos dados de encontros do DefaultScope
- `patients`: Cópia dos dados de pacientes do DefaultScope
- `practitioners`: Cópia dos dados de profissionais do DefaultScope
- `defaulty`: Status de ingestão específico do tenant (`tenant/ingestion_status`)

### Benefícios da Arquitetura

- **Isolamento Lógico**: Cada tenant possui dados completamente separados
- **Escalabilidade Automática**: Novos tenants são criados automaticamente no primeiro acesso
- **Cópia Sob Demanda**: Dados são copiados do DefaultScope apenas quando necessário
- **Revisões Integradas**: Campos de revisão (`reviewed`, `reviewTime`) são adicionados diretamente aos documentos FHIR
- **Performance**: Consultas diretas sem filtros de tenant, aproveitando índices nativos

### Processo de Criação de Tenant

1. **Primeiro Acesso**: API detecta que scope do tenant não existe
2. **Criação Automática**: Scope e collections são criados automaticamente
3. **Cópia de Dados**: Dados são copiados do DefaultScope para o tenant scope
4. **Status de Ingestão**: Flag `tenant/ingestion_status` é definida como `true` quando cópia completa
5. **Próximos Acessos**: API verifica status e serve dados diretamente do tenant scope

## Consequências

A escolha do Couchbase traz consigo as seguintes consequências:

- **Curva de Aprendizado**: N1QL traz muita proximidade com a sintaxe do SQL, KV queries já podem causar mais estranhamento
- **Complexidade Operacional**: Gerenciamento de cluster requer conhecimento específico
- **Consistência**: Menos garantias ACID comparado a bancos relacionais tradicionais
- **Custo**: Licenciamento pode ser mais caro para versão enterprise, ou o DBaaS Couchbase Capella
- **Dependência**: Maior dependência de uma tecnologia específica, dificuldade de migração
- **Isolamento de Dados**: Cada tenant possui dados completamente separados, garantindo segurança e compliance

## Referências

- Couchbase vs MongoDB: https://www.couchbase.com/content/c/cb-v-mongo-brief?x=gMxyf9
- Multi-tenancy com Couchbase: https://www.couchbase.com/blog/scopes-and-collections-for-modern-multi-tenant-applications-couchbase-7-0/
- Couchbase Documentation: https://docs.couchbase.com/
- N1QL Query Language: https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/
