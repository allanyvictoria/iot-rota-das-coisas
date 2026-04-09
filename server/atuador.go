package main

import (
	"fmt"
	"log"
	"net"
)

// Estrutura para representar um atuador
type Atuador struct {
	ID     string
	Status string
	Conn   net.Conn
	Fila   chan Mensagem
}

// Worker dedicado para cada atuador
func workerAtuador(atuador *Atuador) {
	buffer := make([]byte, 1024)
	for msg := range atuador.Fila {
		// Envia comando pro atuador
		_, err := atuador.Conn.Write(ToBytes(msg))
		if err != nil {
			log.Printf("Erro ao enviar comando: %v", err)
			if msg.Resposta != nil {
				msg.Resposta <- "erro ao enviar"
			}
			return
		}

		// Espera a resposta do mesmo atuador
		n, err := atuador.Conn.Read(buffer)
		if err != nil {
			log.Printf("Atuador %s desconectado", atuador.ID)
			rwmu.Lock()
			close(atuador.Fila)
			delete(mapaAtuadores, atuador.ID)
			rwmu.Unlock()
			atuador.Conn.Close()
			if msg.Resposta != nil {
				msg.Resposta <- "atuador desconectado"
			}
			return
		}

		// Devolve a resposta pro cliente certo
		resposta, _ := ParseMensagem(buffer[:n])
		atuador.Status = resposta.COMANDO
		log.Printf("Status do atuador %s: %s", atuador.ID, resposta.COMANDO)

		if msg.Resposta != nil {
			msg.Resposta <- resposta.COMANDO
		}
	}
	log.Printf("Worker do atuador %s finalizado", atuador.ID)
}

// Função para enviar a lista de atuadores disponíveis para o cliente
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
