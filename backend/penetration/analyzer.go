package penetration

import (
	"math"
	"time"

	"ballistics-system/models"
)

type ArmorType string

const (
	LeatherArmor ArmorType = "leather"
	Chainmail    ArmorType = "chainmail"
	PlateArmor   ArmorType = "plate"
)

type ArmorConfig struct {
	Type          ArmorType
	Thickness     float64
	Density       float64
	YieldStrength float64
	Hardness      float64
	Name          string
}

var DefaultArmors = map[ArmorType]ArmorConfig{
	LeatherArmor: {
		Type:          LeatherArmor,
		Thickness:     0.008,
		Density:       1000,
		YieldStrength: 40e6,
		Hardness:      150,
		Name:          "皮甲",
	},
	Chainmail: {
		Type:          Chainmail,
		Thickness:     0.006,
		Density:       7850,
		YieldStrength: 250e6,
		Hardness:      300,
		Name:          "锁子甲",
	},
	PlateArmor: {
		Type:          PlateArmor,
		Thickness:     0.0025,
		Density:       7850,
		YieldStrength: 500e6,
		Hardness:      450,
		Name:          "板甲",
	},
}

type ArrowHeadType string

const (
	BodkinPoint ArrowHeadType = "bodkin"
	BroadHead   ArrowHeadType = "broadhead"
	BluntPoint  ArrowHeadType = "blunt"
)

type ArrowConfig struct {
	Type        ArrowHeadType
	TipDiameter float64
	TipArea     float64
	TipMass     float64
	Hardness    float64
	Name        string
}

var DefaultArrowHeads = map[ArrowHeadType]ArrowConfig{
	BodkinPoint: {
		Type:        BodkinPoint,
		TipDiameter: 0.004,
		TipArea:     1.256e-5,
		TipMass:     0.03,
		Hardness:    550,
		Name:        "穿甲箭镞",
	},
	BroadHead: {
		Type:        BroadHead,
		TipDiameter: 0.03,
		TipArea:     7.068e-4,
		TipMass:     0.05,
		Hardness:    400,
		Name:        "宽刃箭镞",
	},
	BluntPoint: {
		Type:        BluntPoint,
		TipDiameter: 0.015,
		TipArea:     1.767e-4,
		TipMass:     0.04,
		Hardness:    300,
		Name:        "钝头箭镞",
	},
}

type GyroPenetrationParams struct {
	ImpactVelocity float64
	ArrowMass      float64
	ArrowDiameter  float64
	ArrowLength    float64
	SpinRate       float64
}

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) GetArmorConfig(armorType ArmorType) ArmorConfig {
	if config, ok := DefaultArmors[armorType]; ok {
		return config
	}
	return DefaultArmors[LeatherArmor]
}

func (a *Analyzer) GetArrowConfig(arrowType ArrowHeadType) ArrowConfig {
	if config, ok := DefaultArrowHeads[arrowType]; ok {
		return config
	}
	return DefaultArrowHeads[BodkinPoint]
}

func (a *Analyzer) calculateGyroscopicStability(spinRate, velocity, mass, diameter, length float64) float64 {
	if velocity < 1.0 {
		return 10.0
	}
	if length == 0 {
		length = 1.0
	}
	axialMOI := 0.5 * mass * math.Pow(diameter/2.0, 2)
	transverseMOI := (1.0/12.0) * mass * (3.0*math.Pow(diameter/2.0, 2) + length*length)
	angularMomentum := axialMOI * spinRate * 2.0 * math.Pi
	aerodynamicMoment := 0.5 * 1.225 * velocity * velocity * math.Pow(diameter, 2) * length * 0.01
	if aerodynamicMoment < 1e-9 {
		return 10.0
	}
	stability := (angularMomentum * angularMomentum) / (2.0 * axialMOI * transverseMOI * aerodynamicMoment)
	return math.Min(math.Max(stability, 0.1), 50.0)
}

