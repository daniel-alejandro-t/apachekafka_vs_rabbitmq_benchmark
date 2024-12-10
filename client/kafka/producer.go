package kafka

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"example.com/client/utils"

	"github.com/IBM/sarama"
	gometrics "github.com/rcrowley/go-metrics"
)

type MetricsData struct {
	requestSize          float64 // TODO
	SendRate             float64
	RequestLatencyP50    float64
	RequestLatencyP95    float64
	RequestLatencyP99    float64
	BatchSizeAvg         float64
	RecordsPerRequestAvg float64
	CompressionRatioAvg  float64
	IncomingByteRate     float64
	OutgoingByteRate     float64
	RequestsInFlight     int64
}

func StartProducer(cfg *utils.Config, hostname string) {
	producer, config, startTime := initializeProducer(cfg)
	defer closeProducer(producer)

	params := getBenchmarkParameters()

	// El nombre del archivo debe tener las caracterísitcas del test realizado, extraido de params
	name_file_with_params := fmt.Sprintf("kafka_producer_report-rate:%d-increment:%d-testDuration:%s-messageSize:%d.csv",
		params.rate, params.increment, params.testDuration, params.messageSize)

	headers := []string{
		"Rate", "MessagesSent", "Failures", "DurationSeconds",
		"SendRate", "RequestLatencyP50", "RequestLatencyP95", "RequestLatencyP99",
		"BatchSizeAvg", "RecordsPerRequestAvg", "IncomingByteRate", "OutgoingByteRate", "RequestsInFlight",
	}

	outputFile, writer := utils.SetupReportFile(name_file_with_params, headers)
	defer finalizeReportFile(outputFile, writer)

	runBenchmark(cfg, hostname, producer, config, startTime, writer)
}

// initializeProducer configura y devuelve un productor Kafka junto con el tiempo de inicio.
// TODO Remplazar el productor asíncrono por el síncrono que garantiza la recepción de mensajes.
func initializeProducer(cfg *utils.Config) (sarama.AsyncProducer, *sarama.Config, time.Time) {
	version, err := sarama.ParseKafkaVersion(cfg.Kafka.Version)
	utils.Fail(err, "sarama.ParseKafkaVersion failed")

	config := sarama.NewConfig()
	config.Version = version
	config.Producer.Return.Successes = false
	config.Producer.RequiredAcks = sarama.WaitForAll

	producer, err := sarama.NewAsyncProducer([]string{cfg.Kafka.Host}, config)
	utils.Fail(err, "sarama.NewAsyncProducer failed")

	return producer, config, time.Now()
}

func listAllMetrics(registry gometrics.Registry) {
	fmt.Println("Métricas registradas:")
	registry.Each(func(name string, i interface{}) {
		fmt.Printf("Nombre: %s, Tipo: %T\n", name, i)
	})
}

// closeProducer cierra el productor Kafka de forma segura.
func closeProducer(producer sarama.AsyncProducer) {
	if err := producer.Close(); err != nil {
		utils.Warning(err, "Error closing producer")
	}
}

// finalizeReportFile guarda los cambios y cierra el archivo CSV.
func finalizeReportFile(outputFile *os.File, writer *csv.Writer) {
	writer.Flush()
	outputFile.Close()
}

// BenchmarkParameters contiene los parámetros para las pruebas de benchmark
type BenchmarkParameters struct {
	rate         int64
	increment    int64
	testDuration time.Duration
	messageSize  int
}

// getBenchmarkParameters lee los parámetros del benchmark desde variables de entorno
// con el prefijo 'kafka_' y aplica valores predeterminados si no están establecidas.
func getBenchmarkParameters() BenchmarkParameters {
	var (
		rate         int64 = 100             // Tasa inicial predeterminada
		increment    int64 = 100             // Incremento predeterminado
		testDuration       = 5 * time.Second // Duración predeterminada del ciclo
		messageSize  int   = 1024            // Tamaño predeterminado del mensaje en bytes
	)

	if val, exists := os.LookupEnv("kafka_rate"); exists {
		if parsedVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			rate = parsedVal
		} else {
			fmt.Printf("Valor inválido para kafka_rate: %s\n", val)
		}
	}

	if val, exists := os.LookupEnv("kafka_increment"); exists {
		if parsedVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			increment = parsedVal
		} else {
			fmt.Printf("Valor inválido para kafka_increment: %s\n", val)
		}
	}

	if val, exists := os.LookupEnv("kafka_testDuration"); exists {
		if parsedVal, err := strconv.ParseFloat(val, 64); err == nil {
			testDuration = time.Duration(parsedVal * float64(time.Second))
		} else {
			fmt.Printf("Valor inválido para kafka_testDuration: %s\n", val)
		}
	}

	if val, exists := os.LookupEnv("kafka_messageSize"); exists {
		if parsedVal, err := strconv.Atoi(val); err == nil {
			messageSize = parsedVal
		} else {
			fmt.Printf("Valor inválido para kafka_messageSize: %s\n", val)
		}
	}

	return BenchmarkParameters{
		rate:         rate,
		increment:    increment,
		testDuration: testDuration,
		messageSize:  messageSize,
	}
}

