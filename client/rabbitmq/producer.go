package rabbitmq

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"example.com/client/utils"
	// "github.com/streadway/amqp" TODO Evaluar rendimiento entre el anterior y el nuevo protocolo
	amqp "github.com/rabbitmq/amqp091-go"
)

// BenchmarkParameters contiene los parámetros para las pruebas de benchmark
type BenchmarkParameters struct {
	rate         int64
	increment    int64
	testDuration time.Duration
	messageSize  int
}

// MetricsData contiene las métricas recopiladas durante las pruebas
type MetricsData struct {
	Rate                 int64
	MessagesSent         int64
	Failures             int64
	DurationSeconds      float64
	SendRate             float64
	RequestLatencyP50    float64
	RequestLatencyP95    float64
	RequestLatencyP99    float64
	BatchSizeAvg         float64
	RecordsPerRequestAvg float64
	IncomingByteRate     float64
	OutgoingByteRate     float64
	RequestsInFlight     int64
}

// StartProducer inicia el productor de RabbitMQ con incrementos progresivos en la tasa de mensajes.
func StartProducer(cfg *utils.Config, hostname string) {
	conn, ch, queueName := initializeProducer(cfg, hostname)
	defer closeProducer(conn, ch)

	headers := []string{
		"Rate", "MessagesSent", "Failures", "DurationSeconds", "SendRate",
		"RequestLatencyP50", "RequestLatencyP95", "RequestLatencyP99",
		"BatchSizeAvg", "RecordsPerRequestAvg", "IncomingByteRate",
		"OutgoingByteRate", "RequestsInFlight",
	}

	params := getBenchmarkParameters()

	// El nombre del archivo debe tener las caracterísitcas del test realizado, extraido de params
	name_file_with_params := fmt.Sprintf("rabbitmq_producer_report-rate:%d-increment:%d-testDuration:%s-messageSize:%d.csv",
		params.rate, params.increment, params.testDuration, params.messageSize)

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
// TODO Enviar a utils/ para reutilizar en otros módulos
func finalizeReportFile(outputFile *os.File, writer *csv.Writer) {
	writer.Flush()
	outputFile.Close()
}

func getBenchmarkParameters() BenchmarkParameters {
	var (
		rate         int64 = 10000
		increment    int64 = 5000
		testDuration       = 2 * time.Second
		messageSize  int   = 10240
	)

	if val, exists := os.LookupEnv("rabbitmq_rate"); exists {
		if parsedVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			rate = parsedVal
		} else {
			fmt.Printf("Valor inválido para rabbitmq_rate: %s\n", val)
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
		increment:    increment,
		testDuration: testDuration,
		messageSize:  messageSize,
	}
}

func runBenchmark(cfg *utils.Config, hostname string, ch *amqp.Channel, queueName string, writer *csv.Writer) {
	params := getBenchmarkParameters()

	rate := params.rate
	for {
		latencies, messagesSent, failures := performTestCycle(cfg, hostname, ch, queueName, rate, params.testDuration, params.messageSize)

		// Calcular métricas
		elapsedTime := params.testDuration.Seconds()
		sendRate := float64(messagesSent) / elapsedTime

		// Calcular percentiles de latencia
		p50, p95, p99 := computePercentiles(latencies)

		// En este ejemplo, asumimos:
		// - Un mensaje por request => BatchSizeAvg = 1
		// - RecordsPerRequestAvg = 1
		batchSizeAvg := 1.0
		recordsPerRequestAvg := 1.0

		// Cálculo de OutgoingByteRate: (messagesSent * messageSize) / elapsedTime
		outgoingByteRate := float64(messagesSent*int64(params.messageSize)) / elapsedTime
		// IncomingByteRate: 0 en este escenario, ya que solo producimos
		incomingByteRate := 0.0

		// RequestsInFlight: asumiendo operaciones sin async
		requestsInFlight := int64(1)

		metricsData := MetricsData{
			Rate:                 rate,
			MessagesSent:         messagesSent,
			Failures:             failures,
			DurationSeconds:      elapsedTime,
			SendRate:             sendRate,
			RequestLatencyP50:    p50,
			RequestLatencyP95:    p95,
			RequestLatencyP99:    p99,
			BatchSizeAvg:         batchSizeAvg,
			RecordsPerRequestAvg: recordsPerRequestAvg,
			IncomingByteRate:     incomingByteRate,
			OutgoingByteRate:     outgoingByteRate,
			RequestsInFlight:     requestsInFlight,
		}

		logAndRecordMetrics(metricsData, writer)

		if failures > 0 {
			fmt.Printf("Fallo detectado a %d msg/s. Finalizando prueba.\n", rate)
			break
		}

		fmt.Printf("Esperando %.2f segundos para la siguiente prueba...\n", elapsedTime)
		rate += params.increment
	}

	fmt.Printf("Prueba finalizada.\n")
}

func performTestCycle(cfg *utils.Config, hostname string, ch *amqp.Channel, queueName string, rate int64, testDuration time.Duration, messageSize int) (latencies []float64, messagesSent int64, failures int64) {
	startTime := time.Now()
	endTime := startTime.Add(testDuration)
	ticker := time.NewTicker(time.Second / time.Duration(rate))
	defer ticker.Stop()

	fmt.Printf("Iniciando prueba con tasa de %d msg/s\n", rate)

	for time.Now().Before(endTime) {
		select {
		case <-ticker.C:
			sendStart := time.Now()
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

			sendLatency := time.Since(sendStart).Seconds() * 1000.0 // latencia en ms
			if err != nil {
				utils.Warning(err, "ch.Publish failed")
				failures++
			} else {
				messagesSent++
				latencies = append(latencies, sendLatency)
			}
		}
	}

	return latencies, messagesSent, failures
}

// logAndRecordMetrics registra las métricas en la salida estándar y en el archivo CSV.
func logAndRecordMetrics(metricsData MetricsData, writer *csv.Writer) {
	fmt.Printf("Rate: %d msg/s, MessagesSent: %d, Failures: %d, DurationSeconds: %.2f, SendRate: %.2f, P50: %.2f ms, P95: %.2f ms, P99: %.2f ms, BatchSizeAvg: %.2f, RecordsPerRequestAvg: %.2f, IncomingByteRate: %.2f, OutgoingByteRate: %.2f, RequestsInFlight: %d\n",
		metricsData.Rate,
		metricsData.MessagesSent,
		metricsData.Failures,
		metricsData.DurationSeconds,
		metricsData.SendRate,
		metricsData.RequestLatencyP50,
		metricsData.RequestLatencyP95,
		metricsData.RequestLatencyP99,
		metricsData.BatchSizeAvg,
		metricsData.RecordsPerRequestAvg,
		metricsData.IncomingByteRate,
		metricsData.OutgoingByteRate,
		metricsData.RequestsInFlight,
	)

	record := []string{
		strconv.FormatInt(metricsData.Rate, 10),
		strconv.FormatInt(metricsData.MessagesSent, 10),
		strconv.FormatInt(metricsData.Failures, 10),
		fmt.Sprintf("%.2f", metricsData.DurationSeconds),
		fmt.Sprintf("%.2f", metricsData.SendRate),
		fmt.Sprintf("%.2f", metricsData.RequestLatencyP50),
		fmt.Sprintf("%.2f", metricsData.RequestLatencyP95),
		fmt.Sprintf("%.2f", metricsData.RequestLatencyP99),
		fmt.Sprintf("%.2f", metricsData.BatchSizeAvg),
		fmt.Sprintf("%.2f", metricsData.RecordsPerRequestAvg),
		fmt.Sprintf("%.2f", metricsData.IncomingByteRate),
		fmt.Sprintf("%.2f", metricsData.OutgoingByteRate),
		strconv.FormatInt(metricsData.RequestsInFlight, 10),
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

// computePercentiles calcula P50, P95, P99
func computePercentiles(latencies []float64) (p50, p95, p99 float64) {
	if len(latencies) == 0 {
		return 0, 0, 0
	}
	sort.Float64s(latencies)

	p50 = percentile(latencies, 50)
	p95 = percentile(latencies, 95)
	p99 = percentile(latencies, 99)

	return p50, p95, p99
}

// percentile obtiene el valor del percentil sobre un slice ordenado
func percentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return 0.0
	}
	index := (p / 100.0) * float64(len(data)-1)
	i := int(math.Floor(index))
	j := int(math.Ceil(index))
	if i == j {
		return data[i]
	}
	frac := index - float64(i)
	return data[i]*(1-frac) + data[j]*frac
}
