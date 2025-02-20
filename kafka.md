# Kafka

## setting
### Kafka Broker ID
Kafka Broker ID 是 Kafka 集群中 每個節點的唯一識別碼，用於區分不同的 Kafka 代理（broker）。這個 ID 在 Kafka 啟動時指定，並且在集群內部用來標識和管理數據。

🔹 Broker ID 的作用
	•	Kafka 是一個分佈式系統，可以有多個 Broker（節點）。
	•	每個 Broker 需要一個唯一的 ID 來區分不同的節點，否則 Kafka 會無法管理。
	•	當 KAFKA_BROKER_ID=1，表示這是 Kafka 集群中的第 1 個節點。

🔹 如何配置多個 Broker？

如果你要運行 多個 Kafka Broker，則 KAFKA_BROKER_ID 需要不同的數字：
```yaml
  kafka-1:
    environment:
      KAFKA_BROKER_ID: 1
  kafka-2:
    environment:
      KAFKA_BROKER_ID: 2
  kafka-3:
    environment:
      KAFKA_BROKER_ID: 3
```

### 連接zookeeper
Kafka 使用 Zookeeper 來管理集群狀態和選舉 Leader。在 Kafka 2.8 之前，Zookeeper 是必需的，而 Kafka 3.0 之後支持KRaft 模式，可以不依賴 Zookeeper。

🔹 Zookeeper 的作用
	1.	管理 Kafka Broker
	•	Zookeeper 會記錄哪些 Kafka Broker 是在線的。
	2.	管理主題（Topics）和分區（Partitions）
	•	追蹤 Kafka 各個主題的分區信息。
	3.	選舉 Kafka Leader
	•	當某個 Broker 掛掉時，Zookeeper 會自動選舉新的 Leader。

🔹 需要修改 KAFKA_ZOOKEEPER_CONNECT 嗎？

你的 Zookeeper IP 是 192.168.1.62，所以 KAFKA_ZOOKEEPER_CONNECT 可以改成：
```
KAFKA_ZOOKEEPER_CONNECT=192.168.1.62:2181
```

### 监听协议映射
 监听协议映射（KAFKA_LISTENER_SECURITY_PROTOCOL_MAP）
 這個變數用來定義 Kafka 使用的通訊協議，以及不同的 listener（監聽端口）對應的安全協議。
 ```
 KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT
 ```
這代表：
	•	PLAINTEXT 使用 PLAINTEXT 協議（無加密）。
	•	PLAINTEXT_HOST 也使用 PLAINTEXT（同樣無加密）。
	•	這意味著 Kafka 允許來自內部和外部的明文（無加密）連線。

🔹 支持的協議

Kafka 支持多種協議：
|協議|說明|
|--|--|
|PLAINTEXT|無加密傳輸|
|SSL|使用 TLS/SSL 加密|
|SASL_PLAINTEXT|使用 SASL（身份驗證）但不加密|
|SASL_SSL|使用 SASL + SSL|

如果想用加密連接，可以改為：
```
KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=PLAINTEXT:PLAINTEXT,SSL:SSL
```
這樣 Kafka 只允許 明文（內部）和加密（外部）連接。

### 广播监听地址
广播监听地址（KAFKA_ADVERTISED_LISTENERS）
KAFKA_ADVERTISED_LISTENERS 定義了 Kafka 向客戶端宣告的可用地址。Kafka 可能有內部（集群內部使用）和外部（給外部應用使用）兩種連接方式。
```
KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092,PLAINTEXT_HOST://localhost:9092
```
這代表：
	•	PLAINTEXT://kafka:9092 → Kafka 容器內部 使用 kafka:9092 來連接。
	•	PLAINTEXT_HOST://localhost:9092 → 外部客戶端 透過 localhost:9092 連接。

### 主题副本因子
主题副本因子（KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR）

在 Kafka 中，每個主題（topic）都可以有 多個副本（replication），用來提高可用性：
	•	副本因子 = 1：只有 1 份數據，Broker 掛掉數據就丟失。
	•	副本因子 = 2 或 3：Kafka 會自動在多個 Broker 上保存副本，確保高可用性。
```
KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1
```
這代表：
	•	Kafka 內部偏移量主題 只存一份數據，如果這個 Broker 掛掉，偏移量數據可能會丟失。

    如果有多個 Kafka Broker，建議設置 2 或 3：
```
KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=2
```
這樣，即使一個 Broker 掛掉，還有一份副本能恢復。

🔹 這只影響偏移量主題嗎？

是的，這個變數 只影響 Kafka 內部的偏移量主題（__consumer_offsets），不影響普通主題。
	•	如果你要設置普通主題的副本因子，應該在 Kafka CLI 或 API 中設置：
```
kafka-topics.sh --create --topic my-topic --partitions 3 --replication-factor 2 --bootstrap-server localhost:9092
```

|變數	|作用	|推薦設置|
|--|--|--|
|KAFKA_BROKER_ID	|每個 Kafka 節點的唯一 ID	多個 Broker 時，應該唯一|
|KAFKA_ZOOKEEPER_CONNECT	|連接 Zookeeper	改成 192.168.1.62:2181|
|KAFKA_LISTENER_SECURITY_PROTOCOL_MAP	|監聽協議	預設 PLAINTEXT，可用 SSL|
|KAFKA_ADVERTISED_LISTENERS	Kafka |對外宣告的連接地址	PLAINTEXT://192.168.1.63:9092|
|KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR	|偏移量主題的副本數	1（單節點），2+（高可用）|