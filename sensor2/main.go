package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

func getServerAddr() string {
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = "server:1053" // Aqui "server" é o nome do serviço no Docker Compose
	}
	fmt.Println("Endereço do servidor:", addr)
	return addr
}

func main() {

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	addr := getServerAddr()

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatal("Erro ao resolver endereço:", err)
	}

	// Create a UDP connection to the server
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

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

		// Interface terminal do sensor
		fmt.Printf("\033[2K\r[SENSOR %s] Valor enviado: %d | Horário: %s", id, temp, last)

		time.Sleep(1 * time.Millisecond)
	}

}
