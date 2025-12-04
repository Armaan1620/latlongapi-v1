package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Device struct {
	ID           int64     `json:"id"`
	AssetID      *int64    `json:"asset_id,omitempty"`
	SerialNumber *string   `json:"serial_number,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type LocationResponse struct {
	ShortAddress string `json:"short_address"`
	LongAddress  string `json:"long_address"`
}

func main() {
	// Load .env
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Println("DATABASE_URL not set")
	}

	ctx := context.Background()

	// Initialize DB connection pool
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Println("DB connection failed: ", err)
		return
	}
	defer pool.Close()

	app := fiber.New()

	healthCheck := func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	}
	// Health check
	app.Get("/health", healthCheck)

	// /devices?limit=10
	app.Get("/devices", func(c *fiber.Ctx) error {
		limitStr := c.Query("limit", "10")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 10
		}

		rows, err := pool.Query(ctx,
			`SELECT id, asset_id, serial_number, created_at
			 FROM devices ORDER BY id
			 LIMIT $1`,
			limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		defer rows.Close()

		var result []Device
		for rows.Next() {
			var d Device
			if err := rows.Scan(&d.ID, &d.AssetID, &d.SerialNumber, &d.CreatedAt); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			result = append(result, d)
		}

		return c.JSON(result)
	})

	// /api/v1/latlong?lat=..&long=..
	app.Get("/api/v1/latlong", func(c *fiber.Ctx) error {
		latStr := c.Query("lat")
		longStr := c.Query("long")

		if latStr == "" || longStr == "" {
			return fiber.NewError(fiber.StatusBadRequest, "lat and long required")
		}

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid lat")
		}

		lon, err := strconv.ParseFloat(longStr, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid long")
		}

		var resp LocationResponse

		err = pool.QueryRow(ctx,
			`SELECT short_address, long_address
			 FROM locations
			 ORDER BY location <-> point($1, $2)
			 LIMIT 1`,
			lon, lat,
		).Scan(&resp.ShortAddress, &resp.LongAddress)

		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "no nearby location found")
		}

		return c.JSON(resp)
	})

	app.Static("/", "./public")

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
		log.Println("PORT not set, using default port 3001")
	}
	log.Println("Server starting on port: ", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
