package main

import (
	"log"
	"net"
)

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

func unsubscribe(clienteID string) {

	rwmu.Lock()
	defer rwmu.Unlock()

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
