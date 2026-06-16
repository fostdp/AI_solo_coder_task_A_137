package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	ch "ballistics-system/clickhouse"
	"ballistics-system/models"

	ballistic_simulator "ballistics-system/ballistic_simulator"
	penetration_analyzer "ballistics-system/penetration_analyzer"
)

type Server struct {
	engine      *gin.Engine
	store       *ch.Store
	simEngine   *ballistic_simulator.Simulator
	penAnalyzer *penetration_analyzer.Analyzer
	addr        string
}

func NewServer(addr string, store *ch.Store, simEngine *ballistic_simulator.Simulator, penAnalyzer *penetration_analyzer.Analyzer) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	s := &Server{
		engine:      r,
		store:       store,
		simEngine:   simEngine,
		penAnalyzer: penAnalyzer,
		addr:        addr,
	}

	r.Use(CORS())
	s.setupRoutes()
	return s
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func (s *Server) setupRoutes() {
	v1 := s.engine.Group("/api/v1")

	v1.GET("/health", s.health)

	v1.GET("/sensor/:device_id", s.getSensorData)
	v1.POST("/sensor", s.postSensorData)

	v1.POST("/simulate", s.simulate)
	v1.GET("/simulations", s.getSimulations)

	v1.POST("/penetrate", s.analyzePenetration)
	v1.POST("/penetrate/compare", s.compareArmors)
	v1.GET("/armors", s.getArmorTypes)
	v1.GET("/arrowheads", s.getArrowHeadTypes)
	v1.GET("/armor/:type/performance", s.getArmorPerformance)

	v1.GET("/alerts", s.getAlerts)
	v1.GET("/alerts/unacknowledged", s.getUnacknowledgedAlerts)
}

func (s *Server) Start() error {
	return s.engine.Run(s.addr)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "ok",
		"timestamp": time.Now(),
		"service":   "ballistics-system",
	})
}

func (s *Server) getSensorData(c *gin.Context) {
	deviceID := c.Param("device_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	data, err := s.store.QuerySensorData(ctx, deviceID, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": data, "count": len(data)})
}

func (s *Server) postSensorData(c *gin.Context) {
	var data models.SensorData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := s.store.InsertSensorData(ctx, &data); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"status": "ok", "data": data})
}

func (s *Server) simulate(c *gin.Context) {
	var params models.SimulationParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	result := s.simEngine.Simulate(&params)

	deviceID := c.Query("device_id")
	if deviceID == "" {
		deviceID = "api-sim"
	}
	result.DeviceID = deviceID

	armorType := c.Query("armor")
	if armorType == "" {
		armorType = "plate"
	}
	arrowType := c.Query("arrow")
	if arrowType == "" {
		arrowType = "bodkin"
	}

	penResult := s.penAnalyzer.AnalyzeWithSpin(
		result.ImpactVelocity,
		params.ArrowMass,
		params.ArrowDiameter,
		params.ArrowLength,
		result.ImpactSpinRate,
		armorType,
		arrowType,
		0,
	)
	result.ArmorType = armorType
	result.PenetrationDepth = penResult.PenetrationDepth
	result.PenetrationSuccess = penResult.Success

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if s.store != nil {
		if err := s.store.InsertSimulationResult(ctx, result); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if err := s.store.InsertArmorPerformance(ctx, s.penAnalyzer.ToArmorPerformance(penResult)); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(200, gin.H{
		"simulation":  result,
		"penetration": penResult,
	})
}

func (s *Server) getSimulations(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	results, err := s.store.QueryRecentSimulations(ctx, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": results, "count": len(results)})
}

func (s *Server) analyzePenetration(c *gin.Context) {
	var req struct {
		ImpactVelocity float64 `json:"impact_velocity" binding:"required"`
		ArrowMass      float64 `json:"arrow_mass"`
		ArrowDiameter  float64 `json:"arrow_diameter"`
		ArrowLength    float64 `json:"arrow_length"`
		SpinRate       float64 `json:"spin_rate"`
		ArmorType      string  `json:"armor_type" binding:"required"`
		ArrowHeadType  string  `json:"arrow_head_type"`
		ArmorThickness float64 `json:"armor_thickness"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.ArrowMass == 0 {
		req.ArrowMass = 0.2
	}
	if req.ArrowDiameter == 0 {
		req.ArrowDiameter = 0.012
	}
	if req.ArrowLength == 0 {
		req.ArrowLength = 1.0
	}
	if req.SpinRate == 0 {
		req.SpinRate = 25.0
	}
	if req.ArrowHeadType == "" {
		req.ArrowHeadType = "bodkin"
	}

	result := s.penAnalyzer.AnalyzeWithSpin(
		req.ImpactVelocity,
		req.ArrowMass,
		req.ArrowDiameter,
		req.ArrowLength,
		req.SpinRate,
		req.ArmorType,
		req.ArrowHeadType,
		req.ArmorThickness,
	)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if s.store != nil {
		_ = s.store.InsertArmorPerformance(ctx, s.penAnalyzer.ToArmorPerformance(result))
	}

	c.JSON(200, result)
}

func (s *Server) compareArmors(c *gin.Context) {
	var req struct {
		ImpactVelocity float64 `json:"impact_velocity" binding:"required"`
		ArrowMass      float64 `json:"arrow_mass"`
		ArrowDiameter  float64 `json:"arrow_diameter"`
		ArrowLength    float64 `json:"arrow_length"`
		SpinRate       float64 `json:"spin_rate"`
		ArrowHeadType  string  `json:"arrow_head_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.ArrowMass == 0 {
		req.ArrowMass = 0.2
	}
	if req.ArrowDiameter == 0 {
		req.ArrowDiameter = 0.012
	}
	if req.ArrowLength == 0 {
		req.ArrowLength = 1.0
	}
	if req.SpinRate == 0 {
		req.SpinRate = 25.0
	}
	if req.ArrowHeadType == "" {
		req.ArrowHeadType = "bodkin"
	}

	results := s.penAnalyzer.CompareArmorsWithSpin(
		req.ImpactVelocity,
		req.ArrowMass,
		req.ArrowDiameter,
		req.ArrowLength,
		req.SpinRate,
		req.ArrowHeadType,
	)
	c.JSON(200, gin.H{"results": results})
}

func (s *Server) getArmorTypes(c *gin.Context) {
	armors := s.penAnalyzer.ListArmorTypes()
	c.JSON(200, gin.H{"armors": armors})
}

func (s *Server) getArrowHeadTypes(c *gin.Context) {
	arrows := s.penAnalyzer.ListArrowHeadTypes()
	c.JSON(200, gin.H{"arrow_heads": arrows})
}

func (s *Server) getArmorPerformance(c *gin.Context) {
	armorType := c.Param("type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	results, err := s.store.QueryArmorPerformance(ctx, armorType, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": results, "count": len(results)})
}

func (s *Server) getAlerts(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	alerts, err := s.store.QueryAlerts(ctx, nil, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": alerts, "count": len(alerts)})
}

func (s *Server) getUnacknowledgedAlerts(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	unack := false
	alerts, err := s.store.QueryAlerts(ctx, &unack, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": alerts, "count": len(alerts)})
}

var _ = http.StatusOK
