# Proyecto de Pruebas de Rendimiento para Apache Kafka y RabbitMQ

Este proyecto permite ejecutar pruebas de rendimiento (benchmarks) para mi tema de tesis **"Analisis Comparativo de los sistemas de mensajería Apache Kafka  y RabbitMQ para el manejo de grandes volúmenes de datos"**. 

Se proporciona un entorno de pruebas basado en docker para la producción, consumo de mensajes y la recolección de métricas. También se incluyen configuraciones para la creación automática de contenedores que facilitan la ejecución de las pruebas en un entorno aislado y replicable.

```mermaid
graph TD

%% Definición de nodos
User

Producer[Producer - métricas 8081]
Consumer[Consumer - métricas 8081]

Kafka[Kafka con KRaft - puerto 9092 - 9108 - internal 9094]
RabbitMQ[RabbitMQ - puerto 5672 - 15672]

KafkaExporter[Kafka Exporter - métricas 9308]

Kafka_UI

Prometheus[Prometheus - puerto 9090]

Grafana[Grafana - puerto 3000]

%% Relaciones

Producer -- Produce mensajes 9092 --> Kafka
Consumer -- Consume mensajes 9092 --> Kafka

Producer -- Produce mensajes 5672 --> RabbitMQ
Consumer -- Consume mensajes 5672 --> RabbitMQ

%% Producer & Consumer -- Exponen métricas 8081 --> Prometheus

Kafka -- 9094 --> KafkaExporter
KafkaExporter -- Exporta métricas 9308 --> Prometheus

Kafka_UI --> Kafka
RabbitMQ -- API gestión 15672 --> Prometheus

Prometheus  --> Grafana

User -- 9000 --> Prometheus
User --> Kafka_UI
User --> RabbitMQ
User -- 3000 --> Grafana

%% Agrupación de nodos

subgraph Benchamarks_en_Go
    Producer
    Consumer
end

subgraph Sistemas_de_Mensajería
    Kafka
    RabbitMQ
end

subgraph Exportadores
    KafkaExporter
end

subgraph Embudo
    Prometheus
end

subgraph Administracion
    Kafka_UI
end

subgraph Monitoreo
    Grafana
end

subgraph Docker Network
    Sistemas_de_Mensajería
    Administracion
    Exportadores
    Embudo
    Monitoreo
end

```

**Explicación:**

- **Clientes:**
  - **Producer** y **Consumer**: Aplicaciones que producen y consumen mensajes. Exponen métricas en el puerto **8081**, que son recolectadas por Prometheus.
  - **Interacción con sistemas de mensajería:**
    - Kafka: Envío y recepción de mensajes a través del puerto **9092**.
    - RabbitMQ: Envío y recepción de mensajes a través del puerto **5672**.

- **Sistemas de Mensajería:**
  - **Kafka con KRaft**: Opera sin Zookeeper, utilizando KRaft para la coordinación. Escucha en el puerto **9092** para comunicaciones con los clientes.
  - **RabbitMQ**: Servidor de mensajería que escucha en el puerto **5672** para conexiones AMQP de clientes.

- **Exportadores:**
  - **KafkaExporter**:
    - Recopila métricas de Kafka a través del puerto JMX **9108**.
    - Expone métricas en el puerto **9308** para que Prometheus las consuma.
  - **RabbitMQExporter**:
    - Recopila métricas de RabbitMQ utilizando la API de gestión en el puerto **15672**.
    - Expone métricas en el puerto **9419** para Prometheus.

- **Monitoreo:**
  - **Prometheus**: Sistema de monitoreo que recopila métricas de:
    - **Clientes**: A través del puerto **8081**.
    - **KafkaExporter**: A través del puerto **9308**.
    - **RabbitMQExporter**: A través del puerto **9419**.

**Puertos Utilizados:**

- **Producer y Consumer (Clientes):**
  - Comunicación con **Kafka**: Puerto **9092**.
  - Comunicación con **RabbitMQ**: Puerto **5672**.
  - Exposición de métricas: Puerto **8081**.

