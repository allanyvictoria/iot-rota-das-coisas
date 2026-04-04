# Rota das Coisas — Serviço de Integração IoT

Serviço de integração (broker) para dispositivos IoT desenvolvido em Go, containerizado com Docker. Permite que sensores publiquem dados de telemetria, clientes monitorem sensores em tempo real e enviem comandos para atuadores — tudo de forma desacoplada.

---

## Estrutura de Diretórios

```
.
├── docker-compose.yml
├── server/
│   ├── Dockerfile
│   ├── main.go          # Inicialização do servidor TCP/UDP
│   ├── handle.go        # Gerenciamento de conexões TCP (clientes e atuadores)
│   ├── sensor.go        # Recepção de dados UDP, worker pool, listagem de sensores
│   ├── atuador.go       # Worker e escuta de atuadores
│   ├── pubsub.go        # Sistema de publish/subscribe por tópico
│   └── protocolo.go     # Estrutura Mensagem, ParseMensagem, ToBytes
├── sensor/
│   ├── Dockerfile
│   └── main.go          # Envia telemetria UDP ao servidor
├── atuador/
│   ├── Dockerfile
│   └── main.go          # Recebe e executa comandos TCP do servidor
└── cliente/
    ├── Dockerfile
    └── main.go          # Interface de terminal para monitoramento e controle
```

---

## Pacotes e Dependências

O projeto utiliza **apenas a biblioteca padrão do Go**, sem frameworks externos, conforme exigido pelo problema:

| Pacote | Uso |
|--------|-----|
| `net` | Sockets TCP e UDP |
| `sync` | `RWMutex` para proteção de mapas compartilhados |
| `time` | Timestamps, deadlines de conexão, verificação de inatividade |
| `fmt` / `log` | Saída e logging |
| `bufio` | Leitura de input do terminal no cliente |
| `math/rand` | Geração de valores simulados no sensor |
| `strconv` | Conversão de tipos numéricos |
| `strings` | Parsing de mensagens |
| `os` | Hostname, variáveis de ambiente, encerramento |

---

## Protocolo de Comunicação

Todas as mensagens seguem o formato:

```
TIPO;ID;COMANDO;VALOR
```

| Campo | Descrição |
|-------|-----------|
| `TIPO` | Origem da mensagem: `SENSOR`, `CLIENTE`, `ATUADOR` |
| `ID` | Identificador único do dispositivo (hostname do container) |
| `COMANDO` | Ação ou dado principal (ex: `MONITORAR_SENSOR`, timestamp) |
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

**Comando do cliente para monitorar sensor:**
```
CLIENTE;a1b2c3d4;MONITORAR_SENSOR;
COMANDO;92e4c44e5955;;          ← segunda mensagem com o ID do sensor
```

**Comando para atuador:**
```
COMANDO;atuador1;on;
```

**Resposta do atuador:**
```
ATUADOR;atuador1;LIGADO;
```

### Transporte

| Componente | Protocolo | Motivo |
|------------|-----------|--------|
| Sensor → Servidor | **UDP** | Telemetria contínua — velocidade prioritária, perda ocasional aceitável |
| Cliente → Servidor | **TCP** | Comandos críticos — entrega garantida e ordenada |
| Atuador → Servidor | **TCP** | Comandos críticos — entrega garantida e ordenada |

---

##  Como Executar

### Pré-requisitos

- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)

### Subindo o ambiente completo

```bash
# Constrói as imagens e sobe server + todos os sensores e atuadores
docker compose up --build server sensor1 sensor2 atuador1 atuador2
```

### Rodando os clientes (terminais interativos)

Abra um terminal separado para cada cliente:

```bash
# Terminal 2
docker compose run --rm cliente1

# Terminal 3
docker compose run --rm cliente2
```

### Derrubando o ambiente


```bash
docker compose down -v
```

### Conectividade entre máquinas distintas

Para rodar em máquinas diferentes no laboratório, defina a variável de ambiente `SERVER_ADDR` apontando para o IP da máquina que roda o servidor:

```bash
SERVER_ADDR=192.168.1.100:1053 docker compose run --rm cliente1
```

---

## Como Usar

### Cliente

Ao conectar, o cliente recebe o menu:

```
>>>>>>>>>>>>>>>>>> MENU <<<<<<<<<<<<<<<<<<
[1] - Listar sensores disponiveis
[2] - Listar atuadores disponiveis
[3] - Controlar atuador
[4] - Monitorar sensor (tempo real)
[0] - Sair
```

**Monitorar sensor em tempo real:**
1. Digite `4` e pressione ENTER
2. Informe o ID do sensor (ex: `92e4c44e5955`)
3. Os dados serão exibidos atualizando na mesma linha
4. Pressione ENTER para parar o monitoramento

**Controlar atuador:**
1. Digite `3` e pressione ENTER
2. Informe o ID do atuador
3. Informe o comando: `on` ou `off`

### Sensor

Exibe no terminal o valor sendo enviado ao servidor em tempo real:

```
[SENSOR 92e4c44e5955] Valor enviado: 28 | Horário: 2026-04-03 23:43:37
```

### Atuador

Exibe no terminal os comandos recebidos e o status atual:

```
[ATUADOR atuador1] Status: LIGADO | Último comando: on
```

---

## Arquitetura

```
┌─────────┐  UDP  ┌──────────────────────┐  TCP  ┌──────────┐
│ Sensor1 │──────▶│                      │◀─────▶│ Cliente1 │
│ Sensor2 │──────▶│   Servidor (Broker)  │◀─────▶│ Cliente2 │
└─────────┘       │                      │◀─────▶│ Atuador1 │
                  │  - Pub/Sub por tópico│◀─────▶│ Atuador2 │
                  │  - Worker Pool UDP   │       └──────────┘
                  │  - RWMutex           │
                  └──────────────────────┘
```

O broker centraliza a comunicação, eliminando o acoplamento ponto-a-ponto entre dispositivos e aplicações.

---

## Concorrência

- `sync.RWMutex` protege todos os mapas compartilhados (`sensors`, `topicos`, `mapaAtuadores`)
- Worker pool de 10 goroutines processa pacotes UDP dos sensores com fila de 500 entradas
- Cada conexão TCP (cliente/atuador) roda em goroutine dedicada
- Fila `chan Mensagem` por atuador garante entrega ordenada de comandos

---

## Confiabilidade

- Sensores inativos por mais de 5 segundos são removidos automaticamente
- Clientes inativos por mais de 30 segundos têm a conexão encerrada (timeout)
- Desconexão do servidor é detectada e exibe mensagem ao usuário antes de encerrar
- Mensagens malformadas são descartadas com log de erro sem derrubar o servidor
