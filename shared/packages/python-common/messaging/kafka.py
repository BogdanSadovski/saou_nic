"""
Kafka async producer and consumer wrappers.

Provides reliable message production and consumption with proper
connection handling, retries, and graceful shutdown.
"""

import json
import time
from dataclasses import dataclass, field
from typing import Any, AsyncGenerator, Callable, Optional

from kafka import KafkaConsumer, KafkaProducer
from kafka.errors import KafkaError, NoBrokersAvailable

from logger import get_logger

logger = get_logger(__name__, module="messaging.kafka")


@dataclass
class KafkaMessage:
    """Represents a Kafka message with key, value, and metadata."""

    topic: str
    value: Any
    key: Optional[str] = None
    headers: list[tuple[str, bytes]] = field(default_factory=list)
    partition: Optional[int] = None
    offset: Optional[int] = None
    timestamp: Optional[int] = None

    def serialize_value(self) -> bytes:
        """Serialize the value to JSON bytes."""
        if isinstance(self.value, bytes):
            return self.value
        return json.dumps(self.value, default=str).encode("utf-8")

    @classmethod
    def from_kafka_record(cls, topic: str, record: Any) -> "KafkaMessage":
        """Create a KafkaMessage from a consumed Kafka record."""
        value = record.value
        if isinstance(value, bytes):
            try:
                value = json.loads(value.decode("utf-8"))
            except (json.JSONDecodeError, UnicodeDecodeError):
                pass  # Keep as bytes if not JSON

        return cls(
            topic=topic,
            value=value,
            key=record.key.decode("utf-8") if record.key else None,
            partition=record.partition,
            offset=record.offset,
            timestamp=record.timestamp,
        )


class KafkaProducerClient:
    """
    Async-compatible Kafka producer wrapper.

    Handles connection lifecycle, message serialization,
    and retry logic for reliable message delivery.

    Example:
        >>> producer = KafkaProducerClient(bootstrap_servers=["kafka:9092"])
        >>> await producer.start()
        >>> await producer.produce("user-events", {"event": "signup", "user_id": 42})
        >>> await producer.stop()
    """

    def __init__(
        self,
        bootstrap_servers: list[str] | str | None = None,
        retries: int = 3,
        acks: str = "all",
        max_retries: int = 5,
        retry_backoff_ms: int = 100,
        **kwargs: Any,
    ):
        """
        Initialize the Kafka producer configuration.

        Args:
            bootstrap_servers: Kafka broker addresses.
            retries: Number of retries for failed sends.
            acks: Acknowledgment level ('0', '1', 'all').
            max_retries: Maximum retries on transient errors.
            retry_backoff_ms: Backoff between retries in milliseconds.
            **kwargs: Additional kafka-python producer config.
        """
        if isinstance(bootstrap_servers, str):
            bootstrap_servers = [bootstrap_servers]
        self.bootstrap_servers = bootstrap_servers or ["localhost:9092"]
        self.retries = retries
        self.acks = acks
        self.max_retries = max_retries
        self.retry_backoff_ms = retry_backoff_ms
        self._producer: Optional[KafkaProducer] = None
        self._kwargs = kwargs

    async def start(self) -> None:
        """Initialize the Kafka producer connection."""
        for attempt in range(1, self.max_retries + 1):
            try:
                self._producer = KafkaProducer(
                    bootstrap_servers=self.bootstrap_servers,
                    acks=self.acks,
                    retries=self.retries,
                    value_serializer=lambda v: (
                        json.dumps(v, default=str).encode("utf-8")
                        if not isinstance(v, bytes)
                        else v
                    ),
                    key_serializer=lambda k: (
                        k.encode("utf-8") if isinstance(k, str) else k
                    ),
                    **self._kwargs,
                )
                # Verify connectivity
                self._producer.bootstrap_connected()
                logger.info(
                    "kafka_producer_connected",
                    servers=",".join(self.bootstrap_servers),
                )
                return
            except (NoBrokersAvailable, KafkaError) as e:
                logger.warn(
                    "kafka_producer_connect_failed",
                    attempt=attempt,
                    max_retries=self.max_retries,
                    error=str(e),
                )
                if attempt == self.max_retries:
                    raise
                time.sleep(self.retry_backoff_ms / 1000 * attempt)

    async def stop(self) -> None:
        """Flush pending messages and close the producer."""
        if self._producer:
            self._producer.flush(timeout=10)
            self._producer.close(timeout=10)
            self._producer = None
            logger.info("kafka_producer_closed")

    async def produce(
        self,
        topic: str,
        value: Any,
        key: Optional[str] = None,
        headers: Optional[dict[str, str]] = None,
        partition: Optional[int] = None,
    ) -> Any:
        """
        Send a message to a Kafka topic.

        Args:
            topic: Target topic name.
            value: Message payload (auto-serialized to JSON).
            key: Optional message key for partitioning.
            headers: Optional dict of header key-value pairs.
            partition: Optional specific partition number.

        Returns:
            RecordMetadata from the send operation.

        Raises:
            RuntimeError: If the producer is not started.
            KafkaError: If the send fails after retries.
        """
        if self._producer is None:
            raise RuntimeError("Producer not started. Call start() first.")

        kafka_headers = None
        if headers:
            kafka_headers = [(k, v.encode("utf-8")) for k, v in headers.items()]

        future = self._producer.send(
            topic=topic,
            value=value,
            key=key,
            headers=kafka_headers,
            partition=partition,
        )

        try:
            record_metadata = future.get(timeout=10)
            logger.debug(
                "message_produced",
                topic=topic,
                partition=record_metadata.partition,
                offset=record_metadata.offset,
            )
            return record_metadata
        except KafkaError as e:
            logger.error(
                "message_produce_failed",
                topic=topic,
                error=str(e),
            )
            raise


