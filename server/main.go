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

var mapaAtuadores = make(map[string]*Atuador)
var sensors = make(map[string]Sensor)
var lastSensorData []byte
var topicos = make(map[string]map[string]net.Conn)
var clienteTopico = make(map[string]string)
var clientCount int
var rwmu sync.RWMutex
var sensorJobs = make(chan []byte, 500) // buffer de 500 pacotes

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
