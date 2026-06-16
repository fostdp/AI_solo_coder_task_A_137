package simulation

import (
	"math"
	"time"

	"ballistics-system/models"
)

const (
	Gravity            = 9.80665
	TimeStep           = 0.001
	ReleaseTimeStep    = 5e-6
	MaxSimTime         = 30.0
	AirDensitySea      = 1.225
	YoungModulusWood   = 12e9
	PoissonRatioWood   = 0.35
)

type BowDynamicsParams struct {
	ArmLength         float64
	ArmThickness      float64
	ArmWidth          float64
	ArmMass           float64
	StringLength      float64
	StringMass        float64
	StringYoungMod    float64
	DrawLength        float64
	PeakTension       float64
	NonlinearDamping  float64
	HysteresisFactor  float64
	ViscousDamping    float64
	InternalDamping   float64
}

type ReleaseState struct {
	ArrowX            float64
	ArrowV            float64
	ArmAngle          float64
	ArmAngularVel     float64
	StringTension     float64
	StringElong       float64
	PotentialEnergy   float64
	KineticEnergy     float64
	DissipatedEnergy  float64
	Time              float64
}

type Engine struct {
	defaultBow *BowDynamicsParams
}

func NewEngine() *Engine {
	return &Engine{
		defaultBow: &BowDynamicsParams{
			ArmLength:        1.5,
			ArmThickness:     0.05,
			ArmWidth:         0.08,
			ArmMass:          2.5,
			StringLength:     3.2,
			StringMass:       0.05,
			StringYoungMod:   3e9,
			DrawLength:       1.2,
			PeakTension:      5000.0,
			NonlinearDamping: 0.15,
			HysteresisFactor: 0.08,
			ViscousDamping:   2.5,
			InternalDamping:  0.02,
		},
	}
}

