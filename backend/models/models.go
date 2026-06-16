package models

import "time"

type SensorData struct {
	DeviceID            string    `json:"device_id"`
	Timestamp           time.Time `json:"timestamp"`
	BowstringTension    float64   `json:"bowstring_tension"`
	ArmDeformation      float64   `json:"arm_deformation"`
	ArrowInitialVelocity float64  `json:"arrow_initial_velocity"`
	PenetrationDepth    float64   `json:"penetration_depth"`
	Temperature         float64   `json:"temperature"`
	Humidity            float64   `json:"humidity"`
}

type TrajectoryPoint struct {
	Time     float64 `json:"t"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Z        float64 `json:"z"`
	Vx       float64 `json:"vx"`
	Vy       float64 `json:"vy"`
	Vz       float64 `json:"vz"`
	Velocity float64 `json:"v"`
}

type SimulationParams struct {
	InitialVelocity float64 `json:"initial_velocity"`
	LaunchAngle     float64 `json:"launch_angle"`
	AzimuthAngle    float64 `json:"azimuth_angle"`
	ArrowMass       float64 `json:"arrow_mass"`
	ArrowDiameter   float64 `json:"arrow_diameter"`
	DragCoefficient float64 `json:"drag_coefficient"`
	AirDensity      float64 `json:"air_density"`
}

type SimulationResult struct {
	DeviceID           string            `json:"device_id"`
	Timestamp          time.Time         `json:"timestamp"`
	InitialVelocity    float64           `json:"initial_velocity"`
	LaunchAngle        float64           `json:"launch_angle"`
	FlightTime         float64           `json:"flight_time"`
	MaxHeight          float64           `json:"max_height"`
	Range              float64           `json:"range"`
	ImpactVelocity     float64           `json:"impact_velocity"`
	KineticEnergy      float64           `json:"kinetic_energy"`
	Trajectory         []TrajectoryPoint `json:"trajectory"`
	ArmorType          string            `json:"armor_type"`
	PenetrationDepth   float64           `json:"penetration_depth"`
	PenetrationSuccess bool              `json:"penetration_success"`
}

type ArmorParams struct {
	Type         string  `json:"type"`
	Thickness    float64 `json:"thickness"`
	Density      float64 `json:"density"`
	YieldStrength float64 `json:"yield_strength"`
	Hardness     float64 `json:"hardness"`
}

type PenetrationResult struct {
	ArmorType        string  `json:"armor_type"`
	ArmorThickness   float64 `json:"armor_thickness"`
	ImpactVelocity   float64 `json:"impact_velocity"`
	ArrowMass        float64 `json:"arrow_mass"`
	ArrowHeadType    string  `json:"arrow_head_type"`
	PenetrationDepth float64 `json:"penetration_depth"`
	ResidualVelocity float64 `json:"residual_velocity"`
	EnergyAbsorbed   float64 `json:"energy_absorbed"`
	Success          bool    `json:"success"`
}

type Alert struct {
	DeviceID     string    `json:"device_id"`
	Timestamp    time.Time `json:"timestamp"`
	AlertType    string    `json:"alert_type"`
	AlertLevel   string    `json:"alert_level"`
	Message      string    `json:"message"`
	SensorValue  float64   `json:"sensor_value"`
	Threshold    float64   `json:"threshold"`
	Acknowledged bool      `json:"acknowledged"`
}

type ArmorPerformance struct {
	Timestamp        time.Time `json:"timestamp"`
	ArmorType        string    `json:"armor_type"`
	ArmorThickness   float64   `json:"armor_thickness"`
	ImpactVelocity   float64   `json:"impact_velocity"`
	ArrowMass        float64   `json:"arrow_mass"`
	ArrowHeadType    string    `json:"arrow_head_type"`
	PenetrationDepth float64   `json:"penetration_depth"`
	ResidualVelocity float64   `json:"residual_velocity"`
	EnergyAbsorbed   float64   `json:"energy_absorbed"`
}
