package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QUEUE_NAME  = "weather_logs_queue"
	MAX_RETRIES = 3
)

type WeatherLog struct {
	LocationLat float64 `json:"location_lat"`
	LocationLon float64 `json:"location_lon"`
	Timestamp   string  `json:"timestamp"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	WindSpeed   float64 `json:"wind_speed"`
	Condition   string  `json:"condition"`
	Cloudiness  int     `json:"cloudiness"`
}

func main() {
	rabbitmqURI := os.Getenv("RABBITMQ_URI") 
	if rabbitmqURI == "" {
		log.Fatal("ERRO: Variável RABBITMQ_URI não definida.")
	}

	apiURL := os.Getenv("NESTJS_API_URL")
	if apiURL == "" {
		log.Fatal("ERRO: Variável NESTJS_API_URL não definida.")
	}

	// --- Conexão ao RabbitMQ ---
	log.Printf("Conectando ao RabbitMQ em %s", rabbitmqURI)
	conn, err := amqp.Dial(rabbitmqURI)
	if err != nil {
		log.Fatalf("Falha ao conectar ao RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Falha ao abrir um canal: %v", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		QUEUE_NAME,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("Falha ao declarar a fila: %v", err)
	}

	// Define QoS: Consumir 1 por vez
	if err := ch.Qos(1, 0, false); err != nil {
		log.Fatalf("Falha ao definir QoS: %v", err)
	}

	msgs, err := ch.Consume(
		QUEUE_NAME,
		"",    // consumer tag
		false, // auto-ack (desabilitado para controle manual)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Fatalf("Falha ao registrar o consumidor: %v", err)
	}

	log.Println("--- Go Worker Iniciado e Pronto para Consumir ---")
	log.Printf("Aguardando mensagens na fila [%s]. API Target: %s", QUEUE_NAME, apiURL)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("[Mensagem Recebida] Consumindo DeliveryTag: %d", d.DeliveryTag)

			var logData WeatherLog
			if err := json.Unmarshal(d.Body, &logData); err != nil {
				log.Printf("ERRO de Deserialização: %v. Rejeitando mensagem (Não re-enqueue).", err)
				d.Reject(false)
				continue
			}

			if success := sendToNestJS(apiURL, logData); success {
				d.Ack(false)
			} else {
				log.Printf("FALHA CRÍTICA ao enviar para a API. Rejeitando e re-enfileirando.")
				d.Nack(false, true) 
			}
		}
	}()

	<-forever
}

func sendToNestJS(apiURL string, data WeatherLog) bool {
	jsonPayload, _ := json.Marshal(data)
	
	for i := 0; i < MAX_RETRIES; i++ {
		log.Printf("[Tentativa %d/%d] Enviando para API NestJS: %s", i+1, MAX_RETRIES, apiURL)
		
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonPayload))
		if err != nil {
			log.Printf("ERRO ao criar requisição: %v", err)
			cancel()
			return false
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		cancel()

		if err != nil {
			log.Printf("ERRO de Rede/Timeout: %v. Tentando novamente...", err)
			time.Sleep(time.Second * time.Duration(1<<uint(i)))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("--> [SUCESSO] Log de clima persistido na API. Status: %d", resp.StatusCode)
			return true
		} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			log.Printf("ERRO de Cliente (4xx): %d. Assumindo que o dado é inválido ou endpoint ausente. Nack(false).", resp.StatusCode)
			return false 
		} else {
			log.Printf("ERRO de Servidor (5xx): %d. Tentando novamente...", resp.StatusCode)
			time.Sleep(time.Second * time.Duration(1<<uint(i)))
			continue
		}
	}
	
	log.Println("ERRO: Todas as tentativas de envio falharam.")
	return false
}