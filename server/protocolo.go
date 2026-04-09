package main

import (
	"fmt"
	"strings"
)

// Estrutura seguida para as mensagens trocadas entre clientes, atuadores e sensores
type Mensagem struct {
	TIPO     string
	ID       string
	COMANDO  string
	VALOR    string
	Resposta chan string
}

// Função para analisar a mensagem recebida e extrair os campos
func ParseMensagem(data []byte) (Mensagem, error) {
	mensagem := strings.TrimSpace(string(data))
	parts := strings.Split(mensagem, ";")

	if len(parts) < 4 {
		return Mensagem{}, fmt.Errorf("mensagem malformada: '%s'", mensagem)
	}

	return Mensagem{
		TIPO:    strings.TrimSpace(parts[0]),
		ID:      strings.TrimSpace(parts[1]),
		COMANDO: strings.TrimSpace(parts[2]),
		VALOR:   strings.TrimSpace(parts[3]),
	}, nil
}

// Função para converter uma mensagem em bytes para envio
func ToBytes(m Mensagem) []byte {
	return []byte(fmt.Sprintf("%s;%s;%s;%s", m.TIPO, m.ID, m.COMANDO, m.VALOR))
}
