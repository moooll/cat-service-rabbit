// Package config contains utilities for Kafka, Redis, MongoDB configuration
package config

// Config struct contains configuration strings for connection to services
type Config struct {
	// MongoURI is the connection string to mongoDB in cat-service network
	MongoURI string `env:"MONGO_URI"`

	// RabbitURI is the connection string to RabbitMQ in cat-service network
	RabbitURI string `env:"RABBIT_URI"`

	// RedisURI is the connection string to Redis in cat-service network
	RedisURI string `env:"REDIS_URI"`
}
