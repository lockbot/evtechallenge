# Architecture Decision Record (ADR): Separação de Microsserviços para Plataforma de Dados Clínicos

**Data**: 2024-01-15  
**Responsável**: Equipe de Desenvolvimento  
**Status**: Aceito  

## Contexto

Ao projetar a arquitetura para a plataforma de dados clínicos EVT Challenge, foi necessário tomar uma decisão sobre como estruturar os componentes de ingestão de dados FHIR e a API REST. O sistema precisa lidar com ingestão de dados de APIs públicas FHIR, persistência em banco de dados NoSQL, e fornecimento de uma API multi-tenant para acesso aos dados clínicos.

## Decisão

Decidimos separar a ingestão de dados (fhir-client) e a API REST (api-rest) em dois containers/projetos distintos, implementando uma arquitetura de microsserviços.

## Justificativa

A decisão de usar microsserviços separados foi baseada nas seguintes razões:

- **Escalabilidade Independente**: Permite escalar a ingestão de dados e a API REST separadamente baseado na carga de trabalho específica de cada componente
- **Isolamento de Falhas**: Falhas na API não afetam o processo de ingestão de dados, e vice-versa
- **Flexibilidade de Deploy**: Possibilita fazer deploy de atualizações independentemente em cada serviço
- **Otimização de Recursos**: Cada serviço pode ter requisitos de recursos (CPU, memória) otimizados para sua função específica
- **Agendamento de Tarefas**: Facilita a implementação de rotinas de ingestão agendadas (diárias/semanais) sem impactar a disponibilidade da API
- **Múltiplas Fontes de Dados**: Permite adicionar novas fontes de ingestão sem modificar a API existente

## Alternativas Consideradas

Outras alternativas consideradas incluíram:

- **Monolito**: Desenvolver tudo como um único aplicativo. Embora seja mais simples inicialmente, resultaria em dificuldades de escalabilidade e manutenção conforme o sistema crescesse
- **Arquitetura de Camadas**: Organizar o sistema em camadas distintas dentro de um único container. No entanto, isso resultaria em acoplamento excessivo e dificultaria a evolução independente dos componentes

## Consequências

A escolha de microsserviços separados traz consigo as seguintes consequências:

- **Complexidade de Gestão**: Gerenciar múltiplos containers requer configuração adequada de orquestração (Docker Compose)
- **Comunicação entre Serviços**: Necessário estabelecer padrões de comunicação via banco de dados compartilhado (Couchbase)
- **Overhead Inicial**: A configuração inicial dos microsserviços requer mais tempo de setup comparado a uma abordagem monolítica
- **Monitoramento**: Necessário implementar observabilidade distribuída para rastrear ambos os serviços

## Referências

- Martin Fowler - Microsserviços: https://martinfowler.com/articles/microservices.html
- NGINX - O que são Microsserviços: https://www.nginx.com/blog/introduction-to-microservices/
- Docker Compose Documentation: https://docs.docker.com/compose/
