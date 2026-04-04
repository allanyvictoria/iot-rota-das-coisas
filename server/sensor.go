package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

// Função para processar os dados recebidos dos sensores.
func processSensorInput(data []byte) {
	mensagem, err := ParseMensagem(data)
	if err != nil {
		log.Printf("Mensagem inválida recebida: %v", err)
		return
	}

	sensorId := mensagem.ID
	sensorType := mensagem.TIPO
	sensorValue, err := strconv.ParseInt(mensagem.VALOR, 10, 64)
	if err != nil {
		log.Printf("Erro ao converter valor do sensor %s: %v", sensorId, err)
		return
	}

	lastTime, err := time.Parse("2006-01-02 15:04:05", mensagem.COMANDO)
	if err != nil {
		log.Printf("Erro ao converter data do sensor %s: %v", sensorId, err)
		return
	}

	rwmu.Lock()
	sensors[sensorId] = Sensor{
		ID:    sensorId,
		Type:  sensorType,
		Value: sensorValue,
		Last:  lastTime,
	}
	lastSensorData = []byte(strconv.FormatInt(sensorValue, 10))
	rwmu.Unlock()

	topico := "SENSOR:" + sensorId
	msg := fmt.Sprintf("\033[2K\rSensor %s: %d", sensorId, sensorValue)
	enviarParaTopico(topico, msg)
}

// Função para receber dados dos sensores via UDP.
func receiveSensorData(conn *net.UDPConn) {
	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println(err)
			continue
		}

		data := make([]byte, n)
		copy(data, buffer[:n])

		select {
		case sensorJobs <- data:
			// enfileirado com sucesso
		default:
			log.Println("[AVISO] Fila de sensores cheia, pacote descartado")
			// telemetria: perda ocasional é aceitável conforme o problema
		}
	}
}

func verificarSensor() {
	rwmu.Lock()
	defer rwmu.Unlock()
	for sensorID, sensor := range sensors {
		duracao := time.Since(sensor.Last)
		if duracao > (5 * time.Second) {
			delete(sensors, sensorID)
			log.Printf("Sensor %s removido por inatividade", sensor.ID)
		}
	}
}

// Função para enviar a lista de sensores disponíveis para o cliente.
func sendAvailableSensors(conn net.Conn) {
	rwmu.RLock()
	defer rwmu.RUnlock()
	if len(sensors) > 0 {
		msginicial := "\n>>>>>>>> SENSORES DISPONIVEIS <<<<<<<<<<\n"
		conn.Write([]byte(msginicial))

		for _, sensor := range sensors {
			msg := fmt.Sprintf("ID: %s\nTIPO:%s\nULTIMO VALOR:%d\nULTIMA ATUALIZACAO: %s\n\n",
				sensor.ID, sensor.Type, sensor.Value, sensor.Last)

			conn.Write([]byte(msg))
		}
	} else {
		conn.Write([]byte("\n[SENSOR] Nenhum sensor disponivel.\n"))
	}
}

// Função para enviar dados para o cliente.
func sendDataToClient(conn net.Conn, data []byte) {
	_, err := conn.Write(data)
	if err != nil {
		log.Println("Erro ao enviar dados para o cliente:", err)
	}
}

func startSensorWorkerPool(n int) {
	for i := 0; i < n; i++ {
		go func() {
			for data := range sensorJobs {
				processSensorInput(data)
			}
		}()
	}
}
