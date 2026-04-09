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
	// Canal para sinalizar quando o menu do servidor for recebido
	done := make(chan bool)

	// Obtém o hostname do container para usar como ID do sensor
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	// Conecta ao servidor TCP
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

	// Goroutine para ler mensagens do servidor
	go func() {
		<-done
		os.Exit(0) // encerra mesmo que o loop principal esteja bloqueado em ReadString
	}()

	go lerServer(conn, done)

	<-menuPronto // espera o menu ser recebido antes de aceitar comandos do usuário

	// Loop para ler comandos do usuário e enviar para o servidor
	for {
		escolha, _ := leitura.ReadString('\n')
		escolha = strings.TrimSpace(escolha)
		comando = lerComando(escolha)

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

			conn.Write([]byte(";;PARAR;"))
		}
		if comando == "SAIR" {
			fmt.Println("Encerrando cliente...")
			return
		}
	}
}

var menuPronto = make(chan struct{}, 1) // canal para sinalizar que o menu foi recebido

// Função para ler mensagens do servidor
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

// Função para exibir o menu para o cliente
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

// função para usuario indicar o comando do atuador e o id do mesmo
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

// função para usuario indicar o id do sensor
func escolhaSensor(leitura *bufio.Reader) string {
	fmt.Print("ID do sensor:\n")
	bufferSensor := make([]byte, 1024)
	n, _ := leitura.Read(bufferSensor)
	id := strings.TrimSpace(string(bufferSensor[:n]))
	return fmt.Sprintf("COMANDO;%s;%s;%s", id, "", "")
}
