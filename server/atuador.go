package main

import (
	"fmt"
	"log"
	"net"
)

type Atuador struct {
	ID       string
	Status   string
	Conn     net.Conn
	Fila     chan Mensagem
	Resposta chan string
}

func workerAtuador(atuador *Atuador) {
	for msg := range atuador.Fila {
		log.Printf("Enviando comando para atuador %s: %s\n", atuador.ID, msg.COMANDO)
		_, err := atuador.Conn.Write(ToBytes(msg))
		if err != nil {
			log.Printf("Erro ao enviar comando : %v", err)
			return
		}
	}
	log.Printf("Worker do atuador %s finalizado", atuador.ID)
}

func escutarAtuador(conn net.Conn, id string) {
	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			// Verifica se foi o servidor que caiu ou o atuador que desconectou
			rwmu.RLock()
			_, aindaExiste := mapaAtuadores[id]
			rwmu.RUnlock()

			if aindaExiste {
				log.Printf("[ATUADOR %s]: Conexão com servidor perdida. Encerrando...", id)
			} else {
				log.Printf("[ATUADOR %s]: Desconectado", id)
			}

			rwmu.Lock()
			if atuador, ok := mapaAtuadores[id]; ok {
				close(atuador.Fila)
				delete(mapaAtuadores, id)
			}
			rwmu.Unlock()

			conn.Close()
			return
		}

		msgStatus, err := ParseMensagem(buffer[:n])
		if err != nil {
			log.Printf("Mensagem inválida recebida: %v", err)
			continue
		}
		status := msgStatus.COMANDO
		mapaAtuadores[id].Status = status
		mapaAtuadores[id].Resposta <- status
		log.Printf("Status do atuador %s: %s\n", id, status)
	}
}

func sendAvailableAtuadores(conn net.Conn) {
	rwmu.RLock()
	defer rwmu.RUnlock()
	if len(mapaAtuadores) > 0 {
		msginicial := "\n>>>>>>>>>> ATUADORES DISPONIVEIS <<<<<<<<<<<<\n"
		conn.Write([]byte(msginicial))
		for _, atuador := range mapaAtuadores {
			msg := fmt.Sprintf("\nID: %s\nSTATUS:%s\n", atuador.ID, atuador.Status)
			conn.Write([]byte(msg))
		}
	} else {
		conn.Write([]byte("\n[ATUADOR] Nenhum atuador disponivel.\n"))
	}
}
