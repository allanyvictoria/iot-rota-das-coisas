# Rota das Coisas — Serviço de Integração IoT

Serviço de integração (broker) para dispositivos IoT desenvolvido em Go, containerizado com Docker. Permite que sensores publiquem dados de telemetria, clientes monitorem sensores em tempo real e enviem comandos para atuadores — tudo de forma desacoplada.

---

## Estrutura de Diretórios

```
.
├── docker-compose.yml
├── go.mod
├── server/
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go        # Inicialização do servidor TCP e UDP, variáveis globais
│   ├── handlers.go    # Gerenciamento de conexões TCP (clientes e atuadores)
│   ├── sensor.go      # Recepção de dados UDP, worker pool, listagem de sensores
│   ├── atuador.go     # Struct Atuador, worker dedicado, listagem de atuadores
│   ├── pubsub.go      # Sistema de publish/subscribe por tópico (subscribe, unsubscribe, enviarParaTopico)
│   └── protocolo.go   # Struct Mensagem, ParseMensagem, ToBytes
├── sensor/
│   ├── Dockerfile
│   ├── go.mod
│   └── main.go        # Envia telemetria UDP ao servidor periodicamente
├── atuador/
│   ├── Dockerfile
│   ├── go.mod
│   └── main.go        # Conecta via TCP, recebe e executa comandos (on/off)
└── cliente/
    ├── Dockerfile
    ├── go.mod
    └── main.go        # Interface de terminal para monitoramento e controle
```

---

## Pacotes e Dependências

O projeto utiliza **apenas a biblioteca padrão do Go** (Go 1.22), sem dependências externas:

| Pacote | Uso |
|--------|-----|
| `net` | Sockets TCP (`net.Listen`, `net.Dial`) e UDP (`net.ListenUDP`, `net.DialUDP`) |
| `sync` | `sync.RWMutex` para proteção dos mapas globais |
| `time` | Timestamps, deadlines de conexão, verificação de inatividade dos sensores |
| `fmt` | Formatação de strings e saída no terminal |
| `log` | Logging de eventos e erros no servidor |
| `bufio` | Leitura linha a linha do stdin no cliente |
| `math/rand` | Geração de valores simulados no sensor |
| `strconv` | Conversão de tipos numéricos (valor do sensor) |
| `strings` | Parsing e limpeza das mensagens (`Split`, `TrimSpace`) |
| `os` | Hostname do container, variáveis de ambiente, encerramento do processo |

Cada serviço (`server`, `sensor`, `atuador`, `cliente`) possui seu próprio `go.mod` independente, pois são compilados separadamente dentro de seus respectivos containers Docker.

---

## Protocolo de Comunicação

Todas as mensagens seguem o formato de texto simples delimitado por ponto-e-vírgula:

```
TIPO;ID;COMANDO;VALOR
```

| Campo | Descrição |
|-------|-----------|
| `TIPO` | Origem: `SENSOR`, `CLIENTE`, `ATUADOR` ou `COMANDO` |
| `ID` | Identificador único do dispositivo (hostname do container) |
| `COMANDO` | Ação ou dado principal (ex: `LISTAR`, `ATUAR`, timestamp do sensor) |
| `VALOR` | Dado adicional (ex: leitura numérica do sensor) |

### Exemplos de mensagens

**Handshake inicial do cliente:**
```
CLIENTE;a1b2c3d4;INICIAL;
```

**Telemetria do sensor (UDP):**
```
SENSOR;92e4c44e5955;2026-04-03 23:43:37;28
```

**Cliente solicita monitorar sensor:**
```
CLIENTE;a1b2c3d4;MONITORAR_SENSOR;
COMANDO;92e4c44e5955;;          ← segunda mensagem com o ID do sensor desejado
```

**Parar monitoramento:**
```
;;PARAR;
```

**Comando para atuador:**
```
COMANDO;atuador1;on;
```

**Resposta do atuador:**
```
ATUADOR;atuador1;Atuador LIGADO;
```

### Transporte

| Componente | Protocolo | Motivo |
|------------|-----------|--------|
| Sensor → Servidor | **UDP** | Telemetria contínua — velocidade prioritária, perda ocasional aceitável |
| Cliente → Servidor | **TCP** | Comandos críticos — entrega garantida e ordenada |
| Atuador → Servidor | **TCP** | Respostas críticas — entrega garantida e ordenada |

O servidor escuta na **porta 1053** para ambos os protocolos simultaneamente.

---

## Como Executar

### Pré-requisitos

- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)

### Subindo o ambiente

O `docker-compose.yml` define um serviço de cada tipo (`server`, `sensor`, `atuador`, `cliente`). Para subir o servidor, o sensor e o atuador:

