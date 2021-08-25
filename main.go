package main

import (
	"context"
	"time"

	"github.com/moooll/cat-service-mongo/internal/config"
	"github.com/moooll/cat-service-mongo/internal/handler"
	"github.com/moooll/cat-service-mongo/internal/rabbit"
	"github.com/moooll/cat-service-mongo/internal/repository"
	rediscache "github.com/moooll/cat-service-mongo/internal/repository/rediscache"
	service "github.com/moooll/cat-service-mongo/internal/service"
	"github.com/moooll/cat-service-mongo/internal/streams"

	"github.com/caarlos0/env"
	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg := config.Config{}
	if ee := env.Parse(&cfg); ee != nil {
		log.Errorln("error parsing config: ", ee.Error())
	}

	mongoCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

	defer cancel()

	mongoClient, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Errorln("error connecting to the db ", err.Error())
	}

	defer func() {
		err = mongoClient.Disconnect(mongoCtx)
		if err != nil {
			log.Errorln("error disconnecting from the db ", err.Error())
		}
	}()

	collection := mongoClient.Database("catalog").Collection("cats2")
	dbs, err := mongoClient.ListDatabases(context.Background(), bson.M{})
	if err != nil {
		log.Errorln("error listing dbs ", err.Error())
	}

	collections, err := mongoClient.Database("catalog").ListCollectionNames(context.Background(), bson.M{})
	if err != nil {
		log.Errorln("error listing collections ", err.Error())
	}

	log.Infoln(collections)
	log.Infoln(dbs)

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURI,
		Password: "",
		DB:       0,
	})

	redisC := cache.New(&cache.Options{
		Redis:      rdb,
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	})
	ss := streams.NewStreamService(rdb)
	serv := handler.NewService(
		service.NewStorage(repository.NewCatalog(collection), rediscache.NewRedisCache(redisC, rdb)), ss)
	rWriteChan, err := rabbit.NewChan(cfg)
	if err != nil {
		log.Errorln("error connecting to RabbitMQ: ", err.Error())
	}

	defer func() {
		if errr := rWriteChan.Close(); errr != nil {
			log.Errorln("error closing RabbitMQ write chan: ", errr.Error())
		}
	}()

	rReadChan, er := rabbit.NewChan(cfg)
	if er != nil {
		log.Errorln("error connecting to RabbitMQ: ", er.Error())
	}

	defer func() {
		if eR := rReadChan.Close(); eR != nil {
			log.Errorln("error closing RabbitMQ read chan: ", eR.Error())
		}
	}()

	_, e := rabbit.NewQueue(rWriteChan)
	if e != nil {
		log.Errorln("error creating RabbitMQ queue: ", e.Error())
	}

	go func() {
		if ew := rabbit.WriteFromRedis(ss, rWriteChan); ew != nil {
			log.Errorln("error writing to RabbitMQ: ", ew.Error())
		}
	}()

	go func() {
		if eread := rabbit.Read(rReadChan); eread != nil {
			log.Errorln("error reading from RabbitMQ: ", eread.Error())
		}
	}()

	err = echoStart(serv)
	if err != nil {
		log.Errorln("could not start server ", err.Error())
	}
}

func echoStart(serv *handler.Service) error {
	e := echo.New()
	e.POST("/cats", serv.AddCat)
	e.GET("/cats", serv.GetAllCats)
	e.GET("/cats/:id", serv.GetCat)
	e.PUT("/cats", serv.UpdateCat)
	e.DELETE("/cats/:id", serv.DeleteCat)
	if err := e.Start(":8081"); err != nil {
		return err
	}
	return nil
}
