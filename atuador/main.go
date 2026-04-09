package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
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

	// Conecta ao servidor TCP
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

	// Loop principal do atuador, aguardando comandos do servidor
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

// função para processar o comando recebido e atualizar o status do atuador
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
