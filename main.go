package main

import (
	"context"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mg MongoInstance

type MongoInstance struct {
	Client *mongo.Client
	DB     *mongo.Database
}

type Employee struct {
	ID     primitive.ObjectID `json:"id" bson:"_id"`
	Name   string             `json:"name"`
	Salary float64            `json:"salary"`
	Age    int                `json:"age"`
}

func ServerConn() {
	app := fiber.New()

	app.Get("/employee", func(c *fiber.Ctx) error {
		query := bson.D{}

		cursor, err := mg.DB.Collection("employee").Find(c.Context(), query)
		if err != nil {
			log.Error().Err(err).Msg("Пользователи не найдены")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Пользователи не найдены",
			})
		}

		var employee []Employee = make([]Employee, 0)

		if err := cursor.All(c.Context(), &employee); err != nil {
			log.Error().Err(err).Msg("Не удалось вывести список сотрудников")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Не удалось вывести список сотрудников",
			})
		}
		return c.JSON(employee)

	})
	app.Post("/employee", func(c *fiber.Ctx) error {
		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			log.Error().Err(err).Msg("Не удалось получить данные сотрудника")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Не удалось получить данные сотрудника",
			})
		}

		employee.ID = primitive.NewObjectID()

		insert, err := mg.DB.Collection("employee").InsertOne(c.Context(), employee)
		if err != nil {
			log.Error().Err(err).Msg("Не удалось добавить сотрудника")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Не удалось добавить сотрудника",
			})
		}

		filter := bson.D{{Key: "_id", Value: insert.InsertedID}}
		createdREcord := mg.DB.Collection("employee").FindOne(c.Context(), filter)

		createdEmployee := &Employee{}
		createdREcord.Decode(createdEmployee)

		return c.JSON(createdEmployee)
	})
	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		employeeID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			log.Error().Err(err).Msg("Не удалось получить id сотрудника")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Не удалось получить id сотрудника",
			})
		}

		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			log.Error().Err(err).Msg("Не удалось получить данные сотрудника")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Не удалось получить данные сотрудника",
			})
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		update := bson.D{
			{Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "salary", Value: employee.Salary},
					{Key: "age", Value: employee.Age},
				},
			},
		}

		err = mg.DB.Collection("employee").FindOneAndUpdate(c.Context(), query, update).Err()
		if err != nil {
			if err == mongo.ErrNoDocuments {
				log.Error().Err(err).Msg("Сотрудник не найден")
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"message": "Сотрудник не найден",
				})
			}
			log.Error().Err(err).Msg("Не удалось обновить сотрудника")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Не удалось обновить сотрудника",
			})
		}

		employee.ID = employeeID

		return c.JSON(employee)
	})
	app.Delete("/employee/:id", func(c *fiber.Ctx) error {
		id, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			log.Error().Err(err).Msg("Не удалось получить id сотрудника")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Не удалось получить id сотрудника",
			})
		}

		query := bson.D{{Key: "_id", Value: id}}
		res, err := mg.DB.Collection("employee").DeleteOne(c.Context(), &query)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				log.Error().Err(err).Msg("Сотрудник не найден")
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"message": "Сотрудник не найден",
				})
			}
			log.Error().Err(err).Msg("Не удалось удалить сотрудника")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Не удалось удалить сотрудника",
			})
		}

		return c.JSON(res)
	})

	log.Info().Msg("Запуск сервера на порту :8080")
	if err := app.Listen(":8080"); err != nil {
		log.Fatal().Err(err).Msg("Ошибка запуска сервера")
	}
}

func DbConn() error {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017/fiber_mongo")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal().Err(err).Msg("Не удалось подключиться к MongoDB")
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Ошибка подключения к MongoDB")
	}

	db := client.Database("fiber_mongo")

	mg = MongoInstance{
		Client: client,
		DB:     db,
	}

	log.Info().Msg("Подключение к MongoDB успешно!")
	return nil
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	DbConn()
	ServerConn()
}