- **Kafka con KRaft:**
  - Recepción y envío de mensajes: Puerto **9092**.
  - Métricas JMX (para KafkaExporter): Puerto **9108**.

- **RabbitMQ:**
  - Conexiones AMQP de clientes: Puerto **5672**.
  - API de gestión (para RabbitMQExporter): Puerto **15672**.

- **KafkaExporter:**
  - Recolección de métricas JMX de Kafka: Puerto **9108**.
  - Exposición de métricas para Prometheus: Puerto **9308**.

- **RabbitMQExporter:**
  - Recolección de métricas de RabbitMQ: Puerto **15672**.
  - Exposición de métricas para Prometheus: Puerto **9419**.

- **Prometheus:**
  - Recopila métricas de:
    - **Clientes** en el puerto **8081**.
    - **KafkaExporter** en el puerto **9308**.
    - **RabbitMQExporter** en el puerto **9419**.

**Notas Adicionales:**

- **Kafka con KRaft**: A partir de las versiones recientes, Kafka puede funcionar sin Zookeeper, utilizando KRaft como su propio sistema de coordinación y consenso.
- **Interacción de los Exportadores**: Los exportadores de Kafka y RabbitMQ son herramientas que extraen métricas específicas de cada sistema y las exponen en un formato que Prometheus puede recolectar.
- **Clientes Exponiendo Métricas**: Tanto el productor como el consumidor exponen sus propias métricas (como tasa de mensajes procesados, latencia, etc.) en el puerto **8081**, lo cual es útil para un monitoreo detallado de las pruebas de rendimiento.

**Resumen del Flujo de Datos:**

1. **Producción y Consumo de Mensajes:**
   - Los **clientes** producen y consumen mensajes a través de **Kafka** y **RabbitMQ** utilizando sus respectivos puertos de comunicación.
2. **Recolección de Métricas:**
   - **KafkaExporter** y **RabbitMQExporter** recopilan métricas de **Kafka** y **RabbitMQ** y las exponen para **Prometheus**.
   - Los **clientes** también exponen métricas directamente para **Prometheus**.
3. **Monitoreo:**
   - **Prometheus** recopila todas las métricas expuestas para permitir análisis y visualización del rendimiento del sistema completo.

## Tabla de Contenidos

