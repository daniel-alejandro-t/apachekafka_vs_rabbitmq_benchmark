package rabbitmq

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"example.com/client/utils"
	// "github.com/streadway/amqp" TODO Evaluar rendimiento entre el anterior y el nuevo protocolo
	amqp "github.com/rabbitmq/amqp091-go"
)

// BenchmarkParameters contiene los parámetros para las pruebas de benchmark
type BenchmarkParameters struct {
	rate         int64
	maxRate      int64
	increment    int64
	testDuration time.Duration
	messageSize  int
}

// MetricsData contiene las métricas recopiladas durante las pruebas
type MetricsData struct {
	SendRate     float64
	MessagesSent int64
	Failures     int64
	ElapsedTime  float64
	// Agrega más campos si deseas recopilar más métricas
}

// StartProducer inicia el productor de RabbitMQ con incrementos progresivos en la tasa de mensajes.
func StartProducer(cfg *utils.Config, hostname string) {
	conn, ch, queueName := initializeProducer(cfg, hostname)
	defer closeProducer(conn, ch)

	headers := []string{
		"Rate", "MessagesSent", "Failures", "ElapsedTime",
	}

	params := getBenchmarkParameters()

	// El nombre del archivo debe tener las caracterísitcas del test realizado, extraido de params
	name_file_with_params := fmt.Sprintf("rabbitmq_producer_report-rate:%d-maxRate:%d-increment:%d-testDuration:%s-messageSize:%d.csv",
		params.rate, params.maxRate, params.increment, params.testDuration, params.messageSize)

	outputFile, writer := utils.SetupReportFile(name_file_with_params, headers)
	defer finalizeReportFile(outputFile, writer)

	runBenchmark(cfg, hostname, ch, queueName, writer)
}

// initializeProducer configura y devuelve una conexión y canal de RabbitMQ
func initializeProducer(cfg *utils.Config, hostname string) (*amqp.Connection, *amqp.Channel, string) {
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

// closeProducer cierra la conexión y el canal de RabbitMQ
func closeProducer(conn *amqp.Connection, ch *amqp.Channel) {
	if err := ch.Close(); err != nil {
		utils.Warning(err, "Error closing channel")
	}
	if err := conn.Close(); err != nil {
		utils.Warning(err, "Error closing connection")
	}
}

// finalizeReportFile guarda los cambios y cierra el archivo CSV.
func finalizeReportFile(outputFile *os.File, writer *csv.Writer) {
	writer.Flush()
	outputFile.Close()
}

// getBenchmarkParameters lee los parámetros del benchmark desde variables de entorno
// con el prefijo 'rabbitmq_' y aplica valores predeterminados si no están establecidas.
func getBenchmarkParameters() BenchmarkParameters {
	var (
		rate         int64 = 1000            // Tasa inicial predeterminada
		maxRate      int64 = 1000000         // Tasa máxima predeterminada
		increment    int64 = 100             // Incremento predeterminado
		testDuration       = 5 * time.Second // Duración predeterminada del ciclo
		messageSize  int   = 1024            // Tamaño predeterminado del mensaje en bytes
	)

	if val, exists := os.LookupEnv("rabbitmq_rate"); exists {
		if parsedVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			rate = parsedVal
		} else {
			fmt.Printf("Valor inválido para rabbitmq_rate: %s\n", val)
		}
	}

	if val, exists := os.LookupEnv("rabbitmq_maxRate"); exists {
		if parsedVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			maxRate = parsedVal
		} else {
			fmt.Printf("Valor inválido para rabbitmq_maxRate: %s\n", val)
		}
	}

	if val, exists := os.LookupEnv("rabbitmq_increment"); exists {
		if parsedVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			increment = parsedVal
		} else {
			fmt.Printf("Valor inválido para rabbitmq_increment: %s\n", val)
		}
	}

	if val, exists := os.LookupEnv("rabbitmq_testDuration"); exists {
		if parsedVal, err := strconv.ParseFloat(val, 64); err == nil {
			testDuration = time.Duration(parsedVal * float64(time.Second))
		} else {
			fmt.Printf("Valor inválido para rabbitmq_testDuration: %s\n", val)
		}
	}

	if val, exists := os.LookupEnv("rabbitmq_messageSize"); exists {
		if parsedVal, err := strconv.Atoi(val); err == nil {
			messageSize = parsedVal
		} else {
			fmt.Printf("Valor inválido para rabbitmq_messageSize: %s\n", val)
		}
	}

	return BenchmarkParameters{
		rate:         rate,
		maxRate:      maxRate,
		increment:    increment,
		testDuration: testDuration,
		messageSize:  messageSize,
	}
}

