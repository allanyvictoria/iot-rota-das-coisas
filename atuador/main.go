package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
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
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.Dial("tcp", getServerAddr())
	if err != nil {
		log.Fatal("Erro ao conectar ao servidor:", err)
	}
	defer conn.Close()

	var status string = "DESLIGADO"
	id := hostname
	tipo := "ATUADOR"
	comando := status

	dados := fmt.Sprintf("%s;%s;%s;%s", tipo, id, comando, "")
	_, err = conn.Write([]byte(dados))
	if err != nil {
		log.Fatal("Erro ao enviar dados para o servidor:", err)
	}

	for {
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("\n[ATUADOR]: Conexão com servidor perdida. Encerrando...")
			os.Exit(0)
		}

		comando := string(buffer[:n])
		parts := strings.Split(comando, ";")
		if len(parts) < 4 {
			log.Println("Mensagem do servidor em formato inválido:", comando)
			continue
		}
		comando = parts[2]

		fmt.Printf("Comando recebido do servidor: %s\n", comando)

		resposta := comandoAtuador(comando, &status)
		respostaMsg := fmt.Sprintf("%s;%s;%s;%s", tipo, id, resposta, "")

		conn.Write([]byte(respostaMsg))

	}
}

func comandoAtuador(cmd string, status *string) string {
	switch cmd {
	case "on":
		if *status == "DESLIGADO" {
			*status = "LIGADO"
			fmt.Println("Atuador LIGADO")
			return "Atuador LIGADO"
		} else {
			fmt.Println("Atuador já está LIGADO")
			return "Atuador já está LIGADO"
		}

	case "off":
		if *status == "LIGADO" {
			*status = "DESLIGADO"
			fmt.Println("Atuador DESLIGADO")
			return "Atuador DESLIGADO"
		} else {
			fmt.Println("Atuador já está DESLIGADO")
			return "Atuador já está DESLIGADO"
		}
	default:
		fmt.Println("Comando desconhecido recebido.")
		return "Comando desconhecido"
	}
}