- [Proyecto de Pruebas de Rendimiento para Apache Kafka y RabbitMQ](#proyecto-de-pruebas-de-rendimiento-para-apache-kafka-y-rabbitmq)
  - [Tabla de Contenidos](#tabla-de-contenidos)
  - [Estructura del Proyecto](#estructura-del-proyecto)
  - [Requisitos Previos](#requisitos-previos)
  - [Componentes](#componentes)
    - [Directorio `client`](#directorio-client)
    - [Directorio `monitoring`](#directorio-monitoring)
    - [Directorio `test`](#directorio-test)
  - [Métricas y Monitoreo](#métricas-y-monitoreo)
    - [Ejemplo de Métricas Recolectadas](#ejemplo-de-métricas-recolectadas)
  - [Resolución de Problemas](#resolución-de-problemas)
  - [Ejecución](#ejecución)
    - [Sistemas Involucrados](#sistemas-involucrados)
    - [Orden de Ejecución de los Sistemas](#orden-de-ejecución-de-los-sistemas)
    - [Paso 1: Iniciar docker-compose](#paso-1-iniciar-docker-compose)
    - [Paso 2: Configurar y Ejecutar el Cliente (Producer y Consumer)](#paso-2-configurar-y-ejecutar-el-cliente-producer-y-consumer)
    - [Resumen de la Ejecución en Orden](#resumen-de-la-ejecución-en-orden)
  - [Contribuciones](#contribuciones)
    - [Créditos al autor orginal](#créditos-al-autor-orginal)

---

## Estructura del Proyecto

El proyecto contiene los siguientes directorios y archivos clave:

```plaintext
.
├── client               # Código fuente del cliente que genera y consume mensajes
│   ├── config.yaml      # Configuración de cliente para Kafka y RabbitMQ
│   ├── Dockerfile       # Dockerfile para construir la imagen del cliente
│   ├── go.mod           # Definición de módulo de Go y dependencias
│   ├── go.sum           # Checksum de dependencias
│   ├── hardware         # Protobuf de dispositivos
│   ├── kafka            # Código del cliente Kafka (productor y consumidor)
│   ├── main.go          # Archivo principal de ejecución del cliente
│   ├── metrics          # Código para recolectar y exponer métricas
│   ├── rabbitmq         # Código del cliente RabbitMQ (productor y consumidor)
│   ├── rstreams         # Cliente de RabbitMQ Streams
│   └── utils            # Utilidades generales para el cliente
├── kafka.properties     # Configuración de Kafka
├── monitoring           # Archivos YAML para monitorear los clientes de Kafka y RabbitMQ
├── rabbitmq-env.conf    # Configuración de entorno de RabbitMQ
└── test                 # Archivos de configuración y prueba YAML para Kafka y RabbitMQ
```

## Requisitos Previos

1. **Docker**: Asegúrate de que Docker esté instalado y en funcionamiento.
2. **Go**: Versión 1.20 o superior para compilar y ejecutar el cliente.
3. **Protocolo de Mensajería**: Apache Kafka y RabbitMQ deben estar instalados y configurados. Estos servicios se pueden ejecutar en contenedores Docker o en tu entorno local.

## Componentes

### Directorio `client`

Contiene el código fuente del cliente que genera y consume mensajes en Kafka y RabbitMQ. Aquí están los subcomponentes principales:

- **`kafka/`**: Implementación del productor y consumidor de mensajes en Kafka.
- **`rabbitmq/`**: Implementación del productor y consumidor de mensajes en RabbitMQ.
- **`utils/`**: Funciones de utilidad.
- **entorno_pruebas/**: Incluye la receta para docker-compose y la configuración de prometheus

### Directorio `monitoring`

Contiene configuraciones para monitorear la actividad de los clientes de Kafka y RabbitMQ.

- **`kafka-client.yaml`**: Define los parámetros para el cliente de Kafka.
- **`rabbitmq-client.yaml`**: Define los parámetros para el cliente de RabbitMQ.

Ambos archivos usan Kubernetes Jobs para ejecutar y monitorear las pruebas de rendimiento.

### Directorio `test`

Este directorio contiene archivos de configuración YAML utilizados para ejecutar y parametrizar las pruebas en Kafka y RabbitMQ. Estos archivos pueden crear contenedores y configurar los tests.

- **`0-config.yaml`**: Configuración general para las pruebas.
- **`1-kafka-client.yaml`**: Configuración específica para el cliente de Kafka.
- **`2-rabbitmq-client.yaml`**: Configuración específica para el cliente de RabbitMQ.

## Métricas y Monitoreo

El sistema está diseñado para recolectar métricas que se exponen a través de un servidor Prometheus en el puerto especificado en `config.yaml`. Estas métricas se pueden usar para evaluar el rendimiento del cliente durante las pruebas.

### Ejemplo de Métricas Recolectadas

- **Latencia de Producción y Consumo**
- **Tamaño de Mensaje**
- **Errores de Conexión**

Estas métricas pueden ser visualizadas en Prometheus o Grafana.

## Resolución de Problemas

1. **Error `dial tcp: lookup broker: no such host`**: Verifica que el nombre del host `broker` esté correctamente configurado en `config.yaml` o intenta usar una dirección IP.
2. **Problemas con Docker**: Asegúrate de que Docker esté ejecutándose y que no haya conflictos de red.
3. **Compilación de Protobuf**: Si realizas cambios en `hardware/device.proto`, regenera los archivos Go usando `protoc`.

## Ejecución

Para ejecutar correctamente el proyecto, es importante seguir un orden específico para iniciar cada sistema y servicio que participa en la prueba de benchmarking. A continuación, se describe cada sistema involucrado, su propósito en la arquitectura y las instrucciones para ejecutarlos en el orden adecuado.

### Sistemas Involucrados

1. **Kafka con KRaft**: Plataforma de mensajería distribuida utilizada para pruebas de rendimiento. Funciona sin Zookeeper y escucha en el puerto `9092` para enviar y recibir mensajes.
2. **RabbitMQ**: Sistema de mensajería utilizado como alternativa a Kafka en las pruebas de rendimiento. Escucha en el puerto `5672` para comunicación de clientes y expone una interfaz de administración en el puerto `15672`.
3. **Kafka Exporter**: Herramienta para recopilar métricas de Kafka y exponerlas en el puerto `9308` en un formato que Prometheus puede recolectar.
4. **RabbitMQ Exporter**: Exportador de métricas para RabbitMQ que recopila datos del sistema y los expone en el puerto `9419` para Prometheus.
5. **Prometheus**: Sistema de monitoreo que recopila métricas de Kafka, RabbitMQ, Kafka Exporter, RabbitMQ Exporter y los propios clientes (producer y consumer) para un análisis detallado del rendimiento.

### Orden de Ejecución de los Sistemas

Sigue los pasos detallados a continuación para iniciar cada sistema.

---

### Paso 1: Iniciar docker-compose

1. Asegúrese que se encuentra en el directorio `entorno-pruebas`.
2. Ejecute el siguiente comando para iniciar la orquestación de los servicios:

  ```bash
    docker-compose up -d
  ```

### Paso 2: Configurar y Ejecutar el Cliente (Producer y Consumer)

Antes de ejecutar el cliente principal, asegúrate de que todos los sistemas anteriores estén en ejecución. Luego sigue estos pasos para iniciar el cliente:

1. **Configurar Variables de Entorno**:
   
   Asegúrate de que las siguientes variables de entorno estén configuradas según el tipo de cliente que deseas probar, si no se configuran se tomaran los siguientes valores por defecto:

    | Nombre       | Descripción                     | Valor por defecto |
    | ------------ | ------------------------------- | ----------------- |
    | rate         | Tasa inicial                    | 10000             |
    | increment    | Incremento                      | 5000               |
    | testDuration | Duración en segundos            | 2                 |
    | messageSize  | Tamaño de los mensajes en bytes | 10240             |

  ```bash
export CLIENT=kafka       # Puede ser "kafka", "rabbitmq" o "both". Valor por defecto "both"

# Configuración para Kafka
export kafka_rate=500000
export kafka_increment=500000
export kafka_testDuration=2
export kafka_messageSize=10240
   
# Configuración para RabbitMQ
export rabbitmq_rate=500000
export rabbitmq_increment=500000
export rabbitmq_testDuration=2
export rabbitmq_messageSize=10240   
  ```

2. **Ejecutar el Cliente**:

   Desde el directorio `client/`, puedes ejecutar el cliente directamente desde el código fuente si tienes Go instalado:

   ```bash
   go run main.go
   ```

---

### Resumen de la Ejecución en Orden

1. **Iniciar Kafka con KRaft** en el puerto `9092`.
2. **Iniciar RabbitMQ** en los puertos `5672` (para clientes) y `15672` (interfaz de administración).
3. **Iniciar Kafka Exporter** en el puerto `9308` para exponer métricas de Kafka.
4. **Iniciar RabbitMQ Exporter** en el puerto `9419` para exponer métricas de RabbitMQ.
5. **Iniciar Prometheus** para recolectar métricas de Kafka Exporter, RabbitMQ Exporter y los clientes en el puerto `9090`.
6. **Configurar y Ejecutar el Cliente** (Producer y Consumer) en el puerto `8081` para exponer métricas para Prometheus.

Con este flujo de ejecución y los sistemas configurados en el orden correcto, el cliente estará listo para realizar las pruebas de rendimiento en Kafka y RabbitMQ, mientras que Prometheus recopilará las métricas expuestas para análisis de rendimiento y monitoreo.

## Contribuciones

1. Haz un fork de este repositorio.
2. Crea una nueva rama (`feature/nueva-funcionalidad`).
3. Realiza los cambios y crea un pull request.

### Créditos al autor orginal

El autor original ejecutaba los test en la nube, el software fue modificado para realizar los test en local.

https://github.com/antonputra/tutorials/tree/218/lessons/218/client