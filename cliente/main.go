package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func getServerAddr() string {
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		// No Docker, use o nome do serviço (por exemplo, "server:1053")
		addr = "server:1053" // Aqui "server" é o nome do serviço no Docker Compose
	}
	fmt.Println("Endereço do servidor:", addr)
	return addr
}

func main() {
	done := make(chan bool)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.Dial("tcp", getServerAddr())
	if err != nil {
		log.Fatal("Erro ao conectar ao servidor:", err)
	}
	defer conn.Close()

	id := hostname
	tipo := "CLIENTE"
	comando := "INICIAL"
	dados := fmt.Sprintf("%s;%s;%s;%s", tipo, id, comando, "")
	_, err = conn.Write([]byte(dados))
	if err != nil {
		log.Fatal("Erro ao enviar dados para o servidor:", err)
	}

	leitura := bufio.NewReader(os.Stdin)

	go func() {
		<-done
		os.Exit(0) // encerra mesmo que o loop principal esteja bloqueado em ReadString
	}()

	go lerServer(conn, done)

	<-menuPronto // espera o menu ser recebido antes de aceitar comandos do usuário

	for {
		escolha, _ := leitura.ReadString('\n')
		escolha = strings.TrimSpace(escolha)
		comando = lerComando(escolha)
		if comando == "SAIR" {
			fmt.Println("Encerrando cliente...")
			return
		}
		dados := fmt.Sprintf("%s;%s;%s;%s", tipo, id, comando, "")
		conn.Write([]byte(dados))
		if comando == "ATUAR" {
			dadosAtuador := escolhaAtuador(leitura)
			conn.Write([]byte(dadosAtuador))
		}
		if comando == "MONITORAR_SENSOR" {
			dadosSensor := escolhaSensor(leitura)
			conn.Write([]byte(dadosSensor))
			time.Sleep(200 * time.Millisecond)
			// Modo monitoramento: aguarda ENTER para parar
			leitura.ReadString('\n') // bloqueia até ENTER

			// Envia qualquer coisa para desbloquear o conn.Read no servidor
			conn.Write([]byte("PARAR"))
		}
	}
}

var menuPronto = make(chan struct{}, 1)

func lerServer(conn net.Conn, done chan bool) {
	buffer := make([]byte, 4096)
	primeiraMsg := true
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("\n[CLIENTE]: Conexão com servidor perdida. Encerrando...")
			done <- true
			return
		}
		fmt.Printf("%s", string(buffer[:n]))

		if primeiraMsg {
			primeiraMsg = false
			menuPronto <- struct{}{} // sinaliza que o menu foi recebido
		}
	}
}

func lerComando(comando string) string {
	switch comando {
	case "1":
		return "LISTAR"
	case "2":
		return "LISTAR_ATUADORES"
	case "3":
		return "ATUAR"
	case "0":
		return "SAIR"
	case "4":
		return "MONITORAR_SENSOR"
	case "5":
		return "PARAR_SENSOR"
	default:
		fmt.Println("Comando desconhecido recebido.")
		return ""
	}

}

func escolhaAtuador(leitura *bufio.Reader) string {
	fmt.Print("ID do atuador:\n")
	bufferAtuador := make([]byte, 1024)
	n, _ := leitura.Read(bufferAtuador)
	id := strings.TrimSpace(string(bufferAtuador[:n]))

	fmt.Print("Comando (on/off): ")
	buffer := make([]byte, 1024)
	n, _ = leitura.Read(buffer)
	cmd := strings.TrimSpace(string(buffer[:n]))
	return fmt.Sprintf("COMANDO;%s;%s;%s", id, cmd, "")
}

func escolhaSensor(leitura *bufio.Reader) string {
	fmt.Print("ID do sensor:\n")
	bufferSensor := make([]byte, 1024)
	n, _ := leitura.Read(bufferSensor)
	id := strings.TrimSpace(string(bufferSensor[:n]))
	return fmt.Sprintf("COMANDO;%s;%s;%s", id, "", "")
}