```bash
docker compose up --build server sensor atuador
```

### Rodando o cliente (terminal interativo)

Abra um terminal separado:

```bash
docker compose run --rm cliente
```

Para rodar múltiplos sensores, atuadores ou clientes simultaneamente, use `--scale`:

```bash
docker compose up --build --scale sensor=2 --scale atuador=2 server sensor atuador
```

### Derrubando o ambiente

```bash
docker compose down -v
```

### Conectividade entre máquinas distintas

Para rodar em máquinas diferentes (ex: servidor em outro computador no laboratório), defina a variável de ambiente `SERVER_ADDR` apontando para o IP e porta do servidor:

```bash
SERVER_ADDR=192.168.1.100:1053 docker compose run --rm cliente
```

A mesma variável funciona para `sensor` e `atuador`.

---

## Como Usar

### Cliente

Ao conectar, o cliente recebe o menu do servidor e aguarda o primeiro envio antes de aceitar entradas:

```
>>>>>>>>>>>>>>>>>> MENU <<<<<<<<<<<<<<<<<<
[1] - Listar sensores disponiveis
[2] - Listar atuadores disponiveis
[3] - Controlar atuador
[4] - Monitorar sensor (tempo real)
[0] - Sair
```

**Listar sensores disponíveis (`1`):** exibe ID, tipo, último valor e última atualização de cada sensor ativo.

**Listar atuadores disponíveis (`2`):** exibe ID e status atual (`LIGADO`/`DESLIGADO`) de cada atuador conectado.

**Controlar atuador (`3`):**
1. Digite `3` e pressione ENTER
2. Informe o ID do atuador
3. Informe o comando: `on` ou `off`
4. O servidor aguarda a resposta do atuador por até 5 segundos e exibe o resultado

**Monitorar sensor em tempo real (`4`):**
1. Digite `4` e pressione ENTER
2. Informe o ID do sensor (obtido via opção `1`)
3. Os dados são exibidos atualizando na mesma linha via escape ANSI (`\033[2K\r`)
4. Pressione ENTER para parar o monitoramento e voltar ao menu

**Sair (`0`):** encerra a conexão TCP com o servidor.

### Sensor

Fica em loop contínuo enviando pacotes UDP a cada 1 ms com um valor inteiro aleatório entre 0 e 39. Exibe no terminal:

```
[SENSOR 92e4c44e5955] Valor enviado: 28 | Horário: 2026-04-03 23:43:37
```

### Atuador

Conecta ao servidor via TCP, registra-se com status `DESLIGADO` e aguarda comandos. Ao receber um comando, atualiza o estado interno e envia a resposta de volta ao servidor. Exibe no terminal:

```
Atuador LIGADO
```

---

## Arquitetura

```
┌─────────┐  UDP  ┌──────────────────────────┐  TCP  ┌──────────┐
│  Sensor │──────▶│                          │◀─────▶│  Cliente │
└─────────┘       │   Servidor (Broker)      │       └──────────┘
                  │                          │
                  │  - Worker Pool UDP (x10) │  TCP  ┌──────────┐
                  │  - Pub/Sub por tópico    │◀─────▶│  Atuador │
                  │  - RWMutex global        │       └──────────┘
                  │  - Fila por atuador      │
                  └──────────────────────────┘
```

O broker centraliza toda a comunicação, eliminando o acoplamento direto entre dispositivos e clientes.

---

## Concorrência

- `sync.RWMutex` (`rwmu`) protege todos os mapas globais compartilhados: `sensors`, `topicos`, `clienteTopico` e `mapaAtuadores`
- Worker pool de **10 goroutines** (`startSensorWorkerPool`) processa pacotes UDP dos sensores a partir de um canal com buffer de 500 entradas (`sensorJobs`)
- Cada conexão TCP aceita (cliente ou atuador) roda em uma **goroutine dedicada**
- Cada atuador possui um **canal individual** (`Fila chan Mensagem`) com buffer de 10 entradas, garantindo entrega ordenada de comandos e associando respostas ao cliente correto via `chan string`

---

## Confiabilidade

- Sensores inativos por mais de **5 segundos** (sem pacote UDP) são removidos automaticamente do mapa
- Clientes inativos por mais de **30 segundos** (sem enviar comando) têm a conexão encerrada por timeout (`SetReadDeadline`)
- Desconexão do servidor é detectada pelo cliente e pelo atuador, que exibem mensagem e encerram o processo
- Pacotes UDP com fila cheia são descartados com log de aviso, sem bloquear o receptor
- Mensagens malformadas (menos de 4 campos separados por `;`) são descartadas com log de erro, sem derrubar o servidor
- Timeout de **5 segundos** aguardando resposta do atuador; se não responder, o servidor notifica o cliente
