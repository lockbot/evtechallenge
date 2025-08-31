# Architecture Decision Records (ADRs)

Este diretório contém os Architecture Decision Records (ADRs) da plataforma EVT Challenge - Clinical Data Platform.

## O que são ADRs?

Architecture Decision Records são documentos que capturam decisões arquiteturais importantes tomadas durante o desenvolvimento do projeto, incluindo o contexto, as alternativas consideradas, e as consequências de cada decisão.

## ADRs Disponíveis

### [ADR-001: Separação de Microsserviços](./ADR-001-microsservicos-separacao.md)
**Decisão**: Separar ingestão de dados (fhir-client) e API REST (api-rest) em containers distintos.

### [ADR-002: Couchbase como Banco de Dados Primário](./ADR-002-couchbase-banco-dados.md)
**Decisão**: Utilizar Couchbase para persistência de dados FHIR e informações multi-tenant.

### [ADR-003: Modelagem de Dados FHIR com Desnormalização](./ADR-003-modelagem-dados-fhir.md)
**Decisão**: Implementar modelagem desnormalizada para otimizar performance de consultas.

### [ADR-004: Zerolog com Elasticsearch para Observabilidade](./ADR-004-zerolog-elasticsearch.md)
**Decisão**: Utilizar Zerolog com formatação JSON e integração Elasticsearch para logging estruturado.

## Como Usar

Cada ADR segue o formato padrão:
- **Contexto**: Situação que levou à decisão
- **Decisão**: A decisão tomada
- **Justificativa**: Razões para a decisão
- **Alternativas Consideradas**: Outras opções avaliadas
- **Consequências**: Impactos da decisão
- **Referências**: Links e documentação relevante

## Contribuindo

Ao tomar novas decisões arquiteturais importantes:
1. Crie um novo ADR seguindo o template
2. Numere sequencialmente (ADR-005, ADR-006, etc.)
3. Documente todas as seções obrigatórias
4. Inclua referências relevantes
5. Atualize este README com o novo ADR
