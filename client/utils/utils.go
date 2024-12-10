package utils

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Port     int      `yaml:"port"`
	Test     Test     `yaml:"test"`
	Rabbitmq Rabbitmq `yaml:"rabbitmq"`
	Kafka    Kafka    `yaml:"kafka"`
}

type Kafka struct {
	Version string `yaml:"version"`
	Topic   string `yaml:"topic"`
	Group   string `yaml:"group"`
	Host    string `yaml:"host"`
}

type Rabbitmq struct {
	User        string `yaml:"user"`
	Password    string `yaml:"password"`
	Queue       string `yaml:"queue"`
	Port        int    `yaml:"port"`
	StreamsPort int    `yaml:"streamsPort"`
	Host        string `yaml:"host"`
}

type Test struct {
	Type           string `yaml:"type"`
	RequestDelayMs int    `yaml:"requestDelayMs"`
}

func (c *Config) LoadConfig(path string) {
	f, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("os.ReadFile failed: %v", err)
	}

	err = yaml.Unmarshal(f, c)
	if err != nil {
		log.Fatalf("yaml.Unmarshal failed: %v", err)
	}
}

func Sleep(us int) {
	r := rand.Intn(us)
	time.Sleep(time.Duration(r) * time.Millisecond)
}

func Warning(err error, format string, args ...any) {
	if err != nil {
		log.Printf("%s: %s", fmt.Sprintf(format, args...), err)
	}
}

func Fail(err error, format string, args ...any) {
	if err != nil {
		log.Fatalf("%s: %s", fmt.Sprintf(format, args...), err)
	}
}

// BenchmarkParameters contiene los parámetros para las pruebas de benchmark
type BenchmarkParameters struct {
	Rate         int64
	Increment    int64
	TestDuration time.Duration
	MessageSize  int
}

// MetricsData contiene las métricas recopiladas durante las pruebas
type MetricsData struct {
	SendRate         float64
	MessagesSent     int64
	Failures         int64
	ElapsedTime      float64
	TotalLatency     float64
	AvgLatency       float64
	MessagesReceived int64
	// Agregar mas métricas
}

// SetupReportFile configura el archivo CSV para registrar el informe.
func SetupReportFile(fileName string, headers []string) (*os.File, *csv.Writer) {
	// Obtén el directorio de trabajo actual
	baseDir, err := os.Getwd()
	Fail(err, "No se pudo obtener el directorio de trabajo actual")

	// Define la ruta completa de la carpeta output
	outputDir := fmt.Sprintf("%s/output", baseDir)

	// Crea la carpeta output si no existe
	err = os.MkdirAll(outputDir, os.ModePerm)
	Fail(err, "No se pudo crear la carpeta output")

	// Define la ruta completa del archivo
	filePath := fmt.Sprintf("%s/%s", outputDir, fileName)

	// Crea el archivo en la ruta especificada
	outputFile, err := os.Create(filePath)
	Fail(err, "No se pudo crear el archivo de reporte")

	// Configura el escritor CSV
	writer := csv.NewWriter(outputFile)

	err = writer.Write(headers)
	Fail(err, "No se pudieron escribir los encabezados en el archivo de reporte")

	return outputFile, writer
}

// FinalizeReportFile guarda los cambios y cierra el archivo CSV.
func FinalizeReportFile(outputFile *os.File, writer *csv.Writer) {
	writer.Flush()
	outputFile.Close()
}

// GenerateMessage genera un mensaje de un tamaño específico en bytes.
func GenerateMessage(size int) []byte {
	message := make([]byte, size)
	for i := range message {
		message[i] = byte('A' + (i % 26)) // Se rellena con letras ASCII para asegurar un contenido válido.
	}
	return message
}

// GetBenchmarkParameters lee los parámetros del benchmark desde variables de entorno
// con el prefijo especificado y aplica valores predeterminados si no están establecidas.
func GetBenchmarkParameters(prefix string) BenchmarkParameters {
	var (
		rate         int64 = 100             // Tasa inicial predeterminada
		increment    int64 = 100             // Incremento predeterminado
		testDuration       = 5 * time.Second // Duración predeterminada del ciclo
		messageSize  int   = 1024            // Tamaño predeterminado del mensaje en bytes
	)

	if val, exists := os.LookupEnv(prefix + "rate"); exists {
		if parsedVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			rate = parsedVal
		} else {
			fmt.Printf("Valor inválido para %srate: %s\n", prefix, val)
		}
	}

	if val, exists := os.LookupEnv(prefix + "increment"); exists {
		if parsedVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			increment = parsedVal
		} else {
			fmt.Printf("Valor inválido para %sincrement: %s\n", prefix, val)
		}
	}

	if val, exists := os.LookupEnv(prefix + "testDuration"); exists {
		if parsedVal, err := strconv.ParseFloat(val, 64); err == nil {
			testDuration = time.Duration(parsedVal * float64(time.Second))
		} else {
			fmt.Printf("Valor inválido para %stestDuration: %s\n", prefix, val)
		}
	}

	if val, exists := os.LookupEnv(prefix + "messageSize"); exists {
		if parsedVal, err := strconv.Atoi(val); err == nil {
			messageSize = parsedVal
		} else {
			fmt.Printf("Valor inválido para %smessageSize: %s\n", prefix, val)
		}
	}

	return BenchmarkParameters{
		Rate:         rate,
		Increment:    increment,
		TestDuration: testDuration,
		MessageSize:  messageSize,
	}
}