func (a *Analyzer) calculateYawAngle(gyroStability, velocity float64) float64 {
	if gyroStability >= 4.0 {
		return 0.002
	}
	if gyroStability >= 1.5 {
		return 0.005 + 0.01*(1.5-gyroStability)/2.5
	}
	if gyroStability >= 1.0 {
		return 0.015 + 0.05*(1.0-gyroStability)/0.5
	}
	return 0.065 + 0.20*(1.0-gyroStability)
}

func (a *Analyzer) calculateEffectiveArea(baseArea float64, yawAngle float64, diameter float64, length float64) float64 {
	if length == 0 {
		length = 1.0
	}
	cosYaw := math.Cos(yawAngle)
	sinYaw := math.Abs(math.Sin(yawAngle))
	projectedArea := baseArea*cosYaw + diameter*length*sinYaw
	return projectedArea
}

func (a *Analyzer) calculateStabilityPenalty(gyroStability float64) float64 {
	if gyroStability >= 2.0 {
		return 1.0
	}
	if gyroStability >= 1.0 {
		return 0.75 + 0.25*(gyroStability-1.0)
	}
	if gyroStability >= 0.5 {
		return 0.40 + 0.35*(gyroStability-0.5)/0.5
	}
	return 0.15 + 0.25*gyroStability*2.0
}

func (a *Analyzer) calculateRotationalEnergy(mass, diameter, spinRate float64) float64 {
	axialMOI := 0.5 * mass * math.Pow(diameter/2.0, 2)
	angularVel := spinRate * 2.0 * math.Pi
	return 0.5 * axialMOI * angularVel * angularVel
}

func (a *Analyzer) Analyze(impactVelocity float64, arrowMass float64, armorType ArmorType, arrowHeadType ArrowHeadType, armorThickness float64) *models.PenetrationResult {
	return a.AnalyzeWithSpin(impactVelocity, arrowMass, 0.012, 1.0, 25.0, armorType, arrowHeadType, armorThickness)
}

func (a *Analyzer) AnalyzeWithSpin(impactVelocity, arrowMass, arrowDiameter, arrowLength, spinRate float64, armorType ArmorType, arrowHeadType ArrowHeadType, armorThickness float64) *models.PenetrationResult {
	armor := a.GetArmorConfig(armorType)
	if armorThickness > 0 {
		armor.Thickness = armorThickness
	}
	arrow := a.GetArrowConfig(arrowHeadType)

	gyroStability := a.calculateGyroscopicStability(spinRate, impactVelocity, arrowMass, arrowDiameter, arrowLength)
	yawAngle := a.calculateYawAngle(gyroStability, impactVelocity)
	effectiveArea := a.calculateEffectiveArea(arrow.TipArea, yawAngle, arrowDiameter, arrowLength)
	stabFactor := a.calculateStabilityPenalty(gyroStability)

	translationalKE := 0.5 * arrowMass * impactVelocity * impactVelocity
	rotationalKE := a.calculateRotationalEnergy(arrowMass, arrowDiameter, spinRate)
	totalEffectiveKE := translationalKE + rotationalKE*0.3

	basePenetration := a.calculateThompsonPenetration(
		impactVelocity, arrowMass, arrow.TipArea,
		armor.Density, armor.YieldStrength, armor.Hardness, arrow.Hardness,
	)

	areaRatio := arrow.TipArea / effectiveArea
	if areaRatio > 1.0 {
		areaRatio = 1.0
	}

	penetrationDepth := basePenetration * areaRatio * stabFactor

	if rotationalKE > 0 {
		rotaryBoost := 1.0 + (rotationalKE / translationalKE) * 0.15
		penetrationDepth *= rotaryBoost
	}

	residualVelocity := 0.0
	if penetrationDepth > armor.Thickness {
		remainingEnergy := totalEffectiveKE * (1.0 - armor.Thickness/penetrationDepth)
		if remainingEnergy > 0 {
			residualVelocity = math.Sqrt(2.0 * remainingEnergy / arrowMass)
		}
		penetrationDepth = armor.Thickness
	}

	energyAbsorbed := translationalKE - 0.5*arrowMass*residualVelocity*residualVelocity
	success := penetrationDepth >= armor.Thickness

	return &models.PenetrationResult{
		ArmorType:        string(armorType),
		ArmorThickness:   armor.Thickness,
		ImpactVelocity:   impactVelocity,
		ArrowMass:        arrowMass,
		ArrowHeadType:    string(arrowHeadType),
		PenetrationDepth: penetrationDepth,
		ResidualVelocity: residualVelocity,
		EnergyAbsorbed:   energyAbsorbed,
		Success:          success,
		ImpactSpinRate:   spinRate,
		GyroStability:    gyroStability,
		YawAngle:         yawAngle,
		EffectiveArea:    effectiveArea,
		StabilityFactor:  stabFactor,
	}
}

