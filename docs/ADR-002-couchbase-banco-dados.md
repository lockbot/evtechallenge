# Architecture Decision Record (ADR): Couchbase como Banco de Dados Primário

**Data**: 2024-01-15  
**Responsável**: Equipe de Desenvolvimento  
**Status**: Aceito  

## Contexto

Ao projetar a persistência de dados para a plataforma de dados clínicos EVT Challenge, foi necessário escolher um banco de dados adequado para armazenar recursos FHIR. Os dados FHIR possuem estrutura variável e evoluem frequentemente, requerendo flexibilidade de schema e capacidade de lidar com relacionamentos complexos entre diferentes tipos de recursos.

## Decisão

Decidimos utilizar Couchbase como banco de dados primário para persistência dos dados FHIR e informações de revisão multi-tenant.

## Justificativa

A decisão de usar Couchbase foi baseada nas seguintes razões:

- **Flexibilidade de Schema**: Os dados FHIR possuem estrutura variável e evoluem constantemente. Couchbase permite armazenar documentos JSON sem schema rígido
- **Desenvolvimento Rápido**: Elimina a necessidade de migrações de schema complexas ou modelagem de dados tradicional
- **Escalabilidade**: Oferece escalabilidade horizontal com sharding automático, essencial para crescimento futuro
- **Multi-Modelo**: Suporta operações chave-valor, documento e consultas N1QL, oferecendo flexibilidade de acesso aos dados
- **Performance**: Cache em memória com persistência em disco, otimizando performance de leitura
- **Multi-Tenancy**: Suporte nativo a isolamento de dados através de scopes e collections
- **Consistência Eventual**: Adequado para dados clínicos onde consistência eventual é aceitável

## Alternativas Consideradas

Outras alternativas consideradas incluíram:

- **MongoDB**: Banco NoSQL similar, mas Couchbase oferece melhor performance de cache
- **PostgreSQL**: Banco relacional robusto, mas requer modelagem de schema complexa para dados FHIR variáveis

## Consequências

A escolha do Couchbase traz consigo as seguintes consequências:

- **Curva de Aprendizado**: N1QL traz muita proximidade com a sintaxe do SQL, KV queries já podem causar mais estranhamento
- **Complexidade Operacional**: Gerenciamento de cluster requer conhecimento específico
- **Consistência**: Menos garantias ACID comparado a bancos relacionais tradicionais
- **Custo**: Licenciamento pode ser mais caro para versão enterprise, ou o DBaaS Couchbase Capella
- **Dependência**: Maior dependência de uma tecnologia específica, dificuldade de migração

## Referências

- Couchbase vs MongoDB: https://www.couchbase.com/content/c/cb-v-mongo-brief?x=gMxyf9
- Multi-tenancy com Couchbase: https://www.couchbase.com/blog/scopes-and-collections-for-modern-multi-tenant-applications-couchbase-7-0/
- Couchbase Documentation: https://docs.couchbase.com/
- N1QL Query Language: https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/