// runBenchmark ejecuta las pruebas de incremento progresivo de mensajes enviados.
func runBenchmark(cfg *utils.Config, hostname string, ch *amqp.Channel, queueName string, writer *csv.Writer) {
	// Obtener los parámetros del benchmark
	params := getBenchmarkParameters()

	rate := params.rate
	for rate <= params.maxRate {
		// Ejecutar el ciclo de prueba
		messagesSent, failures := performTestCycle(cfg, hostname, ch, queueName, rate, params.testDuration, params.messageSize)

		// Calcular métricas adicionales si es necesario
		elapsedTime := params.testDuration.Seconds()
		sendRate := float64(messagesSent) / elapsedTime

		// Crear un objeto MetricsData
		metricsData := MetricsData{
			SendRate:     sendRate,
			MessagesSent: messagesSent,
			Failures:     failures,
			ElapsedTime:  elapsedTime,
			// Agrega más métricas si lo deseas
		}

		// Registrar las métricas
		logAndRecordMetrics(rate, metricsData, writer)

		// Verificar si se encontraron fallos
		if failures > 0 {
			fmt.Printf("Fallo detectado a %d msg/s. Finalizando prueba.\n", rate)
			break
		}

		fmt.Print("Esperando 5 segundos para la siguiente prueba...\n")

		rate += params.increment
	}

	fmt.Printf("Prueba finalizada. Resultados guardados en rabbitmq_producer_report.csv\n")
}

// performTestCycle ejecuta un ciclo de prueba con una tasa de mensajes específica.
func performTestCycle(cfg *utils.Config, hostname string, ch *amqp.Channel, queueName string, rate int64, testDuration time.Duration, messageSize int) (messagesSent int64, failures int64) {
	startTime := time.Now()
	endTime := startTime.Add(testDuration)
	ticker := time.NewTicker(time.Second / time.Duration(rate))
	defer ticker.Stop()

	fmt.Print("Iniciando prueba con tasa de ", rate, " msg/s\n")

	for time.Now().Before(endTime) {
		select {
		case <-ticker.C:
			err := ch.Publish(
				"",        // exchange
				queueName, // routing key
				false,     // mandatory
				false,     // immediate
				amqp.Publishing{
					ContentType: "application/octet-stream",
					Body:        generateMessage(messageSize),
				},
			)

			if err != nil {
				utils.Warning(err, "ch.Publish failed")
				failures++ // Incrementar el contador de fallos
			} else {
				messagesSent++ // Incrementar el contador de mensajes enviados
			}

		}
	}

	return messagesSent, failures
}

// logAndRecordMetrics registra las métricas en la salida estándar y en el archivo CSV.
func logAndRecordMetrics(rate int64, metricsData MetricsData, writer *csv.Writer) {
	fmt.Printf("Tasa: %d msg/s, Enviados: %d, Fallos: %d, Tiempo: %.2f s, Tasa de envío real: %.2f msg/s\n",
		rate, metricsData.MessagesSent, metricsData.Failures, metricsData.ElapsedTime, metricsData.SendRate)

	record := []string{
		strconv.FormatInt(rate, 10),
		strconv.FormatInt(metricsData.MessagesSent, 10),
		strconv.FormatInt(metricsData.Failures, 10),
		fmt.Sprintf("%.2f", metricsData.ElapsedTime),
		// Agrega más campos si has añadido más métricas
	}
	writer.Write(record)
	writer.Flush()
}

// generateMessage genera un mensaje de un tamaño específico en bytes, incluyendo un timestamp.
func generateMessage(size int) []byte {
	if size < 8 {
		size = 8 // Asegurar que haya espacio para el timestamp
	}
	message := make([]byte, size)
	timestamp := time.Now().UnixNano()
	binary.LittleEndian.PutUint64(message[:8], uint64(timestamp))
	for i := 8; i < size; i++ {
		message[i] = byte('A' + (i % 26))
	}
	return message
}