func (a *Analyzer) calculateThompsonPenetration(
	velocity, mass, area, armorDensity, yieldStrength, armorHardness, arrowHardness float64,
) float64 {
	hardnessRatio := arrowHardness / (armorHardness + arrowHardness)
	if hardnessRatio < 0.3 {
		hardnessRatio = 0.3
	}

	modifiedStrength := yieldStrength * (1.0 + 0.5*(1.0-hardnessRatio))
	term1 := (mass / area) / armorDensity
	term2 := 0.5 * armorDensity * velocity * velocity / modifiedStrength
	term3 := math.Log(1.0 + term2)

	penetration := term1 * term3 * hardnessRatio
	return penetration
}

func (a *Analyzer) calculateLanzPenetration(
	velocity, mass, area, armorDensity, yieldStrength float64,
) float64 {
	n := 0.5
	C := math.Sqrt(armorDensity / yieldStrength)
	term1 := (mass / (2.0 * n * area * armorDensity))
	term2 := math.Log(1.0 + n*C*C*velocity*velocity)
	return term1 * term2
}

func (a *Analyzer) CalculateBallisticLimit(
	arrowMass, tipArea, armorThickness, armorDensity, yieldStrength, armorHardness, arrowHardness float64,
) float64 {
	hardnessRatio := arrowHardness / (armorHardness + arrowHardness)
	if hardnessRatio < 0.3 {
		hardnessRatio = 0.3
	}

	modifiedStrength := yieldStrength * (1.0 + 0.5*(1.0-hardnessRatio))
	term1 := armorThickness * armorDensity * hardnessRatio * tipArea / arrowMass
	term2 := math.Exp(term1) - 1.0
	term3 := 2.0 * modifiedStrength / (armorDensity * term2)
	if term3 < 0 {
		term3 = 0
	}
	return math.Sqrt(term3)
}

func (a *Analyzer) ToArmorPerformance(r *models.PenetrationResult) *models.ArmorPerformance {
	return &models.ArmorPerformance{
		Timestamp:        time.Now(),
		ArmorType:        r.ArmorType,
		ArmorThickness:   r.ArmorThickness,
		ImpactVelocity:   r.ImpactVelocity,
		ArrowMass:        r.ArrowMass,
		ArrowHeadType:    r.ArrowHeadType,
		PenetrationDepth: r.PenetrationDepth,
		ResidualVelocity: r.ResidualVelocity,
		EnergyAbsorbed:   r.EnergyAbsorbed,
	}
}

func (a *Analyzer) CompareArmors(impactVelocity float64, arrowMass float64, arrowHead ArrowHeadType) map[string]*models.PenetrationResult {
	results := make(map[string]*models.PenetrationResult)
	for armorType, config := range DefaultArmors {
		result := a.Analyze(impactVelocity, arrowMass, armorType, arrowHead, 0)
		results[config.Name] = result
	}
	return results
}

func (a *Analyzer) CompareArmorsWithSpin(impactVelocity, arrowMass, arrowDiameter, arrowLength, spinRate float64, arrowHead ArrowHeadType) map[string]*models.PenetrationResult {
	results := make(map[string]*models.PenetrationResult)
	for armorType, config := range DefaultArmors {
		result := a.AnalyzeWithSpin(impactVelocity, arrowMass, arrowDiameter, arrowLength, spinRate, armorType, arrowHead, 0)
		results[config.Name] = result
	}
	return results
}
