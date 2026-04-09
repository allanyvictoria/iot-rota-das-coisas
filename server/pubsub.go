package main

import (
	"log"
	"net"
)

// Funções para gerenciar o sistema de publicação/inscrição (pub/sub) dos sensores
func subscribe(clienteID string, conn net.Conn, SensorID string) {
	topico := "SENSOR" + ":" + SensorID

	rwmu.Lock()
	defer rwmu.Unlock()

	// Remove inscrição antiga
	if antigo, ok := clienteTopico[clienteID]; ok {
		delete(topicos[antigo], clienteID)
	}

	// Adiciona nova
	clienteTopico[clienteID] = topico

	if topicos[topico] == nil {
		topicos[topico] = make(map[string]net.Conn)
	}

	topicos[topico][clienteID] = conn
	log.Printf("Cliente %s inscrito no tópico %s", clienteID, topico)
}

// Função para remover um cliente do tópico em que está inscrito
func unsubscribe(clienteID string) {

	rwmu.Lock()
	defer rwmu.Unlock()

	// Verifica se o cliente está inscrito em algum tópico
	topico, ok := clienteTopico[clienteID]
	if !ok {
		return
	}

	delete(topicos[topico], clienteID)
	delete(clienteTopico, clienteID)

	if len(topicos[topico]) == 0 {
		delete(topicos, topico)
	}

	log.Printf("Cliente %s removido do tópico %s", clienteID, topico)
}

// Função para enviar uma mensagem para todos os clientes inscritos em um tópico específico
func enviarParaTopico(topico string, msg string) {
	rwmu.RLock()
	clientes := topicos[topico]
	rwmu.RUnlock()

	for id, conn := range clientes {
		_, err := conn.Write([]byte(msg))
		if err != nil {
			log.Printf("Erro ao enviar para cliente %s", id)
		}
	}
}