func runBenchmark(cfg *utils.Config, hostname string, producer sarama.AsyncProducer, config *sarama.Config, startTime time.Time, writer *csv.Writer) {
	totalAckLatency := int64(0) // Inicializar la latencia acumulada de ACKs

	// Obtener los parámetros del benchmark
	params := getBenchmarkParameters()

	// Impresión en consola de las métricas disponibles
	// listAllMetrics(config.MetricRegistry)

	rate := params.rate
	for {
		// Ejecutar el ciclo de prueba
		messagesSent, failures := performTestCycle(
			cfg, hostname, producer, rate, params.testDuration, &totalAckLatency, params.messageSize,
		)

		// Recopilar las métricas después del ciclo
		metricsData := collectMetrics(config.MetricRegistry, cfg)

		// Registrar las métricas
		logAndRecordMetricsWithLatencyAndMetricsData(rate, messagesSent, failures, params.testDuration.Seconds(), metricsData, writer)

		// Verificar si se encontraron fallos
		if failures > 0 {
			fmt.Printf("Fallo detectado a %d msg/s. Finalizando prueba.\n", rate)
			break
		}

		rate += params.increment
	}

	fmt.Printf("Prueba finalizada. Resultados guardados en kafka_producer_report.csv\n")
}

func collectMetrics(registry gometrics.Registry, cfg *utils.Config) MetricsData {
	var data MetricsData
	// topic := cfg.Kafka.Topic

	// Send Rate: record-send-rate
	if meter := getMeter("record-send-rate", registry); meter != nil {
		data.SendRate = meter.Rate1()
	} else {
		fmt.Println("Meter 'record-send-rate' no encontrado.")
	}

	// Request Latency: request-latency-in-ms
	if histogram := getHistogram("request-latency-in-ms", registry); histogram != nil {
		percentiles := histogram.Percentiles([]float64{0.5, 0.95, 0.99})
		data.RequestLatencyP50 = percentiles[0]
		data.RequestLatencyP95 = percentiles[1]
		data.RequestLatencyP99 = percentiles[2]
	} else {
		fmt.Println("Histograma 'request-latency-in-ms' no encontrado.")
	}

	// Batch Size: batch-size
	if histogram := getHistogram("batch-size", registry); histogram != nil {
		data.BatchSizeAvg = histogram.Mean()
	} else {
		fmt.Println("Histograma 'batch-size' no encontrado.")
	}

	// Records Per Request: records-per-request
	if histogram := getHistogram("records-per-request", registry); histogram != nil {
		data.RecordsPerRequestAvg = histogram.Mean()
	} else {
		fmt.Println("Histograma 'records-per-request' no encontrado.")
	}

	// Incoming Byte Rate: incoming-byte-rate
	if meter := getMeter("incoming-byte-rate", registry); meter != nil {
		data.IncomingByteRate = meter.Rate1()
	} else {
		fmt.Println("Meter 'incoming-byte-rate' no encontrado.")
	}

	// Outgoing Byte Rate: outgoing-byte-rate
	if meter := getMeter("outgoing-byte-rate", registry); meter != nil {
		data.OutgoingByteRate = meter.Rate1()
	} else {
		fmt.Println("Meter 'outgoing-byte-rate' no encontrado.")
	}

	// Requests In Flight: requests-in-flight
	if counter := getCounter("requests-in-flight", registry); counter != nil {
		data.RequestsInFlight = counter.Count()
	} else {
		fmt.Println("Counter 'requests-in-flight' no encontrado.")
	}

	// Puedes agregar más métricas según tus necesidades

	return data
}

func getMeter(name string, registry gometrics.Registry) gometrics.Meter {
	if metric := registry.Get(name); metric != nil {
		if meter, ok := metric.(gometrics.Meter); ok {
			return meter
		}
	}
	return nil
}

func getHistogram(name string, registry gometrics.Registry) gometrics.Histogram {
	if metric := registry.Get(name); metric != nil {
		if histogram, ok := metric.(gometrics.Histogram); ok {
			return histogram
		}
	}
	return nil
}

