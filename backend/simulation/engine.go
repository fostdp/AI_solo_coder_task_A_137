package simulation

import (
	"math"
	"time"

	"ballistics-system/models"
)

const (
	Gravity        = 9.80665
	TimeStep       = 0.001
	MaxSimTime     = 30.0
	AirDensitySea  = 1.225
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Simulate(params *models.SimulationParams) *models.SimulationResult {
	if params.AirDensity == 0 {
		params.AirDensity = AirDensitySea
	}
	if params.ArrowMass == 0 {
		params.ArrowMass = 0.2
	}
	if params.ArrowDiameter == 0 {
		params.ArrowDiameter = 0.012
	}
	if params.DragCoefficient == 0 {
		params.DragCoefficient = 0.4
	}

	angleRad := params.LaunchAngle * math.Pi / 180.0
	azimuthRad := params.AzimuthAngle * math.Pi / 180.0

	vx := params.InitialVelocity * math.Cos(angleRad) * math.Cos(azimuthRad)
	vy := params.InitialVelocity * math.Sin(angleRad)
	vz := params.InitialVelocity * math.Cos(angleRad) * math.Sin(azimuthRad)

	x, y, z := 0.0, 0.0, 0.0
	maxHeight := 0.0

	crossArea := math.Pi * math.Pow(params.ArrowDiameter/2.0, 2)
	dragFactor := 0.5 * params.DragCoefficient * params.AirDensity * crossArea / params.ArrowMass

	trajectory := make([]models.TrajectoryPoint, 0, int(MaxSimTime/TimeStep))

	var t float64
	for t = 0.0; t < MaxSimTime; t += TimeStep {
		v := math.Sqrt(vx*vx + vy*vy + vz*vz)
		if y < 0 && t > 0.05 {
			break
		}

		point := models.TrajectoryPoint{
			Time:     t,
			X:        x,
			Y:        y,
			Z:        z,
			Vx:       vx,
			Vy:       vy,
			Vz:       vz,
			Velocity: v,
		}
		trajectory = append(trajectory, point)

		if y > maxHeight {
			maxHeight = y
		}

		ax := -dragFactor * v * vx
		ay := -Gravity - dragFactor*v*vy
		az := -dragFactor * v * vz

		vx += ax * TimeStep
		vy += ay * TimeStep
		vz += az * TimeStep

		x += vx * TimeStep
		y += vy * TimeStep
		z += vz * TimeStep
	}

	flightTime := t
	range_ := math.Sqrt(x*x + z*z)
	impactVelocity := math.Sqrt(vx*vx + vy*vy + vz*vz)
	kineticEnergy := 0.5 * params.ArrowMass * impactVelocity * impactVelocity

	return &models.SimulationResult{
		Timestamp:       time.Now(),
		InitialVelocity: params.InitialVelocity,
		LaunchAngle:     params.LaunchAngle,
		FlightTime:      flightTime,
		MaxHeight:       maxHeight,
		Range:           range_,
		ImpactVelocity:  impactVelocity,
		KineticEnergy:   kineticEnergy,
		Trajectory:      trajectory,
	}
}

func (e *Engine) SimulateBowRelease(tension float64, drawLength float64, arrowMass float64, armMass float64, efficiency float64) float64 {
	potentialEnergy := 0.5 * tension * drawLength * efficiency
	totalMass := arrowMass + 0.33*armMass
	velocity := math.Sqrt(2.0 * potentialEnergy / totalMass)
	return velocity
}

func (e *Engine) CalculateDeformationStress(deformation float64, armLength float64, armThickness float64, modulus float64) float64 {
	strain := deformation * armThickness / (2.0 * armLength * armLength)
	stress := modulus * strain
	return stress
}

func (e *Engine) CalculateOptimalAngle(targetDistance float64, velocity float64) float64 {
	g := Gravity
	v2 := velocity * velocity
	discriminant := v2*v2 - g*(g*targetDistance*targetDistance)
	if discriminant < 0 {
		return 45.0
	}
	sqrtDisc := math.Sqrt(discriminant)
	angle1 := math.Asin((v2 - sqrtDisc) / (g * targetDistance))
	angle2 := math.Asin((v2 + sqrtDisc) / (g * targetDistance))
	angle := math.Min(angle1, angle2)
	return angle * 180.0 / math.Pi
}
