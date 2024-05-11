# Distributed-Load-Testing-System

## Project Description

This project was developed as a part of the Semester 5 - Big Data course at PES University.

This project implements a distributed load-testing system utilizing Kafka for efficient communication between Orchestrator and Driver nodes. With a focus on high-concurrency and high-throughput testing, it enables seamless coordination for scalable and fault-tolerant load tests on web servers.

## Demo

## Functionality

### System Communication

The system utilizes Kafka as a single point of communication, publishing and subscribing to the following topics:

* `register`: All driver nodes register themselves to the orchestrator through this topic.
* `test_config`: Orchestrator node sends out test configurations to all driver nodes.
* `trigger`: Orchestrator node triggers the load test.
* `metrics`: Driver nodes publish their aggregate metrics during the load test.
* `heartbeat`: Driver nodes publish their heartbeats at all times.

### Load Test Types

* **Tsunami Testing:**
  * Users can set a delay interval between each request, maintained by the system.
* **Avalanche Testing:**
  * All requests are sent as soon as they are ready in first-come first-serve order.

### Observability

* The Orchestrator node tracks the number of requests sent by each driver node, updating at a maximum interval of one second.
* The frontend a dashboard with aggregated {min, max, mean, median, mode} response latency across all nodes and for each node.
* The orchestrator node stores the reports of each test and each node at the end of the test as JSON files.

### Dockerization

* The orchestrator and driver node services are Dockerized for easy deployment and management. The Docker Compose file provided can be used to spin up the required containers for the entire system.

## Setup

### Server

1. Navigate to the server directory:

   ```bash
   cd server
   ```
2. Start the server and scale the number of driver nodes (`n` is the desired number of nodes):

   ```bash
   docker-compose up --scale driver=n
   ```
3. For troubleshooting Kafka container:

   ```bash
   docker exec -it kafka1 /bin/bash
   ```

### Frontend

1. Navigate to the frontend directory:

   ```bash
   cd frontend
   ```
2. Install Dependencies:

   ```bash
   npm install
   ```
3. Start the frontend server:

   ```bash
   npm start
   ```

## Contributors

- Gayathri Manoj ([@GayathriManoj2003](https://github.com/GayathriManoj2003))
- R.S. Moumitha ([@Moumitha120104](https://github.com/Moumitha120104))
- Sai Manasa Nadimpalli ([@Mana120](https://github.com/Mana120))
- Shreya Kashyap ([@Shreya241103](https://github.com/shreya241103))

## References

* [Apache Kafka Go Client | Confluent Documentation](https://docs.confluent.io/kafka-clients/go/current/overview.html)
* [Conduktor Kafka Docker Compose](https://github.com/conduktor/kafka-stack-docker-compose/blob/master/zk-single-kafka-single.yml)
* [Connect to Apache Kafka Running in Docker | Baeldung](https://www.baeldung.com/kafka-docker-connection)
* [https://levelup.gitconnected.com/how-to-implement-concurrency-and-parallelism-in-go-83c9c453dd2](https://levelup.gitconnected.com/how-to-implement-concurrency-and-parallelism-in-go-83c9c453dd2)
