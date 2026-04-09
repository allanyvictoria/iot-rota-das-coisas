package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Resultado struct {
	Cliente  int
	Comando  string
	Resposta string
	Status   string
	Duracao  time.Duration
}

func montarMensagem(tipo, id, comando, valor string) []byte {
	return []byte(fmt.Sprintf("%s;%s;%s;%s", tipo, id, comando, valor))
}

func clienteWorker(host string, clienteID int, atuadorID string, comando string, wg *sync.WaitGroup, resultados chan<- Resultado, inicio <-chan struct{}) {
	defer wg.Done()

	conn, err := net.DialTimeout("tcp", host, 5*time.Second)
	if err != nil {
		resultados <- Resultado{clienteID, comando, err.Error(), "ERRO", 0}
		return
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Handshake inicial
	id := fmt.Sprintf("teste_%d", clienteID)
	conn.Write(montarMensagem("CLIENTE", id, "INICIAL", ""))
	buf := make([]byte, 4096)
	conn.Read(buf) // boas vindas + menu

	// Aguarda sinal para disparar todos ao mesmo tempo
	<-inicio

	t := time.Now()

	// Envia ATUAR
	conn.Write(montarMensagem("CLIENTE", id, "ATUAR", ""))
	time.Sleep(50 * time.Millisecond)

	// Envia comando para o atuador
	conn.Write(montarMensagem("COMANDO", atuadorID, comando, ""))

	// Lê resposta
	n, err := conn.Read(buf)
	duracao := time.Since(t)

	if err != nil {
		resultados <- Resultado{clienteID, comando, err.Error(), "ERRO", duracao}
		return
	}

	resposta := strings.TrimSpace(string(buf[:n]))
	resultados <- Resultado{clienteID, comando, resposta, "OK", duracao}
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Uso: go run test_concorrencia.go <host:porta> <n_clientes> <atuador_id>")
		fmt.Println("Exemplo: go run test_concorrencia.go localhost:1053 10 atuador1")
		os.Exit(1)
	}

	host := os.Args[1]
	nClientes, err := strconv.Atoi(os.Args[2])
	if err != nil || nClientes < 1 {
		fmt.Println("n_clientes deve ser um número inteiro positivo")
		os.Exit(1)
	}
	atuadorID := os.Args[3]

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("  Teste de Concorrência — %d clientes simultâneos\n", nClientes)
	fmt.Printf("  Alvo: atuador '%s' em %s\n", atuadorID, host)
	fmt.Printf("%s\n\n", strings.Repeat("=", 60))

	var wg sync.WaitGroup
	resultadosCh := make(chan Resultado, nClientes)
	inicio := make(chan struct{}) // canal para disparar todos ao mesmo tempo

	comandos := []string{"on", "off"}

	// Cria todos os clientes já conectados
	for i := 1; i <= nClientes; i++ {
		wg.Add(1)
		cmd := comandos[i%2]
		go clienteWorker(host, i, atuadorID, cmd, &wg, resultadosCh, inicio)
	}

	// Pequena pausa para todos conectarem antes de disparar
	time.Sleep(300 * time.Millisecond)

	// Dispara todos ao mesmo tempo
	tempoInicio := time.Now()
	close(inicio)

	// Aguarda todos terminarem
	wg.Wait()
	close(resultadosCh)
	tempoTotal := time.Since(tempoInicio)

	// Coleta resultados
	var resultados []Resultado
	for r := range resultadosCh {
		resultados = append(resultados, r)
	}

	// Relatório
	fmt.Printf("%-10s %-8s %-8s %-10s %s\n", "Cliente", "Comando", "Status", "Duração", "Resposta")
	fmt.Println(strings.Repeat("-", 60))

	ok, erros := 0, 0
	for _, r := range resultados {
		fmt.Printf("%-10d %-8s %-8s %-10s %s\n",
			r.Cliente, r.Comando, r.Status, r.Duracao.Round(time.Millisecond), r.Resposta)
		if r.Status == "OK" {
			ok++
		} else {
			erros++
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("  Total:        %d requisições\n", len(resultados))
	fmt.Printf("  Sucesso:      %d\n", ok)
	fmt.Printf("  Erros:        %d\n", erros)
	fmt.Printf("  Tempo total:  %s\n", tempoTotal.Round(time.Millisecond))
	fmt.Printf("%s\n\n", strings.Repeat("=", 60))
}
