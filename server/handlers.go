package main

import (
	"log"
	"net"
	"time"
)

// Função para lidar com a conexão do cliente e atuador.
func handleConnection(conn net.Conn) {

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		conn.Close()
		log.Println("Erro ao escutar conexao:", err)
		return
	}
	mensagem, err := ParseMensagem(buffer[:n])
	if err != nil {
		log.Printf("Mensagem inválida recebida: %v", err)
		return
	}
	id := mensagem.ID
	tipo := mensagem.TIPO
	comando := mensagem.COMANDO

	switch tipo {
	case "INICIAL":
		lerComando(conn, id)

	case "ATUADOR":

		rwmu.Lock()
		novoAtuador := &Atuador{
			ID:     id,
			Status: comando,
			Conn:   conn,
			Fila:   make(chan Mensagem, 10),
		}

		mapaAtuadores[id] = novoAtuador
		rwmu.Unlock()

		log.Printf("Atuador %s conectado", id)

		go workerAtuador(novoAtuador)
		return

	case "CLIENTE":
		rwmu.Lock()
		clientCount++
		rwmu.Unlock()
		log.Printf("[%d] CLIENTES CONECTADOS", clientCount)

		id := mensagem.ID

		defer func() {
			rwmu.Lock()
			clientCount--
			rwmu.Unlock()

			unsubscribe(id)
			log.Printf("[%d] CLIENTES CONECTADOS", clientCount)
			conn.Close()
		}()

		lerComando(conn, id)

	}

}

// Função para exibir o menu para o cliente.
func menu(conn net.Conn) {
	mensagem := "\n>>>>>>>>>>>>>>>>>> MENU <<<<<<<<<<<<<<<<<<\n" +
		"[1] - Listar sensores disponiveis\n" +
		"[2] - Listar atuadores disponiveis\n" +
		"[3] - Controlar atuador\n" +
		"[4] - Monitorar sensor (tempo real)\n" +
		"[0] - Sair\n\n" +
		"Digite sua escolha: "
	conn.Write([]byte(mensagem))
}

// Função para ler os comandos do cliente e processá-los.
func lerComando(conn net.Conn, clienteId string) {

	bufferCliente := make([]byte, 1024)
	for {
		menu(conn)

		// Pazo para o cliente enviar um comando(30 segundos) e não ser considerado inativo.
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(bufferCliente)
		conn.SetReadDeadline(time.Time{})

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("Cliente %s inativo, encerrando", clienteId)
			} else {
				log.Printf("Cliente %s desconectado: %v", clienteId, err)
			}
			return
		}

		msg, err := ParseMensagem(bufferCliente[:n])
		if err != nil {
			log.Printf("Mensagem inválida recebida: %v", err)
			return
		}

		switch msg.COMANDO {

		case "MONITORAR_SENSOR":
			for {
				buffer := make([]byte, 1024)
				n, _ := conn.Read(buffer)
				msgSensor, err := ParseMensagem(buffer[:n])
				if err != nil {
					log.Printf("Mensagem inválida recebida: %v", err)
					break
				}
				if msgSensor.COMANDO == "PARAR" {
					unsubscribe(clienteId)
					conn.Write([]byte("\n[SERVIDOR]: Monitoramento parado\n"))
					break
				}
				rwmu.RLock()
				_, exists := sensors[msgSensor.ID]
				rwmu.RUnlock()

				if !exists {
					conn.Write([]byte("[SERVIDOR]: SENSOR_NAO_ENCONTRADO\n"))
					continue // fica no loop esperando novo ID
				}

				sensorID := msgSensor.ID
				subscribe(clienteId, conn, sensorID)
				aviso := "[SERVIDOR]: Inscrito no sensor " + sensorID + "\n" +
					"\n[Pressione ENTER para parar o monitoramento]\n"
				conn.Write([]byte(aviso))

			}
		case "LISTAR":
			sendAvailableSensors(conn)

		case "LISTAR_ATUADORES":
			sendAvailableAtuadores(conn)

		case "ATUAR":
			buff := make([]byte, 1024)
			n, err := conn.Read(buff)
			if err != nil {
				log.Println("Erro ao ler do cliente:", err)
				return
			}

			cmdAtuador, err := ParseMensagem(buff[:n])
			if err != nil {
				log.Printf("Mensagem inválida recebida: %v", err)
				return
			}

			rwmu.RLock()
			atuador, exists := mapaAtuadores[cmdAtuador.ID]
			rwmu.RUnlock()

			if !exists {
				conn.Write([]byte("[SERVIDOR]: Atuador não encontrado"))
				continue
			}

			respCh := make(chan string, 1)
			cmdAtuador.Resposta = respCh // anexa o canal na mensagem
			atuador.Fila <- cmdAtuador

			select {
			case resposta := <-respCh:
				conn.Write([]byte("[ATUADOR]: " + resposta + "\n"))
			case <-time.After(5 * time.Second):
				conn.Write([]byte("[SERVIDOR]: Atuador não respondeu\n"))

			}

		case "SAIR":
			log.Println("Comando para desligar a conexão. Encerrando...")
			return
		}
	}
}
