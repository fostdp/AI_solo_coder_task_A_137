package penetration

import (
	"math"
	"time"

	"ballistics-system/models"
)

type ArmorType string

const (
	LeatherArmor  ArmorType = "leather"
	Chainmail     ArmorType = "chainmail"
	PlateArmor    ArmorType = "plate"
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
	Type       ArrowHeadType
	TipDiameter float64
	TipArea    float64
	TipMass    float64
	Hardness   float64
	Name       string
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

func (a *Analyzer) Analyze(impactVelocity float64, arrowMass float64, armorType ArmorType, arrowHeadType ArrowHeadType, armorThickness float64) *models.PenetrationResult {
	armor := a.GetArmorConfig(armorType)
	if armorThickness > 0 {
		armor.Thickness = armorThickness
	}
	arrow := a.GetArrowConfig(arrowHeadType)

	kineticEnergy := 0.5 * arrowMass * impactVelocity * impactVelocity

	penetrationDepth := a.calculateThompsonPenetration(
		impactVelocity, arrowMass, arrow.TipArea,
		armor.Density, armor.YieldStrength, armor.Hardness, arrow.Hardness,
	)

	residualVelocity := 0.0
	if penetrationDepth > armor.Thickness {
		remainingEnergy := kineticEnergy * (1.0 - armor.Thickness/penetrationDepth)
		if remainingEnergy > 0 {
			residualVelocity = math.Sqrt(2.0 * remainingEnergy / arrowMass)
		}
		penetrationDepth = armor.Thickness
	}

	energyAbsorbed := kineticEnergy - 0.5*arrowMass*residualVelocity*residualVelocity
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
	term1 := armorThickness * armorDensity * hardnessRatio * area / mass
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
