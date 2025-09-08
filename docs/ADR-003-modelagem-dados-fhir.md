# Architecture Decision Record (ADR): Modelagem de Dados FHIR com Desnormalização

**Data**: 2024-01-15  
**Responsável**: Equipe de Desenvolvimento  
**Status**: Aceito  

## Contexto

Ao projetar a modelagem de dados para a plataforma de dados clínicos EVT Challenge, foi necessário decidir como estruturar os recursos FHIR no banco de dados Couchbase. Os dados FHIR possuem relacionamentos complexos entre diferentes tipos de recursos (Encounters, Patients, Practitioners), e é necessário otimizar para consultas eficientes e suporte a multi-tenancy.

## Decisão

Decidimos implementar uma modelagem desnormalizada dos dados FHIR, organizando em três tipos principais de documentos: Encounters, Patients e Practitioners, com campos desnormalizados para relacionamentos e suporte a multi-tenancy através de documentos de revisão separados.

## Justificativa

A decisão de usar modelagem desnormalizada foi baseada nas seguintes razões:

- **Performance de Consulta**: Elimina necessidade de joins complexos, otimizando tempo de resposta
- **Acesso Direto**: Permite acesso direto a IDs relacionados sem consultas adicionais
- **Estrutura FHIR Original**: Mantém a estrutura original dos recursos FHIR para compatibilidade
- **Suporte Multi-Modelo**: Funciona tanto com operações chave-valor quanto consultas N1QL
- **Isolamento Multi-Tenant**: Separação clara entre dados compartilhados (FHIR) e específicos por tenant (revisões)
- **Flexibilidade**: Facilita adição de novos tenants sem modificações de schema

## Estrutura de Dados

### Documentos de Encontro
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

### Documentos de Paciente/Profissional
```json
{
  "id": "patient-456",
  "resourceType": "Patient",
  "docId": "Patient/patient-456",
  // ... dados FHIR originais
}
```

### Documentos de Revisão Multi-Tenant
```json
{
  "tenantId": "tenant-abc",
  "encounters": {
    "Encounter/encounter-123": {
      "reviewRequested": true,
      "reviewTime": "2024-01-15T10:30:00Z",
      "entityType": "Encounter",
      "entityID": "encounter-123"
    }
  },
  "patients": {
    "Patient/patient-456": {
      "reviewRequested": true,
      "reviewTime": "2024-01-15T10:30:00Z",
      "entityType": "Patient",
      "entityID": "patient-456"
    }
  },
  "practitioners": {
    "Practitioner/practitioner-789": {
      "reviewRequested": true,
      "reviewTime": "2024-01-15T10:30:00Z",
      "entityType": "Practitioner",
      "entityID": "practitioner-789"
    }
  },
  "updated": "2024-01-15T10:30:00Z"
}
```

## Alternativas Consideradas

Outras alternativas consideradas incluíram:

- **Modelagem Normalizada**: Separar relacionamentos em documentos distintos, mas resultaria em consultas complexas e performance degradada
- **Embedded Documents**: Aninhar todos os dados relacionados, mas resultaria em documentos muito grandes e duplicação de dados
- **Hybrid Approach**: Combinação de normalização e desnormalização, mas aumentaria complexidade desnecessariamente

## Consequências

A escolha da modelagem desnormalizada traz consigo as seguintes consequências:

- **Duplicação de Dados**: IDs relacionados são armazenados em múltiplos lugares
- **Sincronização**: Necessário manter consistência entre campos desnormalizados e dados originais
- **Tamanho de Documentos**: Documentos podem ser maiores devido aos campos adicionais
- **Complexidade de Atualização**: Atualizações podem requerer modificações em múltiplos documentos
- **Flexibilidade de Consulta**: Facilita consultas complexas sem joins

## Referências

- FHIR Resource References: http://hl7.org/fhir/R4/documentreference.html
- Couchbase Document Modeling: https://docs.couchbase.com/server/current/learn/data/document-data-model.html
- Multi-tenant Data Architecture: https://www.couchbase.com/blog/scopes-and-collections-for-modern-multi-tenant-applications-couchbase-7-0/
- Database Denormalization Patterns: https://martinfowler.com/articles/schemaless/
