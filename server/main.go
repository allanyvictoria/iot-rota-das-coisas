package main

import (
	"log"
	"net"
	"sync"
	"time"
)

type Sensor struct {
	ID    string
	Type  string
	Value int64
	Last  time.Time
}

var mapaAtuadores = make(map[string]*Atuador)      // mapa de ID de atuadores para suas conexões e status
var sensors = make(map[string]Sensor)              // mapa de ID de sensores para seus dados mais recentes
var lastSensorData []byte                          // dados do sensor mais recentes
var topicos = make(map[string]map[string]net.Conn) // mapa de tópicos para clientes inscritos (tópico -> ID do cliente -> conexão)
var clienteTopico = make(map[string]string)        // mapa de ID de cliente para tópico inscrito
var clientCount int                                // contador de clientes conectados
var rwmu sync.RWMutex                              // mutex para proteger o acesso a mapas e contador de clientes
var sensorJobs = make(chan []byte, 500)            // buffer de 500 pacotes para os sensores

// O servidor UDP escuta na porta 1053 para receber dados dos sensores e também aceita conexões TCP para enviar dados aos clientes
func main() {
	log.Println("[Servidor UDP]: iniciado")
	addr, err := net.ResolveUDPAddr("udp", ":1053")
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	startSensorWorkerPool(10) // 10 workers paralelos

	// verificação periodica dos sensores
	go func() {
		for {
			verificarSensor()
			time.Sleep(1 * time.Second)
		}

	}()

	log.Println("[Servidor TCP]: iniciado")
	listener, err := net.Listen("tcp", ":1053")
	if err != nil {
		log.Fatal(err)
	}

	go receiveSensorData(conn)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Erro ao aceitar conexão:", err)
			continue
		}
		go handleConnection(conn)
	}
}
