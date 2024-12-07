package rabbitmq

import (
	"context"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"example.com/client/utils"

	"github.com/streadway/amqp"
)

type ConsumerMetricsData struct {
	MessagesReceived int64
	Failures         int64
	ElapsedTime      float64
	TotalLatency     float64
	AvgLatency       float64
}

// StartConsumer inicia el consumidor de RabbitMQ
func StartConsumer(cfg *utils.Config, hostname string) {
	conn, ch, queueName := initializeConsumer(cfg, hostname)
	defer closeConsumer(conn, ch)

	headers := []string{
		"MessagesReceived", "Failures", "ElapsedTime", "AvgLatency",
		// Agrega más encabezados si tienes más métricas
	}
	outputFile, writer := utils.SetupReportFile("rabbitmq_consumer_report.csv", headers)
	defer utils.FinalizeReportFile(outputFile, writer)

	runConsumerBenchmark(cfg, hostname, ch, queueName, writer)
}

// initializeConsumer configura y devuelve una conexión y canal de RabbitMQ
func initializeConsumer(cfg *utils.Config, hostname string) (*amqp.Connection, *amqp.Channel, string) {
	uri := fmt.Sprintf("amqp://%s:%s@%s:%d/", cfg.Rabbitmq.User, cfg.Rabbitmq.Password, cfg.Rabbitmq.Host, cfg.Rabbitmq.Port)
	conn, err := amqp.Dial(uri)
	utils.Fail(err, "amqp.Dial failed")

	ch, err := conn.Channel()
	utils.Fail(err, "conn.Channel failed")

	queueName := fmt.Sprintf("%s-%s", cfg.Rabbitmq.Queue, hostname)
	q, err := ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	utils.Fail(err, "ch.QueueDeclare failed")

	return conn, ch, q.Name
}

// closeConsumer cierra la conexión y el canal de RabbitMQ
func closeConsumer(conn *amqp.Connection, ch *amqp.Channel) {
	if err := ch.Close(); err != nil {
		utils.Warning(err, "Error closing channel")
	}
	if err := conn.Close(); err != nil {
		utils.Warning(err, "Error closing connection")
	}
}

// runConsumerBenchmark ejecuta el benchmark del consumidor
func runConsumerBenchmark(cfg *utils.Config, hostname string, ch *amqp.Channel, queueName string, writer *csv.Writer) {
	params := utils.GetBenchmarkParameters("rabbitmq_consumer_")

	// Ajustar el tiempo total de ejecución del consumidor para que coincida con el del productor
	totalDuration := calculateTotalDuration(params)

	// Ejecutar el ciclo de consumo durante el tiempo total
	messagesReceived, failures, totalLatency := performConsumerTestCycle(cfg, hostname, ch, queueName, totalDuration)

	elapsedTime := totalDuration.Seconds()

	avgLatency := 0.0
	if messagesReceived > 0 {
		avgLatency = totalLatency / float64(messagesReceived)
	}

	metricsData := ConsumerMetricsData{
		MessagesReceived: messagesReceived,
		Failures:         failures,
		ElapsedTime:      elapsedTime,
		TotalLatency:     totalLatency,
		AvgLatency:       avgLatency,
	}

	// Registrar las métricas
	logAndRecordConsumerMetrics(metricsData, writer)

	fmt.Printf("Prueba del consumidor finalizada. Resultados guardados en rabbitmq_consumer_report.csv\n")
}

func calculateTotalDuration(params utils.BenchmarkParameters) time.Duration {
	var totalDuration time.Duration
	rate := params.Rate
	for rate <= params.MaxRate {
		totalDuration += params.TestDuration
		rate += params.Increment
		// Si deseas agregar un tiempo de espera entre pruebas, inclúyelo aquí
	}
	return totalDuration
}

func performConsumerTestCycle(cfg *utils.Config, hostname string, ch *amqp.Channel, queueName string, testDuration time.Duration) (messagesReceived int64, failures int64, totalLatency float64) {
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	msgs, err := ch.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	utils.Fail(err, "ch.Consume failed")

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				// Canal cerrado
				return messagesReceived, failures, totalLatency
			}
			messagesReceived++

			// Verificar que el mensaje tenga al menos 8 bytes
			if len(msg.Body) < 8 {
				utils.Warning(fmt.Errorf("Mensaje demasiado corto para extraer timestamp"), "")
				failures++
				continue
			}

			// Extraer el timestamp
			timestamp := int64(binary.LittleEndian.Uint64(msg.Body[:8]))
			messageTime := time.Unix(0, timestamp)
			latency := time.Since(messageTime)
			totalLatency += latency.Seconds()

		case <-ctx.Done():
			return messagesReceived, failures, totalLatency
		}
	}
}

// logAndRecordConsumerMetrics registra las métricas en la salida estándar y en el archivo CSV
func logAndRecordConsumerMetrics(metricsData ConsumerMetricsData, writer *csv.Writer) {
	fmt.Printf("Mensajes Recibidos: %d, Fallos: %d, Tiempo Transcurrido: %.2f s, Latencia Promedio: %.4f s\n",
		metricsData.MessagesReceived, metricsData.Failures, metricsData.ElapsedTime, metricsData.AvgLatency)

	record := []string{
		strconv.FormatInt(metricsData.MessagesReceived, 10),
		strconv.FormatInt(metricsData.Failures, 10),
		fmt.Sprintf("%.2f", metricsData.ElapsedTime),
		fmt.Sprintf("%.4f", metricsData.AvgLatency),
		// Agrega más campos si has añadido más métricas
	}
	writer.Write(record)
	writer.Flush()
}
