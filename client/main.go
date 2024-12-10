package main

import (
	"log"
	"os"
	"sync"

	"example.com/client/kafka"

	// "example.com/client/metrics"
	"example.com/client/rabbitmq"
	"example.com/client/utils"
)

var (
	client   string
	hostname string
)

func init() {
	client = os.Getenv("CLIENT")
	if client == "" {
		log.Fatalln("You MUST set CLIENT env variable!")
	}

	hostname = "localhost"
	// hostname = os.Getenv("HOSTNAME")
	//if client == "" {
	//	log.Fatalln("You MUST set HOSTNAME env variable!")
	// }
}

// main is the entry point of the application.
func main() {
	cfg := new(utils.Config)
	cfg.LoadConfig("config.yaml")

	var wg sync.WaitGroup

	switch client {
	case "both":
		wg.Add(2)
		go func() {
			defer wg.Done()
			runKafkaTest(cfg, hostname)
		}()
		go func() {
			defer wg.Done()
			runRabbitMQTest(cfg, hostname)
		}()
		log.Println("Starting both Kafka and RabbitMQ tests")
		wg.Wait()
	case "rabbitmq":
		wg.Add(1)
		go func() {
			defer wg.Done()
			runRabbitMQTest(cfg, hostname)
		}()
		log.Println("Starting RabbitMQ test")
		wg.Wait()
	case "kafka":
		wg.Add(1)
		go func() {
			defer wg.Done()
			runKafkaTest(cfg, hostname)
		}()
		log.Println("Starting Kafka test")
		wg.Wait()
	default:
		log.Fatalf("%s client is NOT supported", client)
	}
}

func runKafkaTest(cfg *utils.Config, hostname string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Kafka test panicked: %v", r)
		}
	}()
	kafka.StartProducer(cfg, hostname)
}

// FIXME Se detiene - ch.Publish failed: Exception (504) Reason: "channel/connection is not open"
func runRabbitMQTest(cfg *utils.Config, hostname string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("RabbitMQ test panicked: %v", r)
		}
	}()
	rabbitmq.StartProducer(cfg, hostname)
	rabbitmq.StartConsumer(cfg, hostname)
}
