package ballistic_simulator

import (
	"math"
	"time"

	"ballistics-system/config"
	"ballistics-system/models"
)

type Simulator struct {
	bow    config.BowConfig
	sim    config.SimulationConfig
	def    config.DefaultsConfig
	aero   config.AerodynamicsConfig
}

func NewSimulator(dynCfg *config.DynamicsConfig) *Simulator {
	return &Simulator{
		bow:  dynCfg.Bow,
		sim:  dynCfg.Simulation,
		def:  dynCfg.Defaults,
		aero: dynCfg.Aerodynamics,
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

func (s *Simulator) fillDefaults(params *models.SimulationParams) {
	if params.AirDensity == 0 {
		params.AirDensity = s.sim.AirDensitySea
	}
	if params.ArrowMass == 0 {
		params.ArrowMass = s.def.ArrowMass
	}
	if params.ArrowDiameter == 0 {
		params.ArrowDiameter = s.def.ArrowDiameter
	}
	if params.ArrowLength == 0 {
		params.ArrowLength = s.def.ArrowLength
	}
	if params.DragCoefficient == 0 {
		params.DragCoefficient = s.def.DragCoefficient
	}
	if params.SpinRate == 0 {
		params.SpinRate = s.def.SpinRate
	}
	if params.LaunchAngle == 0 {
		params.LaunchAngle = s.def.LaunchAngle
	}
}

func (s *Simulator) Simulate(params *models.SimulationParams) *models.SimulationResult {
	s.fillDefaults(params)

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
	liftFactor := 0.5 * s.aero.LiftCoefficient * params.AirDensity * crossArea / params.ArrowMass
	magnusFactor := 0.5 * s.aero.MagnusCoefficient * params.AirDensity * math.Pow(params.ArrowDiameter, 3) * spinRate / params.ArrowMass

	gyroStability := s.CalculateGyroscopicStability(params.SpinRate, params.InitialVelocity, params.ArrowMass, params.ArrowDiameter, params.ArrowLength)

	trajectory := make([]models.TrajectoryPoint, 0, int(s.sim.MaxSimTime/s.sim.TimeStep))

	var t float64
	for t = 0.0; t < s.sim.MaxSimTime; t += s.sim.TimeStep {
		v := math.Sqrt(vx*vx + vy*vy + vz*vz)
		if y < 0 && t > 0.05 {
			break
		}

		pitchDamping := 0.0
		if v > 0 && gyroStability > 1.0 {
			pitchDamping = s.aero.PitchDampingBase / gyroStability
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
		ay := -s.sim.Gravity - dragFactor*v*vy
		az := -dragFactor*v*vz - magnusFactor*vx

		if gyroStability >= 1.0 {
			ax += liftFactor * v * vy * pitchDamping
		}

		spinRate *= (1.0 - s.aero.SpinDecayRate*v*s.sim.TimeStep)

		vx += ax * s.sim.TimeStep
		vy += ay * s.sim.TimeStep
		vz += az * s.sim.TimeStep

		x += vx * s.sim.TimeStep
		y += vy * s.sim.TimeStep
		z += vz * s.sim.TimeStep
	}

	flightTime := t
	range_ := math.Sqrt(x*x + z*z)
	impactVelocity := math.Sqrt(vx*vx + vy*vy + vz*vz)
	kineticEnergy := 0.5 * params.ArrowMass * impactVelocity * impactVelocity
	impactSpin := spinRate
	impactGyro := s.CalculateGyroscopicStability(spinRate, impactVelocity, params.ArrowMass, params.ArrowDiameter, params.ArrowLength)

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

func (s *Simulator) CalculateGyroscopicStability(spinRate, velocity, mass, diameter, length float64) float64 {
	if length == 0 {
		length = s.def.ArrowLength
	}
	if velocity < 1.0 {
		return 10.0
	}
	axialMOI := 0.5 * mass * math.Pow(diameter/2.0, 2)
	transverseMOI := (1.0 / 12.0) * mass * (3.0*math.Pow(diameter/2.0, 2) + length*length)
	angularMomentum := axialMOI * spinRate * 2.0 * math.Pi
	aerodynamicMoment := 0.5 * s.sim.AirDensitySea * velocity * velocity * math.Pow(diameter, 2) * length * s.aero.AeroMomentCoefficient
	if aerodynamicMoment < 1e-9 {
		return 10.0
	}
	stability := (angularMomentum * angularMomentum) / (2.0 * axialMOI * transverseMOI * aerodynamicMoment)
	return math.Min(math.Max(stability, 0.1), 50.0)
}

type ReleaseState struct {
	ArrowX           float64
	ArrowV           float64
	ArmAngle         float64
	ArmAngularVel    float64
	StringTension    float64
	StringElong      float64
	PotentialEnergy  float64
	KineticEnergy    float64
	DissipatedEnergy float64
	Time             float64
}

func (s *Simulator) SimulateFullRelease(arrowMass float64) (float64, map[string]float64) {
	bow := s.bow
	state := &ReleaseState{
		ArrowX:          -bow.DrawLength,
		ArmAngle:        math.Asin(bow.DrawLength / (2.0 * bow.ArmLength)),
		StringElong:     bow.DrawLength * 0.3,
		PotentialEnergy: 0.5 * bow.PeakTension * bow.DrawLength,
	}

	armInertia := (1.0 / 3.0) * bow.ArmMass * bow.ArmLength * bow.ArmLength
	stringCrossArea := bow.StringCrossArea
	if stringCrossArea == 0 {
		stringCrossArea = 5e-5
	}
	stringStiffness := bow.StringYoungMod * stringCrossArea / bow.StringLength

	totalInitialEnergy := state.PotentialEnergy

	var t float64
	for t = 0.0; t < s.sim.ReleaseDuration; t += s.sim.ReleaseTimeStep {
		armRestoringTorque := -bow.PeakTension * bow.ArmLength * state.ArmAngle / (0.5 * math.Pi)

		nonlinearDamTq := -bow.NonlinearDamping * armInertia * state.ArmAngularVel * math.Abs(state.ArmAngularVel)

		hysteresisDamTq := -bow.HysteresisFactor * bow.PeakTension * bow.ArmLength * sign(state.ArmAngularVel)

		viscousDamTq := -bow.ViscousDamping * armInertia * state.ArmAngularVel

		internalDamTq := -bow.InternalDamping * bow.PeakTension * bow.ArmLength * state.ArmAngle / (0.5 * math.Pi)

		totalTorque := armRestoringTorque + nonlinearDamTq + hysteresisDamTq + viscousDamTq + internalDamTq
		armAngularAccel := totalTorque / armInertia

		armTipVel := state.ArmAngularVel * bow.ArmLength
		_ = armTipVel * math.Cos(state.ArmAngle)

		currentDraw := -state.ArrowX
		stringTensionForce := stringStiffness * state.StringElong

		accelOnArrow := (stringTensionForce * math.Cos(state.ArmAngle) * 2.0) / arrowMass
		viscousArrowDam := -bow.ViscousDamping * 0.1 * state.ArrowV

		totalArrowAccel := accelOnArrow + viscousArrowDam

		state.ArmAngularVel += armAngularAccel * s.sim.ReleaseTimeStep
		state.ArmAngle += state.ArmAngularVel * s.sim.ReleaseTimeStep
		state.ArrowV += totalArrowAccel * s.sim.ReleaseTimeStep
		state.ArrowX += state.ArrowV * s.sim.ReleaseTimeStep

		state.StringElong = math.Max(0, currentDraw*0.3+(armTipVel-armTipVel*math.Cos(state.ArmAngle))*s.sim.ReleaseTimeStep)

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

func (s *Simulator) CalculateDeformationStress(deformation, armLength, armThickness, modulus float64) float64 {
	strain := deformation * armThickness / (2.0 * armLength * armLength)
	stress := modulus * strain
	return stress
}

func (s *Simulator) CalculateOptimalAngle(targetDistance, velocity float64) float64 {
	g := s.sim.Gravity
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

func (s *Simulator) RunSimulationWorker(jobCh <-chan *models.SimJob, resultCh chan<- *models.SimulationResult) {
	for job := range jobCh {
		result := s.Simulate(job.Params)
		result.DeviceID = job.DeviceID
		resultCh <- result
	}
}