func sign(x float64) float64 {
	if x > 0 {
		return 1.0
	} else if x < 0 {
		return -1.0
	}
	return 0.0
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
	if params.SpinRate == 0 {
		params.SpinRate = 25.0
	}

	angleRad := params.LaunchAngle * math.Pi / 180.0
	azimuthRad := params.AzimuthAngle * math.Pi / 180.0

	vx := params.InitialVelocity * math.Cos(angleRad) * math.Cos(azimuthRad)
	vy := params.InitialVelocity * math.Sin(angleRad)
	vz := params.InitialVelocity * math.Cos(angleRad) * math.Sin(azimuthRad)

	spinRate := params.SpinRate

	x, y, z := 0.0, 0.0, 0.0
	maxHeight := 0.0

	crossArea := math.Pi * math.Pow(params.ArrowDiameter/2.0, 2)
	dragFactor := 0.5 * params.DragCoefficient * params.AirDensity * crossArea / params.ArrowMass
	liftFactor := 0.5 * 0.05 * params.AirDensity * crossArea / params.ArrowMass
	magnusFactor := 0.5 * 0.001 * params.AirDensity * math.Pow(params.ArrowDiameter, 3) * spinRate / params.ArrowMass

	gyroStability := e.calculateGyroscopicStability(params.SpinRate, params.InitialVelocity, params.ArrowMass, params.ArrowDiameter)

	trajectory := make([]models.TrajectoryPoint, 0, int(MaxSimTime/TimeStep))

	var t float64
	for t = 0.0; t < MaxSimTime; t += TimeStep {
		v := math.Sqrt(vx*vx + vy*vy + vz*vz)
		if y < 0 && t > 0.05 {
			break
		}

		pitchDamping := 0.0
		if v > 0 && gyroStability > 1.0 {
			pitchDamping = 0.02 / gyroStability
		}

		point := models.TrajectoryPoint{
			Time:           t,
			X:              x,
			Y:              y,
			Z:              z,
			Vx:             vx,
			Vy:             vy,
			Vz:             vz,
			Velocity:       v,
			SpinRate:       spinRate,
			GyroStability:  gyroStability,
			AttitudeStable: gyroStability >= 1.0,
		}
		trajectory = append(trajectory, point)

		if y > maxHeight {
			maxHeight = y
		}

		ax := -dragFactor*v*vx + magnusFactor*vz
		ay := -Gravity - dragFactor*v*vy
		az := -dragFactor*v*vz - magnusFactor*vx

		if gyroStability >= 1.0 {
			ax += liftFactor * v * vy * pitchDamping
		}

		spinRate *= (1.0 - 0.0001*v*TimeStep)

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
	impactSpin := spinRate
	impactGyro := e.calculateGyroscopicStability(spinRate, impactVelocity, params.ArrowMass, params.ArrowDiameter)

	return &models.SimulationResult{
		Timestamp:       time.Now(),
		InitialVelocity: params.InitialVelocity,
		LaunchAngle:     params.LaunchAngle,
		FlightTime:      flightTime,
		MaxHeight:       maxHeight,
		Range:           range_,
		ImpactVelocity:  impactVelocity,
		KineticEnergy:   kineticEnergy,
		ImpactSpinRate:  impactSpin,
		ImpactGyroStab:  impactGyro,
		Trajectory:      trajectory,
	}
}

func (e *Engine) calculateGyroscopicStability(spinRate, velocity, mass, diameter float64) float64 {
	if velocity < 1.0 {
		return 10.0
	}
	axialMOI := 0.5 * mass * math.Pow(diameter/2.0, 2)
	transverseMOI := (1.0/12.0) * mass * (3.0*math.Pow(diameter/2.0, 2) + 1.0)
	angularMomentum := axialMOI * spinRate * 2.0 * math.Pi
	aerodynamicMoment := 0.5 * AirDensitySea * velocity * velocity * math.Pow(diameter, 3) * 0.01
	if aerodynamicMoment < 1e-9 {
		return 10.0
	}
	stability := (angularMomentum * angularMomentum) / (2.0 * axialMOI * transverseMOI * aerodynamicMoment)
	return math.Min(stability, 50.0)
}

func (e *Engine) SimulateBowRelease(tension float64, drawLength float64, arrowMass float64, armMass float64, efficiency float64) float64 {
	vel, _ := e.SimulateFullRelease(e.defaultBow, arrowMass)
	return vel
}

func (e *Engine) SimulateFullRelease(bow *BowDynamicsParams, arrowMass float64) (float64, map[string]float64) {
	state := &ReleaseState{
		ArrowX:          -bow.DrawLength,
		ArrowV:          0.0,
		ArmAngle:        math.Asin(bow.DrawLength / (2.0 * bow.ArmLength)),
		ArmAngularVel:   0.0,
		StringElong:     bow.DrawLength * 0.3,
		PotentialEnergy: 0.5 * bow.PeakTension * bow.DrawLength,
	}

	armInertia := (1.0 / 3.0) * bow.ArmMass * bow.ArmLength * bow.ArmLength
	stringCrossArea := 5e-5
	stringStiffness := bow.StringYoungMod * stringCrossArea / bow.StringLength

	totalInitialEnergy := state.PotentialEnergy

	var t float64
	const releaseDuration = 0.025
	for t = 0.0; t < releaseDuration; t += ReleaseTimeStep {
		armRestoringTorque := -bow.PeakTension * bow.ArmLength * state.ArmAngle / (0.5 * math.Pi)

		nonlinearDamTq := -bow.NonlinearDamping * armInertia * state.ArmAngularVel * math.Abs(state.ArmAngularVel)

		hysteresisDamTq := -bow.HysteresisFactor * bow.PeakTension * bow.ArmLength * sign(state.ArmAngularVel)

		viscousDamTq := -bow.ViscousDamping * armInertia * state.ArmAngularVel

		internalDamTq := -bow.InternalDamping * bow.PeakTension * bow.ArmLength * state.ArmAngle / (0.5 * math.Pi)

		totalTorque := armRestoringTorque + nonlinearDamTq + hysteresisDamTq + viscousDamTq + internalDamTq
		armAngularAccel := totalTorque / armInertia

		armTipVel := state.ArmAngularVel * bow.ArmLength
		stringDriveVel := armTipVel * math.Cos(state.ArmAngle)

		currentDraw := -state.ArrowX
		stringTensionForce := stringStiffness * state.StringElong

		accelOnArrow := (stringTensionForce * math.Cos(state.ArmAngle) * 2.0) / arrowMass
		viscousArrowDam := -bow.ViscousDamping * 0.1 * state.ArrowV

		totalArrowAccel := accelOnArrow + viscousArrowDam

		state.ArmAngularVel += armAngularAccel * ReleaseTimeStep
		state.ArmAngle += state.ArmAngularVel * ReleaseTimeStep
		state.ArrowV += totalArrowAccel * ReleaseTimeStep
		state.ArrowX += state.ArrowV * ReleaseTimeStep

		state.StringElong = math.Max(0, currentDraw*0.3+(armTipVel-stringDriveVel)*ReleaseTimeStep)

		armKE := 0.5 * armInertia * state.ArmAngularVel * state.ArmAngularVel * 3.0
		arrowKE := 0.5 * arrowMass * state.ArrowV * state.ArrowV
		stringKE := 0.5 * bow.StringMass * state.ArrowV * state.ArrowV * 0.33
		state.KineticEnergy = armKE + arrowKE + stringKE

		angleRatio := state.ArmAngle / math.Asin(bow.DrawLength/(2.0*bow.ArmLength))
		state.PotentialEnergy = 0.5 * bow.PeakTension * bow.DrawLength * angleRatio * angleRatio

		state.DissipatedEnergy = totalInitialEnergy - state.PotentialEnergy - state.KineticEnergy

		if state.ArrowX >= 0 && state.ArrowV > 0 {
			break
		}
	}

	exitVelocity := state.ArrowV
	armFinalKE := 0.5 * armInertia * state.ArmAngularVel * state.ArmAngularVel * 3.0
	arrowFinalKE := 0.5 * arrowMass * exitVelocity * exitVelocity

	energyBudget := map[string]float64{
		"initial_potential": totalInitialEnergy,
		"arrow_ke":          arrowFinalKE,
		"arm_ke":            armFinalKE,
		"dissipated":        state.DissipatedEnergy,
		"hysteresis_loss":   state.DissipatedEnergy * 0.35,
		"viscous_loss":      state.DissipatedEnergy * 0.30,
		"internal_loss":     state.DissipatedEnergy * 0.20,
		"nonlinear_loss":    state.DissipatedEnergy * 0.15,
		"efficiency":        arrowFinalKE / totalInitialEnergy,
		"release_time":      t,
	}

	return exitVelocity, energyBudget
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

func (e *Engine) GetDefaultBowParams() *BowDynamicsParams {
	return e.defaultBow
}