func getCounter(name string, registry gometrics.Registry) gometrics.Counter {
	if metric := registry.Get(name); metric != nil {
		if counter, ok := metric.(gometrics.Counter); ok {
			return counter
		}
	}
	return nil
}

// performTestCycle ejecuta un ciclo de prueba con una tasa de mensajes específica.
func performTestCycle(cfg *utils.Config, hostname string, producer sarama.AsyncProducer, rate int64, testDuration time.Duration, totalAckLatency *int64, messageSize int) (messagesSent int64, failures int64) {
	var (
		startTime = time.Now()
		endTime   = startTime.Add(testDuration)
	)

	ticker := time.NewTicker(time.Second / time.Duration(rate))
	defer ticker.Stop()

	for time.Now().Before(endTime) {
		select {
		case <-ticker.C:
			// Generar el mensaje de tamaño específico
			msg := &sarama.ProducerMessage{
				Topic: cfg.Kafka.Topic,
				Value: sarama.ByteEncoder(generateMessage(messageSize)),
			}
			producer.Input() <- msg
			messagesSent++
		case err := <-producer.Errors():
			failures++
			fmt.Printf("Error al enviar mensaje: %v\n", err)
		}
	}

	return messagesSent, failures
}

// TODO Implementar una función dinámica que muestre las métricas disponibles en remplazo de la función logAndRecordMetrics
func logAndRecordMetricsWithLatencyAndMetricsData(rate int64, messagesSent int64, failures int64, durationSeconds float64, metricsData MetricsData, writer *csv.Writer) {
	// Imprimir las métricas en la consola
	// fmt.Printf("Tasa: %d msg/s, Enviados: %d, Fallos: %d, Latencia Promedio ACK: %.2f ms\n", rate, messagesSent, failures, avgAckLatency)
	fmt.Printf("Métricas de Sarama:\n")
	fmt.Printf("- Tasa de envío de mensajes: %.2f msg/s\n", metricsData.SendRate)
	fmt.Printf("- Latencia de solicitud P50: %.2f ms\n", metricsData.RequestLatencyP50)
	fmt.Printf("- Latencia de solicitud P95: %.2f ms\n", metricsData.RequestLatencyP95)
	fmt.Printf("- Latencia de solicitud P99: %.2f ms\n", metricsData.RequestLatencyP99)
	fmt.Printf("- Tamaño promedio de lotes: %.2f bytes\n", metricsData.BatchSizeAvg)
	fmt.Printf("- Registros por solicitud promedio: %.2f\n", metricsData.RecordsPerRequestAvg)
	// fmt.Printf("- Ratio de compresión promedio: %.2f\n", metricsData.CompressionRatioAvg)
	fmt.Printf("- Tasa de bytes entrantes: %.2f bytes/s\n", metricsData.IncomingByteRate)
	fmt.Printf("- Tasa de bytes salientes: %.2f bytes/s\n", metricsData.OutgoingByteRate)
	fmt.Printf("- Solicitudes en vuelo: %d\n", metricsData.RequestsInFlight)

	// Escribir las métricas en el archivo CSV
	record := []string{
		strconv.FormatInt(rate, 10),
		strconv.FormatInt(messagesSent, 10),
		strconv.FormatInt(failures, 10),
		// fmt.Sprintf("%.2f", avgAckLatency),
		fmt.Sprintf("%.2f", durationSeconds),
		fmt.Sprintf("%.2f", metricsData.SendRate),
		fmt.Sprintf("%.2f", metricsData.RequestLatencyP50),
		fmt.Sprintf("%.2f", metricsData.RequestLatencyP95),
		fmt.Sprintf("%.2f", metricsData.RequestLatencyP99),
		fmt.Sprintf("%.2f", metricsData.BatchSizeAvg),
		fmt.Sprintf("%.2f", metricsData.RecordsPerRequestAvg),
		// fmt.Sprintf("%.2f", metricsData.CompressionRatioAvg),
		fmt.Sprintf("%.2f", metricsData.IncomingByteRate),
		fmt.Sprintf("%.2f", metricsData.OutgoingByteRate),
		strconv.FormatInt(metricsData.RequestsInFlight, 10),
		// Agrega más campos si has añadido más métricas
	}
	writer.Write(record)
	writer.Flush()
}

// generateMessage crea un mensaje de un tamaño específico en bytes.
func generateMessage(size int) []byte {
	message := make([]byte, size)
	for i := range message {
		// Usamos caracteres alfabéticos para generar un mensaje de prueba
		message[i] = byte(65 + i%26)
	}
	return message
}
