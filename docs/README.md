# Architecture Decision Records (ADRs)

Este diretório contém os Architecture Decision Records (ADRs) da plataforma EVT Challenge - Clinical Data Platform.

## ADRs Disponíveis

### [ADR-001: Separação de Microsserviços](./ADR-001-microsservicos-separacao.md)
**Decisão**: Separar ingestão de dados (fhir-client) e API REST (api-rest) em containers distintos.

### [ADR-002: Couchbase como Banco de Dados Primário](./ADR-002-couchbase-banco-dados.md)
**Decisão**: Utilizar Couchbase para persistência de dados FHIR e informações multi-tenant.

### [ADR-003: Modelagem de Dados FHIR com Desnormalização](./ADR-003-modelagem-dados-fhir.md)
**Decisão**: Implementar modelagem desnormalizada para otimizar performance de consultas.

### [ADR-004: Zerolog com Elasticsearch para Observabilidade](./ADR-004-zerolog-elasticsearch.md)
**Decisão**: Utilizar Zerolog com formatação JSON e integração Elasticsearch para logging estruturado.

### [ADR-005: Autenticação JWT e Roteamento Baseado em Tenant](./ADR-005-autenticacao-roteamento-tenant.md)

**Decisão**: Implementar autenticação JWT com Keycloak e roteamento baseado em tenant através de URLs estruturadas.

## Contribuindo

Ao tomar novas decisões arquiteturais importantes:
1. Crie um novo ADR seguindo o template
2. Numere sequencialmente (ADR-005, ADR-006, etc.)
3. Documente todas as seções obrigatórias
4. Inclua referências relevantes
5. Atualize este README com o novo ADR
