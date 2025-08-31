# Architecture Decision Record (ADR): Zerolog com Elasticsearch para Observabilidade

**Data**: 2024-01-15  
**Responsável**: Equipe de Desenvolvimento  
**Status**: Aceito  

## Contexto

Ao projetar a estratégia de observabilidade para a plataforma de dados clínicos EVT Challenge, foi necessário escolher uma solução de logging que oferecesse estruturação adequada, integração com sistemas de monitoramento, e compatibilidade com padrões de observabilidade modernos. O sistema precisa lidar com logs de múltiplos microsserviços e fornecer visibilidade operacional abrangente.

## Decisão

Decidimos utilizar Zerolog com formatação JSON e integração com Elasticsearch para logging estruturado e observabilidade centralizada.

## Justificativa

A decisão de usar Zerolog com Elasticsearch foi baseada nas seguintes razões:

- **Logging Estruturado**: Zerolog produz logs JSON estruturados, facilitando análise e parsing automatizado
- **Performance**: Zerolog é otimizado para performance, com overhead mínimo comparado a outras bibliotecas de logging
- **Compatibilidade ECS**: Suporte nativo ao Elastic Common Schema (ECS), padronizando campos de log
- **Integração Elasticsearch**: Logs JSON são facilmente ingeridos pelo Elasticsearch para análise e visualização
- **Correlação de Logs**: Facilita correlação de logs entre microsserviços através de campos estruturados
- **Sem Exposição de PHI**: Logs estruturados permitem controle granular sobre quais dados são logados
- **Observabilidade Centralizada**: Elasticsearch oferece capacidades avançadas de busca, agregação e visualização

## Implementação

### Configuração Zerolog
```go
logger := ecszerolog.New(os.Stdout)
log.Logger = logger

// Uso com campos estruturados
log.Info().
    Str("tenant_id", tenantID).
    Str("resource_type", "Encounter").
    Str("resource_id", resourceID).
    Msg("Resource processed successfully")
```

### Integração Elasticsearch
- Logs JSON são enviados para Elasticsearch via Filebeat
- Configuração de parsers NDJSON para processamento automático
- Mapeamento de campos ECS para padronização

## Alternativas Consideradas

Outras alternativas consideradas incluíram:

- **Zap**: Biblioteca de logging estruturado, mas Zerolog oferece melhor integração ECS
- **Logging Simples**: Logs em texto plano, mas dificulta análise e correlação

## Consequências

A escolha do Zerolog com Elasticsearch traz consigo as seguintes consequências:

- **Dependência de Infraestrutura**: Requer setup e manutenção do Elasticsearch
- **Curva de Aprendizado**: Equipe precisa aprender configuração e uso do Elasticsearch
- **Complexidade de Setup**: Configuração inicial requer conhecimento de Filebeat e Elasticsearch

## Referências

- Zerolog ECS Integration: https://www.elastic.co/docs/reference/ecs/logging/go-zerolog/setup
- Elastic Common Schema: https://www.elastic.co/guide/en/ecs/current/index.html
- Zerolog Documentation: https://github.com/rs/zerolog
- Elasticsearch Logging Best Practices (TODO: log level do ECS): https://www.elastic.co/guide/en/elasticsearch/reference/current/logging.html
