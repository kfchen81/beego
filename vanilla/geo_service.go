package vanilla

import (
	"context"
	"fmt"
	"github.com/kfchen81/beego"
	"math"
	"strconv"
)

const MAX_DISTANCE = -1

type GeoService struct {
	ServiceBase
}

func NewGeoService(ctx context.Context) *GeoService {
	service := new(GeoService)
	service.Ctx = ctx
	return service
}

func (this *GeoService) CalculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	radius := float64(6371000)
	rad := math.Pi / 180.0
	
	lat1 = lat1 * rad
	lng1 = lng1 * rad
	lat2 = lat2 * rad
	lng2 = lng2 * rad
	theta := lng2 - lng1
	
	distance := math.Acos(math.Sin(lat1)*math.Sin(lat2) + math.Cos(lat1)*math.Cos(lat2)*math.Cos(theta))
	
	value := distance * radius / 1000
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.3f", value), 64)
	
	return value
}

func (this *GeoService) CalculateDistanceUseStr(strLat1, strLng1, strLat2, strLng2 string) float64 {
	if strLat1 == "" || strLng1 == "" || strLat2 == "" || strLng2 == "" {
		return MAX_DISTANCE
	}
	
	lat1, err := strconv.ParseFloat(strLat1, 64)
	if err != nil {
		beego.Error(err)
		return MAX_DISTANCE
	}
	
	lng1, err := strconv.ParseFloat(strLng1, 64)
	if err != nil {
		beego.Error(err)
		return MAX_DISTANCE
	}
	
	lat2, err := strconv.ParseFloat(strLat2, 64)
	if err != nil {
		beego.Error(err)
		return MAX_DISTANCE
	}
	
	lng2, err := strconv.ParseFloat(strLng2, 64)
	if err != nil {
		beego.Error(err)
		return MAX_DISTANCE
	}
	
	return this.CalculateDistance(lat1, lng1, lat2, lng2)
}

func init() {
}
