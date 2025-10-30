package app

import (
	"fmt"
	"math"
	"time"
)

type Rates struct {
	Base      float64
	PerKm     float64
	PerMin    float64
	MinFare   float64 // опционально: минималка
	RoundTo   float64 // например, 10₸
}

var tariff = map[string]Rates{
	"ECONOMY": {Base: 500, PerKm: 100, PerMin: 50, MinFare: 600, RoundTo: 10},
	"PREMIUM": {Base: 800, PerKm: 120, PerMin: 60, MinFare: 900, RoundTo: 10},
	"XL":      {Base: 1000, PerKm: 150, PerMin: 75, MinFare: 1100, RoundTo: 10},
}

func roundToTenths(x float64) float64 {
	return math.Round(x*10) / 10
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // радиус Земли в км
	rad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := rad(lat2 - lat1)
	dLon := rad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	fmt.Println(lat1, "  ", lat2, "  ", lon1, "  ", lon2, "  ", R*c)
	return R * c
}

func estimateDurationMin(distanceKm, avgSpeedKmh float64) int {
	if avgSpeedKmh <= 1 {
		avgSpeedKmh = 25 // дефолт для города
	}
	minutes := distanceKm / avgSpeedKmh * 60
	if minutes < 1 {
		minutes = 1
	}
	return int(math.Ceil(minutes))
}


// ride number generator

var dailyCounter int
var lastDate string

func getDailyCounter() int {
    today := time.Now().Format("20060102")
    if today != lastDate {
        dailyCounter = 0
        lastDate = today
    }
    dailyCounter++
    return dailyCounter
}

func generateRideNumber() string {
    now := time.Now()
    return fmt.Sprintf("RIDE_%s_%03d", now.Format("20060102"), getDailyCounter())
}