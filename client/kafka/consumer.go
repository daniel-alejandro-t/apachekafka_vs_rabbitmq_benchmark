package kafka

import (
	"context"
	"fmt"

	"example.com/client/utils"

	"github.com/IBM/sarama"
)

// Consumer es la estructura que implementa la interfaz sarama.ConsumerGroupHandler.
// Ya no se incluyen campos para métricas ni para la estructura hardware.Device.
type Consumer struct{}

// Setup se ejecuta antes de que el consumidor comience a consumir.
// Aquí no se realiza ninguna acción adicional.
func (Consumer) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup se ejecuta cuando el consumidor va a detenerse o redistribuir las particiones.
// Aquí tampoco se realiza ninguna acción.
func (Consumer) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim es el método que realmente procesa los mensajes.
// Cada mensaje tiene contenido aleatorio y de distintos tamaños.
func (Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		// Aquí solo se imprime el tamaño del mensaje.
		// El contenido es aleatorio, así que no se deserializa.
		fmt.Printf("Mensaje recibido con longitud %d bytes\n", len(msg.Value))
		// Marcamos el mensaje como procesado.
		session.MarkMessage(msg, "")
	}
	return nil
}

// Ejemplo de inicialización del consumidor
func StartConsumer(cfg *utils.Config, hostname string) {
	config := sarama.NewConfig()
	// config.Version = version
	config.Consumer.Return.Errors = true

	ctx := context.Background()

	// Se crea el grupo de consumidores utilizando el host y el grupo definido.
	group, err := sarama.NewConsumerGroup([]string{cfg.Kafka.Host}, fmt.Sprintf("%s-%s", cfg.Kafka.Group, hostname), config)
	utils.Fail(err, "sarama.NewConsumerGroup failed")

	// Goroutine para manejar errores del grupo.
	go func() {
		for err := range group.Errors() {
			utils.Warning(err, "mc.group.Errors")
		}
	}()

	// Goroutine para el loop infinito de consumo.
	go func() {
		for {
			topics := []string{fmt.Sprintf("%s-%s", cfg.Kafka.Topic, hostname)}
			handler := Consumer{}

			err := group.Consume(ctx, topics, handler)
			utils.Fail(err, "mc.group.Consume failed")
		}
	}()
}
