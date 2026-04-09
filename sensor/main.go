package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

// Função para obter o endereço do servidor
func getServerAddr() string {
	// Tenta obter o endereço do servidor a partir da variável de ambiente
	addr := os.Getenv("SERVER_ADDR")
	// Se não estiver definida, usa o nome do serviço no Docker Compose
	if addr == "" {
		addr = "server:1053"
	}
	fmt.Println("Endereço do servidor:", addr)
	return addr
}

func main() {
	// Obtém o hostname do container para usar como ID do sensor
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	addr := getServerAddr()

	// Resolve o endereço UDP do servidor
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatal("Erro ao resolver endereço:", err)
	}

	// Cria uma conexão UDP para enviar dados dos sensores
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Loop principal do sensor, enviando dados periodicamente
	for {

		id := hostname
		tipo := "SENSOR"
		temp := rand.Intn(40)
		last := time.Now().UTC().Format("2006-01-02 15:04:05")
		mensagem := fmt.Sprintf("%s;%s;%s;%d", tipo, id, last, temp)
		_, err := conn.Write([]byte(mensagem))
		if err != nil {
			log.Fatal(err)
		}

		// Exibe o valor enviado e o horário no console do sensor
		fmt.Printf("\r\033[2K\r[SENSOR %s] Valor enviado: %d | Horário: %s", id, temp, last)

		time.Sleep(1 * time.Millisecond)
	}

}
