package models

import "time"

type SensorData struct {
	DeviceID             string    `json:"device_id"`
	Timestamp            time.Time `json:"timestamp"`
	BowstringTension     float64   `json:"bowstring_tension"`
	ArmDeformation       float64   `json:"arm_deformation"`
	ArrowInitialVelocity float64   `json:"arrow_initial_velocity"`
	ArrowSpinRate        float64   `json:"arrow_spin_rate"`
	PenetrationDepth     float64   `json:"penetration_depth"`
	Temperature          float64   `json:"temperature"`
	Humidity             float64   `json:"humidity"`
}

type ValidatedSensorData struct {
	Data    *SensorData
	IsValid bool
	Errors  []string
}

type TrajectoryPoint struct {
	Time           float64 `json:"t"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Z              float64 `json:"z"`
	Vx             float64 `json:"vx"`
	Vy             float64 `json:"vy"`
	Vz             float64 `json:"vz"`
	Velocity       float64 `json:"v"`
	SpinRate       float64 `json:"spin_rate"`
	GyroStability  float64 `json:"gyro_stab"`
	AttitudeStable bool    `json:"stable"`
}

type SimulationParams struct {
	InitialVelocity float64 `json:"initial_velocity"`
	LaunchAngle     float64 `json:"launch_angle"`
	AzimuthAngle    float64 `json:"azimuth_angle"`
	ArrowMass       float64 `json:"arrow_mass"`
	ArrowDiameter   float64 `json:"arrow_diameter"`
	ArrowLength     float64 `json:"arrow_length"`
	DragCoefficient float64 `json:"drag_coefficient"`
	AirDensity      float64 `json:"air_density"`
	SpinRate        float64 `json:"spin_rate"`
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
	ImpactSpinRate     float64           `json:"impact_spin"`
	ImpactGyroStab     float64           `json:"impact_gyro"`
	Trajectory         []TrajectoryPoint `json:"trajectory"`
	ArmorType          string            `json:"armor_type"`
	PenetrationDepth   float64           `json:"penetration_depth"`
	PenetrationSuccess bool              `json:"penetration_success"`
}

type ArmorParams struct {
	Type          string  `json:"type"`
	Thickness     float64 `json:"thickness"`
	Density       float64 `json:"density"`
	YieldStrength float64 `json:"yield_strength"`
	Hardness      float64 `json:"hardness"`
	Name          string  `json:"name"`
}

type ArrowHeadParams struct {
	Type        string  `json:"type"`
	TipDiameter float64 `json:"tip_diameter"`
	TipArea     float64 `json:"tip_area"`
	TipMass     float64 `json:"tip_mass"`
	Hardness    float64 `json:"hardness"`
	Name        string  `json:"name"`
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
	ImpactSpinRate   float64 `json:"impact_spin"`
	GyroStability    float64 `json:"gyro_stab"`
	YawAngle         float64 `json:"yaw_angle"`
	EffectiveArea    float64 `json:"effective_area"`
	StabilityFactor  float64 `json:"stab_factor"`
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
	ArrowDiameter    float64   `json:"arrow_diameter"`
	ArrowLength      float64   `json:"arrow_length"`
	SpinRate         float64   `json:"spin_rate"`
	GyroStability    float64   `json:"gyro_stability"`
	YawAngle         float64   `json:"yaw_angle"`
	EffectiveArea    float64   `json:"effective_area"`
	ArrowHeadType    string    `json:"arrow_head_type"`
	PenetrationDepth float64   `json:"penetration_depth"`
	ResidualVelocity float64   `json:"residual_velocity"`
	EnergyAbsorbed   float64   `json:"energy_absorbed"`
}

type BowReleaseEnergy struct {
	DeviceID             string    `json:"device_id"`
	Timestamp            time.Time `json:"timestamp"`
	InitialPotentialEnergy float64 `json:"initial_potential_energy"`
	ArrowKE              float64   `json:"arrow_ke"`
	ArmKE                float64   `json:"arm_ke"`
	StringKE             float64   `json:"string_ke"`
	HysteresisLoss       float64   `json:"hysteresis_loss"`
	ViscousLoss          float64   `json:"viscous_loss"`
	InternalLoss         float64   `json:"internal_loss"`
	NonlinearLoss        float64   `json:"nonlinear_loss"`
	TotalDissipated      float64   `json:"total_dissipated"`
	Efficiency           float64   `json:"efficiency"`
	ReleaseTime          float64   `json:"release_time"`
	ExitVelocity         float64   `json:"exit_velocity"`
}

type SimJob struct {
	Params   *SimulationParams
	DeviceID string
}

type PenJob struct {
	ImpactVelocity float64
	ArrowMass      float64
	ArrowDiameter  float64
	ArrowLength    float64
	SpinRate       float64
	ArmorType      string
	ArrowHeadType  string
	ArmorThickness float64
	DeviceID       string
}

type PipelineResult struct {
	DeviceID      string
	SensorData    *SensorData
	SimResult     *SimulationResult
	PenResult     *PenetrationResult
	ReleaseEnergy map[string]float64
}