class KafkaConsumerClient:
    """
    Async-compatible Kafka consumer wrapper.

    Handles subscription, message polling, offset management,
    and graceful shutdown.

    Example:
        >>> consumer = KafkaConsumerClient(
        ...     bootstrap_servers=["kafka:9092"],
        ...     group_id="scoring-service",
        ...     topics=["interview.completed"],
        ... )
        >>> await consumer.start()
        >>> async for message in consumer.consume():
        ...     process(message)
    """

    def __init__(
        self,
        bootstrap_servers: list[str] | str | None = None,
        group_id: str = "default",
        topics: list[str] | str | None = None,
        auto_offset_reset: str = "earliest",
        enable_auto_commit: bool = True,
        max_poll_records: int = 500,
        session_timeout_ms: int = 30000,
        heartbeat_interval_ms: int = 10000,
        **kwargs: Any,
    ):
        """
        Initialize the Kafka consumer configuration.

        Args:
            bootstrap_servers: Kafka broker addresses.
            group_id: Consumer group ID.
            topics: List of topics to subscribe to.
            auto_offset_reset: Where to start reading ('earliest', 'latest').
            enable_auto_commit: Auto-commit offsets after each poll.
            max_poll_records: Maximum records per poll.
            session_timeout_ms: Timeout for consumer session.
            heartbeat_interval_ms: Interval between heartbeats.
            **kwargs: Additional kafka-python consumer config.
        """
        if isinstance(bootstrap_servers, str):
            bootstrap_servers = [bootstrap_servers]
        self.bootstrap_servers = bootstrap_servers or ["localhost:9092"]
        self.group_id = group_id
        self.topics = [topics] if isinstance(topics, str) else (topics or [])
        self.auto_offset_reset = auto_offset_reset
        self.enable_auto_commit = enable_auto_commit
        self.max_poll_records = max_poll_records
        self.session_timeout_ms = session_timeout_ms
        self.heartbeat_interval_ms = heartbeat_interval_ms
        self._consumer: Optional[KafkaConsumer] = None
        self._running = False
        self._kwargs = kwargs

    async def start(self) -> None:
        """Initialize the Kafka consumer and subscribe to topics."""
        self._consumer = KafkaConsumer(
            *self.topics,
            bootstrap_servers=self.bootstrap_servers,
            group_id=self.group_id,
            auto_offset_reset=self.auto_offset_reset,
            enable_auto_commit=self.enable_auto_commit,
            max_poll_records=self.max_poll_records,
            session_timeout_ms=self.session_timeout_ms,
            heartbeat_interval_ms=self.heartbeat_interval_ms,
            value_deserializer=lambda m: (
                json.loads(m.decode("utf-8"))
                if m
                else None
            ),
            key_deserializer=lambda k: k.decode("utf-8") if k else None,
            **self._kwargs,
        )
        self._running = True
        logger.info(
            "kafka_consumer_connected",
            servers=",".join(self.bootstrap_servers),
            group_id=self.group_id,
            topics=",".join(self.topics),
        )

    async def stop(self) -> None:
        """Close the consumer and leave the group."""
        self._running = False
        if self._consumer:
            self._consumer.close()
            self._consumer = None
            logger.info("kafka_consumer_closed")

    async def consume(
        self,
        poll_timeout_ms: int = 1000,
        handler: Optional[Callable[[KafkaMessage], Any]] = None,
    ) -> AsyncGenerator[KafkaMessage, None]:
        """
        Continuously poll messages from subscribed topics.

        Args:
            poll_timeout_ms: Timeout for each poll call.
            handler: Optional callback for each message.

        Yields:
            KafkaMessage objects for each consumed record.

        Example:
            >>> async for msg in consumer.consume():
            ...     print(f"Got message: {msg.value}")
        """
        if self._consumer is None:
            raise RuntimeError("Consumer not started. Call start() first.")

        while self._running:
            try:
                records = self._consumer.poll(timeout_ms=poll_timeout_ms)

                for topic_partition, messages in records.items():
                    for record in messages:
                        message = KafkaMessage.from_kafka_record(
                            topic=record.topic, record=record
                        )
                        logger.debug(
                            "message_consumed",
                            topic=message.topic,
                            partition=message.partition,
                            offset=message.offset,
                        )

                        if handler:
                            try:
                                result = handler(message)
                                if hasattr(result, "__await__"):
                                    await result
                            except Exception as e:
                                logger.error(
                                    "message_handler_failed",
                                    topic=message.topic,
                                    error=str(e),
                                )

                        yield message

            except KafkaError as e:
                logger.error(
                    "consumer_poll_error",
                    error=str(e),
                )
                await self._consumer._client.await_ready(
                    self._consumer._client.cluster.leader_for_partition(
                        topic_partition.topic
                    ),
                    timeout=poll_timeout_ms / 1000,
                )

    async def commit(self) -> None:
        """Manually commit offsets (when auto_commit is disabled)."""
        if self._consumer:
            self._consumer.commit()
            logger.debug("offsets_committed")


# Convenience aliases
KafkaProducer = KafkaProducerClient
KafkaConsumer = KafkaConsumerClient
